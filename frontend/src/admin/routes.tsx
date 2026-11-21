import { Navigate, Route, Routes } from "react-router";
import { Placeholder } from "../pages/placeholder";
import { AdminEnrol } from "./enrol";
import { AdminLogin } from "./login";
import { SchoolWizard } from "./school-new";
import { SchoolPage, SchoolsBoard } from "./schools";
import { AdminShell } from "./shell";

const ADMIN_NAV = [
  { to: "/admin/schools", label: "Schools" },
  { to: "/admin/units", label: "Units" },
  { to: "/admin/billing", label: "Billing" },
];

/** Previa superadmin. Loaded lazily so the school portal's main chunk stays small. */
export function AdminRoutes() {
  return (
    <Routes>
      <Route path="login" element={<AdminLogin />} />
      <Route path="enrol" element={<AdminEnrol />} />
      <Route element={<AdminShell items={ADMIN_NAV} />}>
        <Route index element={<Navigate to="schools" replace />} />
        <Route path="schools" element={<SchoolsBoard />} />
        <Route path="schools/new" element={<SchoolWizard />} />
        <Route path="schools/:id" element={<SchoolPage />} />
        {ADMIN_NAV.filter((n) => n.to !== "/admin/schools").map((n) => (
          <Route
            key={n.to}
            path={n.to.replace("/admin/", "")}
            element={<Placeholder title={n.label} />}
          />
        ))}
      </Route>
    </Routes>
  );
}
