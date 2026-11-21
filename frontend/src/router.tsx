import { createBrowserRouter, Navigate } from "react-router";
import { Login } from "./pages/login";
import { Placeholder } from "./pages/placeholder";
import { SetPassword } from "./pages/set-password";
import { NAV } from "./shell/nav";
import { Shell } from "./shell/shell";

/** School portal at `/`, Previa superadmin at `/admin` (lazy chunk, own shell). */
export const router = createBrowserRouter([
  { path: "/login", element: <Login /> },
  { path: "/set-password", element: <SetPassword /> },
  {
    path: "/",
    element: <Shell items={NAV} brand="School Transport" />,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      ...NAV.map((n) => ({
        path: n.to.slice(1),
        element: <Placeholder title={n.label} />,
      })),
    ],
  },
  {
    path: "/admin/*",
    lazy: async () => {
      const m = await import("./admin/routes");
      return { Component: m.AdminRoutes };
    },
  },
  { path: "*", element: <Navigate to="/" replace /> },
]);
