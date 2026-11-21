import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { useAdminEnrol } from "../api/admin";
import { passkeysSupported } from "./passkey";

const TOKEN_RE = /^[A-Za-z0-9_-]{43}$/;

/**
 * One-time enrolment link: `/admin/enrol#<token>`. The token is in the fragment so it never reaches a server log;
 * it is read once and wiped from the address bar so it doesn't stay in history.
 */
export function AdminEnrol() {
  const enrol = useAdminEnrol();
  const navigate = useNavigate();
  // Read in the initializer (pure, so StrictMode's double call sees the same hash); wiped after mount.
  const [token] = useState(() => {
    const t = window.location.hash.slice(1);
    return TOKEN_RE.test(t) ? t : "";
  });
  useEffect(() => {
    if (window.location.hash) window.history.replaceState(null, "", window.location.pathname);
  }, []);

  return (
    <div className="card" style={{ maxWidth: 420, margin: "80px auto" }}>
      <h1 className="fs20 b">Set up your passkey</h1>
      <p className="mut mt8">
        Previa Admin uses passkeys only. Your device will ask for your fingerprint, face or screen
        lock; the passkey stays on the device (or your password manager). The link works once, for
        15 minutes.
      </p>
      {!token ? (
        <p role="alert" className="mt16">
          This link is incomplete. Open the full link you were sent, or ask a superadmin for a new
          one.
        </p>
      ) : !passkeysSupported() ? (
        <p role="alert" className="mt16">
          This browser can't use passkeys. Use a current Chrome, Safari, Edge or Firefox.
        </p>
      ) : (
        <div className="mt16" style={{ display: "grid", gap: 12 }}>
          {enrol.error && (
            <p role="alert" style={{ color: "var(--red, #c62828)" }}>
              {enrol.error.message}
            </p>
          )}
          <button
            className="btn btn-primary btn-block"
            type="button"
            disabled={enrol.isPending}
            onClick={() =>
              enrol.mutate(token, { onSuccess: () => navigate("/admin", { replace: true }) })
            }
          >
            {enrol.isPending ? "Waiting for your device…" : "Create passkey"}
          </button>
        </div>
      )}
    </div>
  );
}
