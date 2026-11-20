export type NavItem = { to: string; label: string };

/** Placeholder school nav; real screens arrive with /ui-port tasks. */
export const NAV: NavItem[] = [
  { to: "/dashboard", label: "Dashboard" },
  { to: "/live", label: "Live" },
  { to: "/routes", label: "Routes" },
  { to: "/students", label: "Students" },
  { to: "/fleet", label: "Fleet" },
  { to: "/reports", label: "Reports" },
];
