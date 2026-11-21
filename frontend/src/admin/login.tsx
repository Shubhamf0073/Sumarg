import { Navigate, useNavigate } from "react-router";
import { useAdminLogin, useAdminMe } from "../api/admin";
import { passkeysSupported } from "./passkey";

/** Previa staff sign-in: passkey only (docs/02 A24). No e-mail is typed; the passkey names the account. */
export function AdminLogin() {
  const me = useAdminMe();
  const login = useAdminLogin();
  const navigate = useNavigate();

  if (me.data) return <Navigate to="/admin" replace />;

  return (
    <div className="card" style={{ maxWidth: 380, margin: "80px auto" }}>
      <h1 className="fs20 b">Previa Admin</h1>
      <p className="mut mt8">Sign in with your passkey.</p>
      {!passkeysSupported() ? (
        <p role="alert" className="mt16">
          This browser can't use passkeys. Use a current Chrome, Safari, Edge or Firefox.
        </p>
      ) : (
        <div className="mt16" style={{ display: "grid", gap: 12 }}>
          {login.error && (
            <p role="alert" style={{ color: "var(--red, #c62828)" }}>
              {login.error.message}
            </p>
          )}
          <button
            className="btn btn-primary btn-block"
            type="button"
            disabled={login.isPending}
            onClick={() =>
              login.mutate(undefined, { onSuccess: () => navigate("/admin", { replace: true }) })
            }
          >
            {login.isPending ? "Waiting for your passkey…" : "Sign in with passkey"}
          </button>
          <p className="mut">
            New to Previa Admin, or lost your passkey? Ask a superadmin for an enrolment link.
          </p>
        </div>
      )}
    </div>
  );
}
