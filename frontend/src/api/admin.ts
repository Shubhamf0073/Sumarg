import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createPasskey, getPasskey, passkeyMessage } from "../admin/passkey";
import { SignedOut } from "./auth";
import { api } from "./client";
import type { components } from "./schema";

export type AdminMe = components["schemas"]["AdminMeBody"];

const ADMIN_ME = ["admin", "me"] as const;

/** Which Previa staff member is signed in. Errors with SignedOut on 401. */
export function useAdminMe() {
  return useQuery({
    queryKey: ADMIN_ME,
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/admin/me");
      if (response.status === 401) throw new SignedOut();
      if (error || !data) throw new Error(error?.message ?? "Could not load your account.");
      return data;
    },
    retry: false,
    staleTime: 5 * 60_000,
  });
}

/** Sign in with a passkey: options from the API → browser prompt → assertion back. */
export function useAdminLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const begin = await api.POST("/api/admin/auth/login/begin");
      if (begin.error || !begin.data)
        throw new Error(begin.error?.message ?? "Could not start sign-in.");
      let body: Record<string, unknown>;
      try {
        body = await getPasskey(begin.data);
      } catch (e) {
        throw new Error(passkeyMessage(e, "Sign-in failed. Please try again."));
      }
      const { data, error } = await api.POST("/api/admin/auth/login/finish", { body });
      if (error || !data) throw new Error(error?.message ?? "Sign-in failed. Please try again.");
      return data;
    },
    onSuccess: (me) => {
      qc.clear();
      qc.setQueryData(ADMIN_ME, me);
    },
  });
}

/** Register this device's passkey from a one-time enrolment link, which also signs in. */
export function useAdminEnrol() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (token: string) => {
      const begin = await api.POST("/api/admin/auth/enrol/begin", { body: { token } });
      if (begin.error || !begin.data)
        throw new Error(begin.error?.message ?? "Could not start enrolment.");
      let body: Record<string, unknown>;
      try {
        body = await createPasskey(begin.data);
      } catch (e) {
        throw new Error(passkeyMessage(e, "Enrolment failed. Please try again."));
      }
      const { data, error } = await api.POST("/api/admin/auth/enrol/finish", { body });
      if (error || !data) throw new Error(error?.message ?? "Enrolment failed. Please try again.");
      return data;
    },
    onSuccess: (me) => {
      qc.clear();
      qc.setQueryData(ADMIN_ME, me);
    },
  });
}

export function useAdminLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.POST("/api/admin/auth/logout");
    },
    onSettled: () => qc.clear(),
  });
}
