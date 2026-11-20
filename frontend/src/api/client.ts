import createClient, { type Middleware } from "openapi-fetch";
import type { paths } from "./schema";

const CSRF_COOKIE = "__Host-sf_csrf";

function csrfToken(): string {
  const hit = document.cookie.split("; ").find((c) => c.startsWith(`${CSRF_COOKIE}=`));
  return hit ? decodeURIComponent(hit.slice(CSRF_COOKIE.length + 1)) : "";
}

const csrf: Middleware = {
  onRequest({ request }) {
    if (request.method !== "GET" && request.method !== "HEAD") {
      request.headers.set("X-CSRF-Token", csrfToken());
    }
    return request;
  },
};

export const api = createClient<paths>({ baseUrl: "/", credentials: "same-origin" });
api.use(csrf);
