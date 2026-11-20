import createClient, { type Middleware } from "openapi-fetch";
import { clearSession, currentToken } from "../session/store";
import type { paths } from "./schema";

const BASE_URL = process.env.EXPO_PUBLIC_API_URL ?? "https://sumarg.previasoftwares.com";

const auth: Middleware = {
  onRequest({ request }) {
    const token = currentToken();
    if (token) request.headers.set("Authorization", `Bearer ${token}`);
    return request;
  },
  async onResponse({ response }) {
    // Expired or revoked session: drop it; the root layout redirects to login.
    if (response.status === 401) await clearSession();
    return response;
  },
};

export const api = createClient<paths>({ baseUrl: BASE_URL });
api.use(auth);
