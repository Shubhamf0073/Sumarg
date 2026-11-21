import createClient, { type Middleware } from "openapi-fetch";
import type { paths } from "./schema";

// The school portal and Previa admin have separate sessions, each with its own CSRF cookie (docs/04 "Auth").
const CSRF_COOKIE = "__Host-sf_csrf";
const ADMIN_CSRF_COOKIE = "__Host-sf_admin_csrf";

function csrfToken(name: string): string {
  const hit = document.cookie.split("; ").find((c) => c.startsWith(`${name}=`));
  return hit ? decodeURIComponent(hit.slice(name.length + 1)) : "";
}

const csrf: Middleware = {
  onRequest({ request }) {
    if (request.method !== "GET" && request.method !== "HEAD") {
      const admin = new URL(request.url).pathname.startsWith("/api/admin/");
      request.headers.set("X-CSRF-Token", csrfToken(admin ? ADMIN_CSRF_COOKIE : CSRF_COOKIE));
    }
    return request;
  },
};

export const api = createClient<paths>({ baseUrl: "/", credentials: "same-origin" });
api.use(csrf);
