import { type FormEvent, useState } from "react";
import { Link } from "react-router";
import { useSetPassword } from "../api/auth";

const MIN = 10;

/** `/set-password#<tenant_id>.<token>`: a one-time link from Previa or the school (72 h). The token stays in the
 * fragment, so it is never sent in a URL or logged; the page posts it. */
export function SetPassword() {
  const [token] = useState(() => window.location.hash.slice(1));
  const set = useSetPassword();
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const mismatch = confirm !== "" && confirm !== password;

  function submit(e: FormEvent) {
    e.preventDefault();
    if (mismatch) return;
    set.mutate(
      { token, password },
      {
        onSuccess: () => {
          setPassword("");
          setConfirm("");
          window.history.replaceState(null, "", window.location.pathname);
        },
      },
    );
  }

  if (set.isSuccess) {
    return (
      <div className="card" style={{ maxWidth: 380, margin: "80px auto" }}>
        <h1 className="fs20 b">Password set</h1>
        <p className="mut mt8">
          Sign in with your school code, mobile number and the new password.
        </p>
        <Link className="btn btn-primary btn-block mt16" to="/login">
          Go to sign in
        </Link>
      </div>
    );
  }

  return (
    <div className="card" style={{ maxWidth: 380, margin: "80px auto" }}>
      <h1 className="fs20 b">Set your password</h1>
      <p className="mut mt8">At least {MIN} characters. This link works once.</p>
      {!token ? (
        <p role="alert" className="mt16">
          This link is incomplete. Open it again from the message you received.
        </p>
      ) : (
        <form onSubmit={submit} className="mt16" style={{ display: "grid", gap: 12 }}>
          <label>
            <span className="mut">New password</span>
            <input
              className="inp"
              type="password"
              autoComplete="new-password"
              required
              minLength={MIN}
              maxLength={128}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>
          <label>
            <span className="mut">Type it again</span>
            <input
              className="inp"
              type="password"
              autoComplete="new-password"
              required
              minLength={MIN}
              maxLength={128}
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
            />
          </label>
          {mismatch && <p role="alert">The two passwords are different.</p>}
          {set.error && (
            <p role="alert" style={{ color: "var(--red, #c62828)" }}>
              {set.error.message}
            </p>
          )}
          <button
            className="btn btn-primary btn-block"
            type="submit"
            disabled={set.isPending || mismatch}
          >
            {set.isPending ? "Saving…" : "Set password"}
          </button>
        </form>
      )}
    </div>
  );
}
