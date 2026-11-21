import { Navigate, NavLink, Outlet, useNavigate } from "react-router";
import { useAdminLogout, useAdminMe } from "../api/admin";
import { SignedOut } from "../api/auth";
import type { NavItem } from "../shell/nav";

/** The /admin frame: the Previa staff session, not a school's (signed out → /admin/login). */
export function AdminShell({ items }: { items: NavItem[] }) {
  const me = useAdminMe();
  const logout = useAdminLogout();
  const navigate = useNavigate();

  if (me.error instanceof SignedOut) return <Navigate to="/admin/login" replace />;
  if (me.error) {
    return (
      <div className="card" style={{ maxWidth: 380, margin: "80px auto" }}>
        <p>{me.error.message}</p>
      </div>
    );
  }
  if (!me.data) return null;

  return (
    <div className="dk">
      <aside className="sb">
        <div className="sb-brand">
          <div>
            <div className="t1">Previa Admin</div>
            <div className="t2">Sumarg</div>
          </div>
        </div>
        {items.map((n) => (
          <NavLink
            key={n.to}
            to={n.to}
            className={({ isActive }) => `sb-item ${isActive ? "active" : ""}`}
          >
            <span>{n.label}</span>
          </NavLink>
        ))}
      </aside>
      <div className="dk-main">
        <header className="tb">
          <span className="sp" />
          <span className="mut">
            {me.data.name} · {me.data.role}
          </span>
          <button
            className="btn btn-sm btn-outline"
            type="button"
            disabled={logout.isPending}
            onClick={() =>
              logout.mutate(undefined, {
                onSettled: () => navigate("/admin/login", { replace: true }),
              })
            }
          >
            Sign out
          </button>
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
