import { useState } from "react";
import { Link, useParams } from "react-router";
import { useAdminMe } from "../api/admin";
import {
  type LinkBody,
  type SchoolDetail,
  STAGES,
  type Stage,
  type Status,
  TRANSITIONS,
  useCancelPasswordLink,
  useDeactivateLogin,
  useEditPackage,
  useMoveStage,
  useNewPasswordLink,
  useReactivateLogin,
  useSchool,
  useSchools,
  useSetStatus,
} from "../api/schools";
import { formatDate } from "../lib/dates";
import { paiseFromRupees, rupees } from "../lib/money";

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
        <span className={`bd ${STATUS_BADGE[s.status] ?? "bd-gray"}`}>{s.status}</span>
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
            Moving to Live sets the go-live date to today unless an earlier one is set. Stages can
            move back, but a live school stays live. Parent sign-in SMS stay at 60/min until 14 days
            after go-live.
          </p>
          {move.error && <p role="alert">{move.error.message}</p>}
        </div>
      </section>

      <Logins s={s} />

      <Billing s={s} />

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

type LoginAction = "cancel" | "deactivate";

/** Owner/admin logins: new link, cancel a link sent to the wrong person, deactivate / reactivate (all audited). */
function Logins({ s }: { s: SchoolDetail }) {
  const newLink = useNewPasswordLink(s.id);
  const cancel = useCancelPasswordLink(s.id);
  const deactivate = useDeactivateLogin(s.id);
  const reactivate = useReactivateLogin(s.id);
  // Destructive actions are confirmed inline (no browser dialog): the row shows Confirm / Keep.
  const [confirm, setConfirm] = useState<{ userId: number; action: LoginAction } | null>(null);
  const [done, setDone] = useState<string | null>(null);
  const busy =
    newLink.isPending || cancel.isPending || deactivate.isPending || reactivate.isPending;
  const error = newLink.error ?? cancel.error ?? deactivate.error ?? reactivate.error;
  const link = reactivate.data ?? newLink.data;

  const run = (userId: number, action: LoginAction) => {
    setConfirm(null);
    setDone(null);
    const m = action === "cancel" ? cancel : deactivate;
    m.mutate(userId, {
      onSuccess: () =>
        setDone(
          action === "cancel"
            ? "Link cancelled. Make a new one when you have the right number."
            : "Login deactivated: signed out everywhere and its links stopped.",
        ),
    });
  };

  return (
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
              <td>
                {!u.is_active ? (
                  <span className="bd bd-gray">deactivated</span>
                ) : u.password_set ? (
                  "password set"
                ) : (
                  "waiting for the link"
                )}
              </td>
              <td>
                {confirm?.userId === u.id ? (
                  <span className="row" style={{ gap: 6 }}>
                    <span className="fs12">
                      {confirm.action === "cancel"
                        ? "Cancel the unused link?"
                        : "Deactivate? They are signed out everywhere."}
                    </span>
                    <button
                      className="btn btn-sm btn-danger"
                      type="button"
                      disabled={busy}
                      onClick={() => run(u.id, confirm.action)}
                    >
                      Confirm
                    </button>
                    <button
                      className="btn btn-sm btn-outline"
                      type="button"
                      onClick={() => setConfirm(null)}
                    >
                      Keep
                    </button>
                  </span>
                ) : u.is_active ? (
                  <span className="row" style={{ gap: 6 }}>
                    <button
                      className="btn btn-sm btn-outline"
                      type="button"
                      disabled={busy}
                      onClick={() => {
                        setDone(null);
                        reactivate.reset();
                        newLink.mutate(u.id);
                      }}
                    >
                      New password link
                    </button>
                    <button
                      className="btn btn-sm btn-outline"
                      type="button"
                      disabled={busy}
                      onClick={() => setConfirm({ userId: u.id, action: "cancel" })}
                    >
                      Cancel link
                    </button>
                    <button
                      className="btn btn-sm btn-outline"
                      type="button"
                      disabled={busy}
                      onClick={() => setConfirm({ userId: u.id, action: "deactivate" })}
                    >
                      Deactivate
                    </button>
                  </span>
                ) : (
                  <button
                    className="btn btn-sm btn-outline"
                    type="button"
                    disabled={busy}
                    onClick={() => {
                      setDone(null);
                      newLink.reset();
                      reactivate.mutate(u.id);
                    }}
                  >
                    Reactivate with a new link
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {error && <p role="alert">{error.message}</p>}
      {done && <p className="mut mt8">{done}</p>}
      {link && <OneTimeLink link={link} />}
    </section>
  );
}

const NEEDS_REASON: Status[] = ["suspended", "cancelled"];

/** Billing (billing role or superadmin; the API enforces): status changes and edits to the active package. */
function Billing({ s }: { s: SchoolDetail }) {
  const me = useAdminMe();
  const role = me.data?.role;
  if (role !== "billing" && role !== "superadmin") return null;
  const pkg = s.package as SchoolDetail["package"] | null;
  return (
    <section className="card mt16">
      <div className="card-h">
        <span className="card-t">Billing</span>
      </div>
      <div className="card-b">
        <StatusForm key={s.status} s={s} />
        {pkg && <PackageForm key={`${pkg.id}.${pkg.version}`} s={s} pkg={pkg} />}
      </div>
    </section>
  );
}

function StatusForm({ s }: { s: SchoolDetail }) {
  const set = useSetStatus(s.id);
  const next = TRANSITIONS[s.status];
  const [to, setTo] = useState<Status | "">(next[0] ?? "");
  const [reason, setReason] = useState("");
  const [typed, setTyped] = useState("");
  if (next.length === 0) {
    return <p className="mut">Cancelled is final: this school can't be reopened from here.</p>;
  }
  const needsReason = to !== "" && NEEDS_REASON.includes(to);
  const cancelling = to === "cancelled";
  const ready =
    to !== "" && (!needsReason || reason.trim() !== "") && (!cancelling || typed === s.code);
  return (
    <form
      className="row"
      style={{ gap: 8, alignItems: "end", flexWrap: "wrap" }}
      onSubmit={(e) => {
        e.preventDefault();
        if (!ready) return;
        set.mutate({
          from_status: s.status,
          status: to,
          ...(needsReason ? { reason: reason.trim() } : {}),
        });
      }}
    >
      <label>
        <span className="mut">Status</span>
        <select className="sel" value={to} onChange={(e) => setTo(e.target.value as Status)}>
          {next.map((st) => (
            <option key={st} value={st}>
              {st}
            </option>
          ))}
        </select>
      </label>
      {needsReason && (
        <label style={{ flex: 1 }}>
          <span className="mut">Reason</span>
          <input
            className="inp"
            maxLength={500}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
        </label>
      )}
      {cancelling && (
        <label>
          <span className="mut">Type {s.code} to confirm</span>
          <input
            className="inp"
            autoComplete="off"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
          />
        </label>
      )}
      <button
        className={`btn ${cancelling ? "btn-danger" : "btn-primary"}`}
        type="submit"
        disabled={!ready || set.isPending}
      >
        {cancelling ? "Cancel school" : "Change status"}
      </button>
      <p className="mut fs12" style={{ flexBasis: "100%" }}>
        {cancelling
          ? "Cancelling is final: every staff and parent login is refused from their next request, and it can't be undone here."
          : to === "suspended"
            ? "Suspended: the portal turns read-only from the next request; units keep logging."
            : "Takes effect on each login's next request."}
      </p>
      {set.error && <p role="alert">{set.error.message}</p>}
    </form>
  );
}

function PackageForm({ s, pkg }: { s: SchoolDetail; pkg: NonNullable<SchoolDetail["package"]> }) {
  const edit = useEditPackage(s.id, pkg.id);
  const [start, setStart] = useState(pkg.period_start);
  const [students, setStudents] = useState(String(pkg.students_committed));
  const [price, setPrice] = useState(String(pkg.price_per_student_paise / 100));
  const paise = paiseFromRupees(price);
  const count = /^\d{1,6}$/.test(students) ? Number(students) : null;
  return (
    <form
      className="row mt16"
      style={{ gap: 8, alignItems: "end", flexWrap: "wrap" }}
      onSubmit={(e) => {
        e.preventDefault();
        if (paise === null || count === null) return;
        edit.mutate({
          version: pkg.version,
          period_start: start,
          students_committed: count,
          price_per_student_paise: paise,
        });
      }}
    >
      <label>
        <span className="mut">Package start</span>
        <input
          className="inp"
          type="date"
          value={start}
          onChange={(e) => setStart(e.target.value)}
        />
      </label>
      <label>
        <span className="mut">Students committed</span>
        <input
          className="inp"
          inputMode="numeric"
          value={students}
          onChange={(e) => setStudents(e.target.value)}
        />
      </label>
      <label>
        <span className="mut">₹ per student (12 months)</span>
        <input
          className="inp"
          inputMode="decimal"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
        />
      </label>
      <button
        className="btn btn-primary"
        type="submit"
        disabled={paise === null || count === null || edit.isPending}
      >
        Save package
      </button>
      <p className="mut fs12" style={{ flexBasis: "100%" }}>
        {paise !== null && count !== null
          ? `${count} × ${rupees(paise)} ${s.price_includes_gst ? "(GST incl.)" : "+ GST"}; ends 12 months after the start.`
          : "Enter whole students and a ₹ amount."}
      </p>
      {edit.error && <p role="alert">{edit.error.message}</p>}
    </form>
  );
}
