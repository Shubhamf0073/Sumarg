import createClient, { type Middleware } from "openapi-fetch";
import { installId } from "../session/install";
import { clearSession, currentToken } from "../session/store";
import type { paths } from "./schema";

const BASE_URL = process.env.EXPO_PUBLIC_API_URL ?? "https://sumarg.previasoftwares.com";

const auth: Middleware = {
  async onRequest({ request }) {
    const token = currentToken();
    if (token) request.headers.set("Authorization", `Bearer ${token}`);
    // Every app session is bound to this install (docs/05 §10).
    request.headers.set("X-Install-Id", await installId());
    return request;
  },
  async onResponse({ request, response }) {
    // Expired or revoked session: drop it; the root layout redirects to login. A 401 to a sign-in call is a wrong
    // code, not a lost session.
    if (response.status === 401 && !new URL(request.url).pathname.startsWith("/api/auth/app/")) {
      await clearSession();
    }
    return response;
  },
};

export const api = createClient<paths>({ baseUrl: BASE_URL });
api.use(auth);
