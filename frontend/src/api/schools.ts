import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { SignedOut } from "./auth";
import { api } from "./client";
import type { components } from "./schema";

export type SchoolRow = components["schemas"]["SchoolRow"];
export type SchoolDetail = components["schemas"]["SchoolDetail"];
export type WizardBody = components["schemas"]["CreateBody"];
export type LinkBody = components["schemas"]["LinkBody"];
export type Stage = SchoolRow["onboarding_stage"];

/** Onboarding stages in order (onboarding/README.md). */
export const STAGES: { id: Stage; label: string }[] = [
  { id: "setup", label: "Setup" },
  { id: "data", label: "Data collection" },
  { id: "survey", label: "Survey + cards" },
  { id: "install", label: "Installation" },
  { id: "training", label: "Training + invites" },
  { id: "live", label: "Live" },
];

const SCHOOLS = ["admin", "schools"] as const;

function fail(status: number, message: string | undefined, fallback: string): never {
  if (status === 401) throw new SignedOut();
  throw new Error(message ?? fallback);
}

/** Onboarding board: every school (pages of 200 followed until the last). */
export function useSchools() {
  return useQuery({
    queryKey: SCHOOLS,
    queryFn: async () => {
      const all: SchoolRow[] = [];
      let after = 0;
      for (;;) {
        const { data, error, response } = await api.GET("/api/admin/schools", {
          params: { query: { after } },
        });
        if (error || !data) fail(response.status, error?.message, "Could not load schools.");
        all.push(...(data.schools ?? []));
        if (data.next == null) return all;
        after = data.next;
      }
    },
  });
}

export function useSchool(id: number) {
  return useQuery({
    queryKey: [...SCHOOLS, id],
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/admin/schools/{id}", {
        params: { path: { id } },
      });
      if (error || !data) fail(response.status, error?.message, "Could not load the school.");
      return data;
    },
  });
}

/** The whole wizard in one save; the response carries the owner's one-time link (shown once). */
export function useCreateSchool() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: WizardBody) => {
      const { data, error, response } = await api.POST("/api/admin/schools", { body });
      if (error || !data) fail(response.status, error?.message, "Could not save the school.");
      return data;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: SCHOOLS }),
  });
}

export function useMoveStage(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { from_stage: Stage; stage: Stage; go_live_on?: string }) => {
      const { data, error, response } = await api.PATCH("/api/admin/schools/{id}/onboarding", {
        params: { path: { id } },
        body,
      });
      if (error || !data) fail(response.status, error?.message, "Could not move the school.");
      return data;
    },
    onSettled: () => qc.invalidateQueries({ queryKey: SCHOOLS }),
  });
}

export function useNewPasswordLink(schoolId: number) {
  return useMutation({
    mutationFn: async (userId: number) => {
      const { data, error, response } = await api.POST(
        "/api/admin/schools/{id}/users/{user_id}/password-link",
        { params: { path: { id: schoolId, user_id: userId } } },
      );
      if (error || !data) fail(response.status, error?.message, "Could not make a new link.");
      return data;
    },
  });
}

/** Cancel a login's unused set-password links without issuing a new one (a link sent to the wrong person). */
export function useCancelPasswordLink(schoolId: number) {
  return useMutation({
    mutationFn: async (userId: number) => {
      const { error, response } = await api.DELETE(
        "/api/admin/schools/{id}/users/{user_id}/password-link",
        { params: { path: { id: schoolId, user_id: userId } } },
      );
      if (error || !response.ok)
        fail(response.status, error?.message, "Could not cancel the link.");
    },
  });
}

/** Deactivate a school login: it can't sign in, its sessions end and its links stop working. */
export function useDeactivateLogin(schoolId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (userId: number) => {
      const { error, response } = await api.POST(
        "/api/admin/schools/{id}/users/{user_id}/deactivate",
        { params: { path: { id: schoolId, user_id: userId } } },
      );
      if (error || !response.ok)
        fail(response.status, error?.message, "Could not deactivate the login.");
    },
    onSettled: () => qc.invalidateQueries({ queryKey: SCHOOLS }),
  });
}

/** Reactivate a login with a new one-time link; the old password no longer works. */
export function useReactivateLogin(schoolId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (userId: number) => {
      const { data, error, response } = await api.POST(
        "/api/admin/schools/{id}/users/{user_id}/reactivate",
        { params: { path: { id: schoolId, user_id: userId } } },
      );
      if (error || !data) fail(response.status, error?.message, "Could not reactivate the login.");
      return data;
    },
    onSettled: () => qc.invalidateQueries({ queryKey: SCHOOLS }),
  });
}

export type Status = SchoolRow["status"];
export type PackageEdit = components["schemas"]["PackageEdit"];

/** Billing status moves /admin allows (mirrors the API; the API enforces). cancelled is final. */
export const TRANSITIONS: Record<Status, Status[]> = {
  trial: ["active", "cancelled"],
  active: ["past_due", "suspended", "cancelled"],
  past_due: ["active", "suspended", "cancelled"],
  suspended: ["active", "past_due", "cancelled"],
  cancelled: [],
};

export function useSetStatus(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: { from_status: Status; status: Status; reason?: string }) => {
      const { data, error, response } = await api.PATCH("/api/admin/schools/{id}/status", {
        params: { path: { id } },
        body,
      });
      if (error || !data) fail(response.status, error?.message, "Could not change the status.");
      return data;
    },
    onSettled: () => qc.invalidateQueries({ queryKey: SCHOOLS }),
  });
}

export function useEditPackage(id: number, packageId: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: PackageEdit) => {
      const { data, error, response } = await api.PATCH(
        "/api/admin/schools/{id}/package/{package_id}",
        { params: { path: { id, package_id: packageId } }, body },
      );
      if (error || !data) fail(response.status, error?.message, "Could not save the package.");
      return data;
    },
    onSettled: () => qc.invalidateQueries({ queryKey: SCHOOLS }),
  });
}
