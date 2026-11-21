import { type FormEvent, type ReactNode, useState } from "react";
import { Link } from "react-router";
import { useCreateSchool, type WizardBody } from "../api/schools";
import { rupees } from "../lib/money";
import { OneTimeLink } from "./schools";

type Campus = WizardBody["campuses"] extends (infer C)[] | null ? C : never;

const STEPS = ["School", "Campuses", "Settings", "Owner + package"];

const emptyCampus = (): Campus => ({
  code: "",
  name: "",
  address: "",
  lat: 0,
  lon: 0,
  geofence_m: 150,
});

const initial: WizardBody = {
  school: {
    code: "",
    legal_name: "",
    display_name: "",
    short_name: "",
    address: "",
    city: "",
    state_code: "24",
    contact_name: "",
    contact_mobile: "",
    notes: "",
  },
  campuses: [emptyCampus()],
  settings: {
    approach_minutes: 5,
    delay_alert_minutes: 10,
    speed_limit_kmph: 40,
    photo_on_tap: true,
    photo_retention_days: 30,
    position_retention_days: 30,
    sms_fallback: false,
  },
  owner: { name: "", mobile: "" },
  package: {
    period_start: "",
    students_committed: 0,
    price_per_student_paise: 100000,
    price_includes_gst: true,
  },
};

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    // biome-ignore lint/a11y/noLabelWithoutControl: the control is passed in as children
    <label style={{ display: "grid", gap: 4 }}>
      <span className="mut">{label}</span>
      {children}
    </label>
  );
}

const grid = {
  display: "grid",
  gap: 12,
  gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
} as const;

/** Drops empty optional strings, so the API sees them as absent rather than invalid. */
function clean(b: WizardBody): WizardBody {
  const s = { ...b.school };
  for (const k of ["board", "gstin", "contact_email", "go_live_on"] as const)
    if (!s[k]) delete s[k];
  const owner = { ...b.owner };
  if (!owner.email) delete owner.email;
  const pkg = { ...b.package };
  if (!pkg.agent_id) delete pkg.agent_id;
  return { ...b, school: s, owner, package: pkg };
}

/** `/admin/schools/new`: onboarding/01 "Setup in /admin". Four steps, one save at the end; nothing is stored before. */
export function SchoolWizard() {
  const [step, setStep] = useState(0);
  const [b, setB] = useState<WizardBody>(initial);
  const create = useCreateSchool();
  const campuses = b.campuses ?? [];

  const school = (k: keyof WizardBody["school"], v: string) =>
    setB({ ...b, school: { ...b.school, [k]: v } });
  const settings = (k: keyof WizardBody["settings"], v: number | boolean) =>
    setB({ ...b, settings: { ...b.settings, [k]: v } });
  const campus = (i: number, patch: Partial<Campus>) =>
    setB({ ...b, campuses: campuses.map((c, j) => (j === i ? { ...c, ...patch } : c)) });

  function next(e: FormEvent) {
    e.preventDefault();
    if (step < STEPS.length - 1) setStep(step + 1);
    else create.mutate(clean(b));
  }

  if (create.data) {
    return (
      <div className="card" style={{ maxWidth: 720 }}>
        <h1 className="fs20 b">{b.school.display_name} is set up</h1>
        <p className="mut mt8">
          Stage: setup · package ends {create.data.period_end}. Next: send the data request
          (onboarding/02).
        </p>
        <OneTimeLink link={create.data.owner_link} />
        <Link className="btn btn-primary mt16" to={`/admin/schools/${create.data.id}`}>
          Open the school
        </Link>
      </div>
    );
  }

  return (
    <form className="card" style={{ maxWidth: 900 }} onSubmit={next}>
      <h1 className="fs20 b">New school</h1>
      <div className="stepper mt12">
        {STEPS.map((s, i) => (
          <button
            key={s}
            type="button"
            className={`step ${i < step ? "done" : i === step ? "cur" : ""}`}
            style={{ background: "none", border: 0 }}
            onClick={() => i < step && setStep(i)}
          >
            <span className="c">{i + 1}</span>
            <span>{s}</span>
          </button>
        ))}
      </div>

      <div className="mt16">
        {step === 0 && (
          <div style={grid}>
            <Field label="School code (typed at login)">
              <input
                className="inp"
                required
                pattern="[a-z0-9\-]{3,30}"
                value={b.school.code}
                onChange={(e) => school("code", e.target.value.toLowerCase())}
              />
            </Field>
            <Field label="Legal name">
              <input
                className="inp"
                required
                maxLength={200}
                value={b.school.legal_name}
                onChange={(e) => school("legal_name", e.target.value)}
              />
            </Field>
            <Field label="Display name">
              <input
                className="inp"
                required
                maxLength={120}
                value={b.school.display_name}
                onChange={(e) => school("display_name", e.target.value)}
              />
            </Field>
            <Field label="Short name (signs notifications)">
              <input
                className="inp"
                required
                maxLength={40}
                value={b.school.short_name}
                onChange={(e) => school("short_name", e.target.value)}
              />
            </Field>
            <Field label="Board">
              <select
                className="sel"
                value={b.school.board ?? ""}
                onChange={(e) => school("board", e.target.value)}
              >
                <option value="">—</option>
                {["CBSE", "ICSE", "GSEB", "IB", "IGCSE", "other"].map((x) => (
                  <option key={x}>{x}</option>
                ))}
              </select>
            </Field>
            <Field label="GSTIN">
              <input
                className="inp"
                pattern="[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]"
                value={b.school.gstin ?? ""}
                onChange={(e) => school("gstin", e.target.value.toUpperCase())}
              />
            </Field>
            <Field label="State code (GST, 24 = Gujarat)">
              <input
                className="inp"
                required
                pattern="[0-9]{2}"
                value={b.school.state_code}
                onChange={(e) => school("state_code", e.target.value)}
              />
            </Field>
            <Field label="City">
              <input
                className="inp"
                value={b.school.city}
                onChange={(e) => school("city", e.target.value)}
              />
            </Field>
            <Field label="Address">
              <input
                className="inp"
                value={b.school.address}
                onChange={(e) => school("address", e.target.value)}
              />
            </Field>
            <Field label="Contact name">
              <input
                className="inp"
                required
                value={b.school.contact_name}
                onChange={(e) => school("contact_name", e.target.value)}
              />
            </Field>
            <Field label="Contact mobile">
              <input
                className="inp"
                required
                type="tel"
                pattern="[6-9][0-9]{9}"
                maxLength={10}
                value={b.school.contact_mobile}
                onChange={(e) => school("contact_mobile", e.target.value)}
              />
            </Field>
            <Field label="Contact e-mail">
              <input
                className="inp"
                type="email"
                value={b.school.contact_email ?? ""}
                onChange={(e) => school("contact_email", e.target.value)}
              />
            </Field>
            <Field label="Planned go-live">
              <input
                className="inp"
                type="date"
                value={b.school.go_live_on ?? ""}
                onChange={(e) => school("go_live_on", e.target.value)}
              />
            </Field>
          </div>
        )}

        {step === 1 && (
          <>
            {campuses.map((c, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: rows have no id until saved
              <div key={i} style={{ ...grid, marginBottom: 12 }}>
                <Field label="Code">
                  <input
                    className="inp"
                    required
                    pattern="[A-Z0-9]{2,8}"
                    value={c.code}
                    onChange={(e) => campus(i, { code: e.target.value.toUpperCase() })}
                  />
                </Field>
                <Field label="Name">
                  <input
                    className="inp"
                    required
                    value={c.name}
                    onChange={(e) => campus(i, { name: e.target.value })}
                  />
                </Field>
                <Field label="Latitude">
                  <input
                    className="inp"
                    required
                    type="number"
                    step="any"
                    min={-90}
                    max={90}
                    value={c.lat}
                    onChange={(e) => campus(i, { lat: Number(e.target.value) })}
                  />
                </Field>
                <Field label="Longitude">
                  <input
                    className="inp"
                    required
                    type="number"
                    step="any"
                    min={-180}
                    max={180}
                    value={c.lon}
                    onChange={(e) => campus(i, { lon: Number(e.target.value) })}
                  />
                </Field>
                <Field label="Geofence (m)">
                  <input
                    className="inp"
                    type="number"
                    min={30}
                    max={1000}
                    value={c.geofence_m ?? 150}
                    onChange={(e) => campus(i, { geofence_m: Number(e.target.value) })}
                  />
                </Field>
                {campuses.length > 1 && (
                  <button
                    className="btn btn-sm btn-outline"
                    type="button"
                    style={{ alignSelf: "end" }}
                    onClick={() => setB({ ...b, campuses: campuses.filter((_, j) => j !== i) })}
                  >
                    Remove
                  </button>
                )}
              </div>
            ))}
            {campuses.length < 20 && (
              <button
                className="btn btn-soft"
                type="button"
                onClick={() => setB({ ...b, campuses: [...campuses, emptyCampus()] })}
              >
                Add campus
              </button>
            )}
          </>
        )}

        {step === 2 && (
          <div style={grid}>
            {(
              [
                ["approach_minutes", "Approach notice (min)", 1, 30],
                ["delay_alert_minutes", "Delay alert after (min)", 3, 60],
                ["speed_limit_kmph", "Speed limit (km/h)", 20, 80],
                ["photo_retention_days", "Keep tap photos (days)", 1, 90],
                ["position_retention_days", "Keep positions (days)", 7, 90],
              ] as const
            ).map(([k, label, min, max]) => (
              <Field key={k} label={label}>
                <input
                  className="inp"
                  type="number"
                  required
                  min={min}
                  max={max}
                  value={b.settings[k] as number}
                  onChange={(e) => settings(k, Number(e.target.value))}
                />
              </Field>
            ))}
            <label className="chkrow">
              <input
                type="checkbox"
                checked={b.settings.photo_on_tap ?? true}
                onChange={(e) => settings("photo_on_tap", e.target.checked)}
              />{" "}
              Photo on each tap
            </label>
            <label className="chkrow">
              <input
                type="checkbox"
                checked={b.settings.sms_fallback ?? false}
                onChange={(e) => settings("sms_fallback", e.target.checked)}
              />{" "}
              SMS to parents without the app (costs)
            </label>
          </div>
        )}

        {step === 3 && (
          <div style={grid}>
            <Field label="Owner name">
              <input
                className="inp"
                required
                value={b.owner.name}
                onChange={(e) => setB({ ...b, owner: { ...b.owner, name: e.target.value } })}
              />
            </Field>
            <Field label="Owner mobile (their login)">
              <input
                className="inp"
                required
                type="tel"
                pattern="[6-9][0-9]{9}"
                maxLength={10}
                value={b.owner.mobile}
                onChange={(e) => setB({ ...b, owner: { ...b.owner, mobile: e.target.value } })}
              />
            </Field>
            <Field label="Owner e-mail">
              <input
                className="inp"
                type="email"
                value={b.owner.email ?? ""}
                onChange={(e) => setB({ ...b, owner: { ...b.owner, email: e.target.value } })}
              />
            </Field>
            <Field label="Package starts">
              <input
                className="inp"
                type="date"
                required
                value={b.package.period_start}
                onChange={(e) =>
                  setB({ ...b, package: { ...b.package, period_start: e.target.value } })
                }
              />
            </Field>
            <Field label="Committed students">
              <input
                className="inp"
                type="number"
                required
                min={0}
                max={100000}
                value={b.package.students_committed}
                onChange={(e) =>
                  setB({
                    ...b,
                    package: { ...b.package, students_committed: Number(e.target.value) },
                  })
                }
              />
            </Field>
            <Field label={`Price per student (₹) — ${rupees(b.package.price_per_student_paise)}`}>
              <input
                className="inp"
                type="number"
                required
                min={0}
                step={1}
                value={b.package.price_per_student_paise / 100}
                onChange={(e) =>
                  setB({
                    ...b,
                    package: {
                      ...b.package,
                      price_per_student_paise: Math.round(Number(e.target.value) * 100),
                    },
                  })
                }
              />
            </Field>
            <label className="chkrow">
              <input
                type="checkbox"
                checked={b.package.price_includes_gst ?? true}
                onChange={(e) =>
                  setB({ ...b, package: { ...b.package, price_includes_gst: e.target.checked } })
                }
              />{" "}
              Price includes GST
            </label>
            <Field label="Agent id (optional)">
              <input
                className="inp"
                type="number"
                min={1}
                value={b.package.agent_id ?? ""}
                onChange={(e) =>
                  setB({
                    ...b,
                    package: {
                      ...b.package,
                      agent_id: e.target.value ? Number(e.target.value) : undefined,
                    },
                  })
                }
              />
            </Field>
            <p className="mut fs12" style={{ gridColumn: "1 / -1" }}>
              Total: {rupees(b.package.students_committed * b.package.price_per_student_paise)} for
              12 months. The owner gets a one-time link to set their password; nothing is saved
              until you press Save.
            </p>
          </div>
        )}
      </div>

      {create.error && (
        <p role="alert" className="mt12" style={{ color: "var(--red, #c62828)" }}>
          {create.error.message}
        </p>
      )}
      <div className="row mt16" style={{ gap: 8 }}>
        {step > 0 && (
          <button className="btn btn-outline" type="button" onClick={() => setStep(step - 1)}>
            Back
          </button>
        )}
        <button className="btn btn-primary" type="submit" disabled={create.isPending}>
          {step < STEPS.length - 1 ? "Next" : create.isPending ? "Saving…" : "Save school"}
        </button>
      </div>
    </form>
  );
}
