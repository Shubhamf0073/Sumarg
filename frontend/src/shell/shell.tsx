import { NavLink, Outlet } from "react-router";
import type { NavItem } from "./nav";

export function Shell({ items, brand }: { items: NavItem[]; brand: string }) {
  return (
    <div className="dk">
      <aside className="sb">
        <div className="sb-brand">
          <div>
            <div className="t1">{brand}</div>
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
        </header>
        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
