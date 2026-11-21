import { useState } from "react";
import { Link, useParams } from "react-router";
import {
  type LinkBody,
  type SchoolDetail,
  STAGES,
  type Stage,
  useMoveStage,
  useNewPasswordLink,
  useSchool,
  useSchools,
} from "../api/schools";
import { formatDate } from "../lib/dates";
import { rupees } from "../lib/money";

const STATUS_BADGE: Record<string, string> = {
  trial: "bd-blue",
  active: "bd-green",
  past_due: "bd-orange",
  suspended: "bd-red",
  cancelled: "bd-gray",
};

/** `/admin/schools`: the onboarding board, schools grouped by stage (onboarding/README.md). */
export function SchoolsBoard() {
  const schools = useSchools();
  return (
    <>
      <div className="row" style={{ alignItems: "center", marginBottom: 16 }}>
        <h1 className="fs20 b">Schools</h1>
        <span className="sp" style={{ flex: 1 }} />
        <Link className="btn btn-primary" to="/admin/schools/new">
          New school
        </Link>
      </div>
      {schools.error && <p role="alert">{schools.error.message}</p>}
      {schools.data?.length === 0 && (
        <p className="mut">No schools yet. Start with “New school”.</p>
      )}
      {schools.data &&
        STAGES.map((s) => {
          const rows = schools.data.filter((r) => r.onboarding_stage === s.id);
          if (rows.length === 0) return null;
          return (
            <section className="card mb12" key={s.id}>
              <div className="card-h">
                <span className="card-t">
                  {s.label} · {rows.length}
                </span>
              </div>
              <table className="tbl">
                <thead>
                  <tr>
                    <th>School</th>
                    <th>Code</th>
                    <th>Status</th>
                    <th>Go-live</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.id}>
                      <td>
                        <Link className="link" to={`/admin/schools/${r.id}`}>
                          {r.display_name}
                        </Link>
                        {r.is_demo && <span className="bd bd-purple"> demo</span>}
                      </td>
                      <td>{r.code}</td>
                      <td>
                        <span className={`bd ${STATUS_BADGE[r.status] ?? "bd-gray"}`}>
                          {r.status}
                        </span>
                      </td>
                      <td>{r.go_live_on ? formatDate(r.go_live_on) : "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </section>
          );
        })}
    </>
  );
}

/** A one-time link, shown once: copy it and hand it over on a channel the person controls. */
export function OneTimeLink({ link }: { link: LinkBody }) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="card mt12" style={{ background: "#f8fafc" }}>
      <p className="b">One-time set-password link — shown only now</p>
      <p className="mut fs12 mt4">
        Works once, until {formatDate(link.expires_at)}. Send it to the person directly (WhatsApp or
        SMS to their own mobile), never to a group.
      </p>
      <div className="row mt8" style={{ gap: 8 }}>
        <input
          className="inp"
          readOnly
          value={link.url}
          style={{ flex: 1 }}
          onFocus={(e) => e.target.select()}
        />
        <button
          className="btn btn-outline"
          type="button"
          onClick={() => {
            void navigator.clipboard?.writeText(link.url).then(() => setCopied(true));
          }}
        >
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
    </div>
  );
}

/** `/admin/schools/:id`: details, stage, logins, package. */
export function SchoolPage() {
  const id = Number(useParams().id);
  const school = useSchool(id);
  if (school.error) return <p role="alert">{school.error.message}</p>;
  if (!school.data) return null;
  return <SchoolView s={school.data} />;
}

function SchoolView({ s }: { s: SchoolDetail }) {
  const move = useMoveStage(s.id);
  const newLink = useNewPasswordLink(s.id);
  const [to, setTo] = useState<Stage>(s.onboarding_stage);
  const [goLive, setGoLive] = useState(s.go_live_on ?? "");
  const pkg = s.package as SchoolDetail["package"] | null;

  return (
    <>
      <p className="mut">
        <Link className="link" to="/admin/schools">
          Schools
        </Link>{" "}
        / {s.code}
      </p>
      <h1 className="fs20 b mt4">{s.display_name}</h1>
      <p className="mut">
        {s.legal_name} · {s.city} · {s.board ?? "board not set"} ·{" "}
        <span className="bd bd-blue">{s.status}</span>
      </p>

      <section className="card mt16">
        <div className="card-h">
          <span className="card-t">Onboarding</span>
        </div>
        <div className="card-b">
          <div className="stepper">
            {STAGES.map((st, i) => {
              const cur = STAGES.findIndex((x) => x.id === s.onboarding_stage);
              return (
                <div key={st.id} className={`step ${i < cur ? "done" : i === cur ? "cur" : ""}`}>
                  <span className="c">{i + 1}</span>
                  <span>{st.label}</span>
                </div>
              );
            })}
          </div>
          <form
            className="row mt12"
            style={{ gap: 8, alignItems: "end", flexWrap: "wrap" }}
            onSubmit={(e) => {
              e.preventDefault();
              move.mutate({
                from_stage: s.onboarding_stage,
                stage: to,
                ...(goLive && goLive !== s.go_live_on ? { go_live_on: goLive } : {}),
              });
            }}
          >
            <label>
              <span className="mut">Move to</span>
              <select className="sel" value={to} onChange={(e) => setTo(e.target.value as Stage)}>
                {STAGES.map((st) => (
                  <option key={st.id} value={st.id}>
                    {st.label}
                  </option>
                ))}
              </select>
            </label>
            <label>
              <span className="mut">Go-live date</span>
              <input
                className="inp"
                type="date"
                value={goLive}
                onChange={(e) => setGoLive(e.target.value)}
              />
            </label>
            <button className="btn btn-primary" type="submit" disabled={move.isPending}>
              Save
            </button>
          </form>
          <p className="mut fs12 mt8">
            Moving to Live with no date sets today. Parent sign-in SMS stay at 60/min until 14 days
            after go-live.
          </p>
          {move.error && <p role="alert">{move.error.message}</p>}
        </div>
      </section>

      <section className="card mt16">
        <div className="card-h">
          <span className="card-t">Owner and admin logins</span>
        </div>
        <table className="tbl">
          <tbody>
            {(s.logins ?? []).map((u) => (
              <tr key={u.id}>
                <td>{u.name}</td>
                <td>{u.role}</td>
                <td>{u.mobile}</td>
                <td>{u.password_set ? "password set" : "waiting for the link"}</td>
                <td>
                  <button
                    className="btn btn-sm btn-outline"
                    type="button"
                    disabled={newLink.isPending || !u.is_active}
                    onClick={() => newLink.mutate(u.id)}
                  >
                    New password link
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {newLink.error && <p role="alert">{newLink.error.message}</p>}
        {newLink.data && <OneTimeLink link={newLink.data} />}
      </section>

      <section className="card mt16">
        <div className="card-h">
          <span className="card-t">Campuses · package</span>
        </div>
        <div className="card-b">
          {(s.campuses ?? []).map((c) => (
            <p key={c.id}>
              <b>{c.code}</b> {c.name} · {c.lat.toFixed(5)}, {c.lon.toFixed(5)} · geofence{" "}
              {c.geofence_m} m
            </p>
          ))}
          {pkg ? (
            <p className="mt8">
              {formatDate(pkg.period_start)} – {formatDate(pkg.period_end)} ·{" "}
              {pkg.students_committed} students × {rupees(pkg.price_per_student_paise)}{" "}
              {s.price_includes_gst ? "(GST incl.)" : "+ GST"}
            </p>
          ) : (
            <p className="mut mt8">No active package.</p>
          )}
        </div>
      </section>
    </>
  );
}
