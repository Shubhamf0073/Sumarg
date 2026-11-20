export type Principal = "parent" | "driver";

export type Session = {
  /** `<tenant_id>.<token>`; sent as a bearer header, never logged. */
  token: string;
  principal: Principal;
};

export type Group = "(auth)" | "(parent)" | "(driver)";

/** Group URLs are shared, so each group's landing screen has a unique file name. */
export const LANDING = {
  "(auth)": "/(auth)/login",
  "(parent)": "/(parent)/parent-home",
  "(driver)": "/(driver)/driver-home",
} as const;

/** The session's principal picks the route group; no session means the auth stack. */
export function homeGroup(session: Session | null): Group {
  if (!session) return "(auth)";
  return session.principal === "driver" ? "(driver)" : "(parent)";
}
