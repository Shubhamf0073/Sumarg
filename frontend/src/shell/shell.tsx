import { Navigate, NavLink, Outlet, useNavigate } from "react-router";
import { SignedOut, useLogout, useMe } from "../api/auth";
import type { NavItem } from "./nav";

export function Shell({ items, brand }: { items: NavItem[]; brand: string }) {
  const me = useMe();
  const logout = useLogout();
  const navigate = useNavigate();

  if (me.error instanceof SignedOut) return <Navigate to="/login" replace />;
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
            <div className="t1">{me.data.school.name || brand}</div>
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
          {me.data.read_only && <span className="mut">Read-only</span>}
          <span className="mut">{me.data.user?.name}</span>
          <button
            className="btn btn-sm btn-outline"
            type="button"
            disabled={logout.isPending}
            onClick={() =>
              logout.mutate(undefined, { onSettled: () => navigate("/login", { replace: true }) })
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
