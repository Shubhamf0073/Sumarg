import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "./client";
import type { components } from "./schema";

export type Me = components["schemas"]["MeBody"];

/** Thrown for 401: the caller is not signed in (the shell redirects to /login). */
export class SignedOut extends Error {}

const ME = ["me"] as const;

/** Who is signed in. Errors with SignedOut on 401. */
export function useMe() {
  return useQuery({
    queryKey: ME,
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/me");
      if (response.status === 401) throw new SignedOut();
      if (error || !data) throw new Error(error?.message ?? "Could not load your account.");
      return data;
    },
    retry: false,
    staleTime: 5 * 60_000,
  });
}

export type LoginInput = { school_code: string; mobile: string; password: string };

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: LoginInput) => {
      const { data, error } = await api.POST("/api/auth/login", { body });
      if (error || !data) throw new Error(error?.message ?? "Sign-in failed. Please try again.");
      return data;
    },
    onSuccess: (me) => {
      qc.clear();
      qc.setQueryData(ME, me);
      try {
        localStorage.setItem(SCHOOL_CODE_KEY, me.school.code);
      } catch {
        // storage blocked: the code just isn't prefilled next time
      }
    },
  });
}

/** Set a password from a one-time link (`<tenant_id>.<token>` from the URL fragment). */
export function useSetPassword() {
  return useMutation({
    mutationFn: async (body: { token: string; password: string }) => {
      const { error, response } = await api.POST("/api/auth/set-password", { body });
      if (!response.ok) throw new Error(error?.message ?? "Could not set the password.");
    },
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.POST("/api/auth/logout");
    },
    onSettled: () => qc.clear(),
  });
}

/** The school code is remembered on this device only to prefill the form (never used to pick a tenant). */
export const SCHOOL_CODE_KEY = "sf_school_code";

export function rememberedSchoolCode(): string {
  try {
    const v = localStorage.getItem(SCHOOL_CODE_KEY) ?? "";
    return /^[a-z0-9-]{3,30}$/.test(v) ? v : "";
  } catch {
    return "";
  }
}
