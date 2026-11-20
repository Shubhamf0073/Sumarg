import { type FormEvent, useState } from "react";
import { Navigate, useNavigate } from "react-router";
import { rememberedSchoolCode, useLogin, useMe } from "../api/auth";

/** Portal sign-in: school code + mobile + password (docs/05 §10). The API answers every wrong combination the same way. */
export function Login() {
  const me = useMe();
  const login = useLogin();
  const navigate = useNavigate();
  const [schoolCode, setSchoolCode] = useState(rememberedSchoolCode);
  const [mobile, setMobile] = useState("");
  const [password, setPassword] = useState("");

  if (me.data) return <Navigate to="/" replace />;

  function submit(e: FormEvent) {
    e.preventDefault();
    login.mutate(
      { school_code: schoolCode.trim().toLowerCase(), mobile: mobile.trim(), password },
      { onSuccess: () => navigate("/", { replace: true }) },
    );
  }

  return (
    <div className="card" style={{ maxWidth: 380, margin: "80px auto" }}>
      <h1 className="fs20 b">Sign in</h1>
      <p className="mut mt8">School transport portal</p>
      <form onSubmit={submit} className="mt16" style={{ display: "grid", gap: 12 }}>
        <label>
          <span className="mut">School code</span>
          <input
            className="inp"
            name="school_code"
            autoComplete="organization"
            autoCapitalize="none"
            required
            pattern="[A-Za-z0-9\-]{3,30}"
            value={schoolCode}
            onChange={(e) => setSchoolCode(e.target.value)}
          />
        </label>
        <label>
          <span className="mut">Mobile number</span>
          <input
            className="inp"
            name="mobile"
            type="tel"
            inputMode="numeric"
            autoComplete="username"
            required
            pattern="[6-9][0-9]{9}"
            maxLength={10}
            value={mobile}
            onChange={(e) => setMobile(e.target.value)}
          />
        </label>
        <label>
          <span className="mut">Password</span>
          <input
            className="inp"
            name="password"
            type="password"
            autoComplete="current-password"
            required
            maxLength={128}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        {login.error && (
          <p role="alert" style={{ color: "var(--red, #c62828)" }}>
            {login.error.message}
          </p>
        )}
        <button className="btn btn-primary btn-block" type="submit" disabled={login.isPending}>
          {login.isPending ? "Signing in…" : "Sign in"}
        </button>
      </form>
    </div>
  );
}
