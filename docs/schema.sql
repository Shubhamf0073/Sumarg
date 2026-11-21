-- Sumarg — reference schema (PostgreSQL 17), multi-tenant. Phase 1 tables only (docs/01).
-- This is the contract. The goose baseline (backend/migrations/00001_init.sql) must produce exactly this; sqlc
-- reads the migrations. Change this file first, then add a goose migration, then `make sqlc`
-- (`make schema-diff` proves the two still match).
--
-- Conventions (same as Previa Cargo)
--   * bigint identity PKs; *_paise bigint for money; timestamptz in UTC; business dates are Asia/Kolkata
--   * enumerations are TEXT + CHECK; nothing is hard-deleted (status + reason); `version` = optimistic lock
--   * coordinates as double precision degrees (WGS-84); distances in metres; speeds km/h
--
-- Tenancy (docs/02 A19–A20, A27)
--   * Tenant = a school organisation (one school, or a trust with several campuses).
--   * Every tenant-owned table has `tenant_id`, UNIQUE (tenant_id, id) and COMPOSITE foreign keys.
--   * RLS is ENABLED + FORCED on every table with tenant_id; policy tenant_id = app_tenant(); unset ⇒ zero rows.
--   * Roles (infra/postgres/init/01-roles.sh): sf_owner (owns schema, runs goose, BYPASSRLS) · sf_app (school
--     portal + mobile apps; RLS; no DELETE) · sf_platform (superadmin API, worker, ingest; BYPASSRLS).
--   * The tenant is known before any tenant row is read: session tokens are `<tenant_id>.<token>`; login takes a
--     school code. The only SECURITY DEFINER functions are at the end of this file.
--
-- Children's data (docs/07 §3): location, boarding events and boarding photos are processed for the school
-- (data fiduciary) for the children's safety during transport — DPDP Rules 2025, Fourth Schedule. Photos and raw
-- positions have short retention (tenant_settings) and are deleted by the worker. No biometric templates are stored.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE FUNCTION app_tenant() RETURNS bigint LANGUAGE sql STABLE AS $$
    SELECT nullif(current_setting('app.tenant_id', true), '')::bigint
$$;

CREATE FUNCTION forbid_update_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% is append-only', TG_TABLE_NAME;
END $$;

-- ═══════════════════════════════ Platform (Previa) ═══════════════════════════════

CREATE TABLE platform_users (                   -- Previa staff; passkeys only (docs/02 A24)
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name             text    NOT NULL,
    email            text    NOT NULL UNIQUE CHECK (email = lower(email)),
    mobile           text    CHECK (mobile ~ '^[6-9][0-9]{9}$'),
    role             text    NOT NULL CHECK (role IN ('superadmin','support','billing','installer')),
    is_active        boolean NOT NULL DEFAULT true,
    last_login_at    timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE platform_passkeys (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    platform_user_id  bigint NOT NULL REFERENCES platform_users(id),
    credential_id     bytea  NOT NULL UNIQUE,
    credential        jsonb  NOT NULL,              -- go-webauthn Credential
    label             text   NOT NULL DEFAULT '',
    created_at        timestamptz NOT NULL DEFAULT now(),
    last_used_at      timestamptz,
    revoked_at        timestamptz
);
CREATE TABLE platform_enrolments (              -- one-time passkey enrolment links
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    platform_user_id  bigint NOT NULL REFERENCES platform_users(id),
    token_hash        bytea  NOT NULL UNIQUE,
    expires_at        timestamptz NOT NULL,
    used_at           timestamptz,
    created_by        bigint REFERENCES platform_users(id),
    created_at        timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE platform_sessions (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    token_hash         bytea  NOT NULL UNIQUE,
    platform_user_id   bigint NOT NULL REFERENCES platform_users(id),
    passkey_id         bigint NOT NULL REFERENCES platform_passkeys(id),
    created_at         timestamptz NOT NULL DEFAULT now(),
    last_seen_at       timestamptz NOT NULL DEFAULT now(),
    expires_at         timestamptz NOT NULL,        -- 12 h
    ip                 inet,
    user_agent         text,
    revoked_at         timestamptz
);

CREATE TABLE agents (                           -- sales agents who bring in schools (docs/10 §5)
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name             text    NOT NULL,
    mobile           text    NOT NULL UNIQUE CHECK (mobile ~ '^[6-9][0-9]{9}$'),
    pan              text    CHECK (pan ~ '^[A-Z]{5}[0-9]{4}[A-Z]$'),
    commission_bp    integer NOT NULL DEFAULT 4000 CHECK (commission_bp BETWEEN 0 AND 10000),  -- 40.00 % of net profit
    commission_mode  text    NOT NULL DEFAULT 'recurring' CHECK (commission_mode IN ('first_year','recurring')),  -- every renewal (docs/09 V3)
    is_active        boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tenants (
    id                       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code                     text    NOT NULL UNIQUE CHECK (code ~ '^[a-z0-9-]{3,30}$'),   -- school code for login
    legal_name               text    NOT NULL,
    display_name             text    NOT NULL,
    short_name               text    NOT NULL,             -- signs notifications: "— DPS Bopal"
    board                    text    CHECK (board IN ('CBSE','ICSE','GSEB','IB','IGCSE','other')),
    gstin                    text    CHECK (gstin ~ '^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$'),
    address                  text    NOT NULL DEFAULT '',
    city                     text    NOT NULL DEFAULT '',
    state_code               char(2) NOT NULL CHECK (state_code ~ '^[0-9]{2}$'),
    contact_name             text    NOT NULL,
    contact_mobile           text    NOT NULL CHECK (contact_mobile ~ '^[6-9][0-9]{9}$'),
    contact_email            text,
    status                   text    NOT NULL DEFAULT 'trial'
                             CHECK (status IN ('trial','active','past_due','suspended','cancelled')),
    trial_ends_at            timestamptz,
    onboarding_stage         text    NOT NULL DEFAULT 'setup'    -- onboarding/README.md; /admin onboarding board
                             CHECK (onboarding_stage IN ('setup','data','survey','install','training','live')),
    go_live_on               date,                               -- planned, then actual
    is_demo                  boolean NOT NULL DEFAULT false,      -- demo school: no SMS/push to real numbers, never invoiced
    price_per_student_paise  bigint  NOT NULL DEFAULT 100000 CHECK (price_per_student_paise >= 0), -- ₹1,000/student/12 months
    price_includes_gst       boolean NOT NULL DEFAULT true,
    hardware_model           text    NOT NULL DEFAULT 'included' CHECK (hardware_model IN ('included','purchased')),
    agent_id                 bigint  REFERENCES agents(id),
    notes                    text    NOT NULL DEFAULT '',
    suspended_at             timestamptz,
    suspend_reason           text,
    cancelled_at             timestamptz,
    created_by               bigint  REFERENCES platform_users(id),
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriptions (                    -- 12-month packages (docs/05 §11)
    id                       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id                bigint  NOT NULL REFERENCES tenants(id),
    period_start             date    NOT NULL,
    period_end               date    NOT NULL,     -- period_start + 12 months − 1 day
    students_committed       integer NOT NULL CHECK (students_committed >= 0),
    price_per_student_paise  bigint  NOT NULL CHECK (price_per_student_paise >= 0),
    status                   text    NOT NULL DEFAULT 'active' CHECK (status IN ('active','ended','cancelled')),
    created_by               bigint  REFERENCES platform_users(id),
    created_at               timestamptz NOT NULL DEFAULT now(),
    version                  integer NOT NULL DEFAULT 1,   -- /admin package edits (optimistic lock)
    UNIQUE (tenant_id, id),
    CONSTRAINT subscriptions_period CHECK (period_end > period_start)
);

CREATE TABLE platform_counters (
    kind        text    NOT NULL CHECK (kind IN ('invoice')),
    period      text    NOT NULL CHECK (period ~ '^[0-9]{4}$'),
    next_value  integer NOT NULL CHECK (next_value > 0),
    PRIMARY KEY (kind, period)
);

CREATE TABLE platform_invoices (                -- Previa → school (PS/{FY}/{n:05})
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id         bigint  NOT NULL REFERENCES tenants(id),
    subscription_id   bigint,
    invoice_no        text    NOT NULL UNIQUE,
    fy                text    NOT NULL CHECK (fy ~ '^[0-9]{4}$'),
    issue_date        date    NOT NULL,
    due_date          date    NOT NULL,
    lines             jsonb   NOT NULL,       -- [{"kind":"package|top_up|hardware|adjustment","students":420,"rate_paise":100000,"amount_paise":…}]
    gross_paise       bigint  NOT NULL CHECK (gross_paise >= 0),
    taxable_paise     bigint  NOT NULL CHECK (taxable_paise >= 0),
    cgst_paise        bigint  NOT NULL DEFAULT 0,
    sgst_paise        bigint  NOT NULL DEFAULT 0,
    igst_paise        bigint  NOT NULL DEFAULT 0,
    status            text    NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','issued','paid','void')),
    paid_at           timestamptz,
    payment_ref       text,
    void_reason       text,
    created_by        bigint  REFERENCES platform_users(id),
    created_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, subscription_id) REFERENCES subscriptions (tenant_id, id),
    CONSTRAINT platform_invoices_tax CHECK (gross_paise = taxable_paise + cgst_paise + sgst_paise + igst_paise)
);

CREATE TABLE agent_commissions (                -- computed when an invoice is paid (docs/10 §5)
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    agent_id            bigint  NOT NULL REFERENCES agents(id),
    tenant_id           bigint  NOT NULL REFERENCES tenants(id),
    invoice_id          bigint  NOT NULL,
    package_no          smallint NOT NULL DEFAULT 1 CHECK (package_no >= 1),  -- 1 = first package, 2+ = renewals
    net_revenue_paise   bigint  NOT NULL,       -- invoice taxable value
    direct_cost_paise   bigint  NOT NULL,       -- hardware amortisation + SIM + server share + SMS (docs/10 §3)
    net_profit_paise    bigint  NOT NULL,
    commission_bp       integer NOT NULL CHECK (commission_bp BETWEEN 0 AND 10000),
    amount_paise        bigint  NOT NULL CHECK (amount_paise >= 0),
    status              text    NOT NULL DEFAULT 'due' CHECK (status IN ('due','paid','void')),
    paid_at             timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (invoice_id, agent_id),
    FOREIGN KEY (tenant_id, invoice_id) REFERENCES platform_invoices (tenant_id, id)
);

CREATE TABLE platform_audit_log (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    at                timestamptz NOT NULL DEFAULT now(),
    platform_user_id  bigint  REFERENCES platform_users(id),
    actor_name        text    NOT NULL,
    action            text    NOT NULL,
    tenant_id         bigint  REFERENCES tenants(id),
    entity            text    NOT NULL,
    entity_id         bigint,
    summary           text    NOT NULL,
    changes           jsonb,
    ip                inet,
    request_id        text
);
CREATE INDEX platform_audit_at ON platform_audit_log (at);
CREATE TRIGGER platform_audit_append_only BEFORE UPDATE OR DELETE ON platform_audit_log
    FOR EACH ROW EXECUTE FUNCTION forbid_update_delete();

CREATE TABLE firmware_releases (                -- signed images (docs/14 §8)
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version       text    NOT NULL UNIQUE CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'),
    hw_revs       text[]  NOT NULL,
    channel       text    NOT NULL CHECK (channel IN ('dev','beta','stable')),
    sha256        bytea   NOT NULL,
    signature     bytea   NOT NULL,             -- ECDSA P-256 over sha256 (Previa firmware key)
    storage_key   text    NOT NULL,
    size_bytes    integer NOT NULL,
    notes         text    NOT NULL DEFAULT '',
    created_by    bigint  REFERENCES platform_users(id),
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE system_state (
    key         text PRIMARY KEY,
    value       jsonb NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- ═══════════════════════════════ School setup ═══════════════════════════════

CREATE TABLE tenant_settings (
    tenant_id                  bigint  PRIMARY KEY REFERENCES tenants(id),
    approach_minutes           smallint NOT NULL DEFAULT 5  CHECK (approach_minutes BETWEEN 1 AND 30),
    delay_alert_minutes        smallint NOT NULL DEFAULT 10 CHECK (delay_alert_minutes BETWEEN 3 AND 60),
    speed_limit_kmph           smallint NOT NULL DEFAULT 40 CHECK (speed_limit_kmph BETWEEN 20 AND 80),
    photo_on_tap               boolean  NOT NULL DEFAULT true,
    photo_retention_days       smallint NOT NULL DEFAULT 30 CHECK (photo_retention_days BETWEEN 1 AND 90),
    position_retention_days    smallint NOT NULL DEFAULT 30 CHECK (position_retention_days BETWEEN 7 AND 90),
    parent_sees_live_only_on_trip boolean NOT NULL DEFAULT true,
    notify_board               boolean  NOT NULL DEFAULT true,
    notify_alight              boolean  NOT NULL DEFAULT true,
    notify_approach            boolean  NOT NULL DEFAULT true,
    sms_fallback               boolean  NOT NULL DEFAULT false,   -- SMS when a parent has no app (costs; docs/10)
    updated_at                 timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE campuses (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id        bigint  NOT NULL REFERENCES tenants(id),
    code             text    NOT NULL CHECK (code ~ '^[A-Z0-9]{2,8}$'),
    name             text    NOT NULL,
    address          text    NOT NULL DEFAULT '',
    lat              double precision NOT NULL CHECK (lat BETWEEN -90 AND 90),
    lon              double precision NOT NULL CHECK (lon BETWEEN -180 AND 180),
    geofence_m       integer NOT NULL DEFAULT 150 CHECK (geofence_m BETWEEN 30 AND 1000),
    is_active        boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, code)
);

CREATE TABLE users (                             -- school staff and crew (not parents)
    id                     bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id              bigint  NOT NULL REFERENCES tenants(id),
    campus_id              bigint,                       -- null = all campuses (owner, admin of a trust)
    name                   text    NOT NULL,
    mobile                 text    NOT NULL CHECK (mobile ~ '^[6-9][0-9]{9}$'),
    email                  text,
    role                   text    NOT NULL CHECK (role IN ('owner','admin','transport_manager','driver','attendant')),
    password_hash          text,                         -- web portal roles
    pin_hash               text,                         -- driver/attendant app: 6-digit PIN + device binding
    must_change_secret     boolean NOT NULL DEFAULT true,
    failed_logins          smallint NOT NULL DEFAULT 0,  -- in a row, since the last success (support view)
    is_active              boolean NOT NULL DEFAULT true,
    version                integer NOT NULL DEFAULT 1,
    created_at             timestamptz NOT NULL DEFAULT now(),
    fail_window_start      timestamptz,                  -- daily cap over every address (docs/05 §10): 24 h window start
    fail_window_count      smallint NOT NULL DEFAULT 0,  -- failures in that window; at 50 every try gets the same 401
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, mobile),
    FOREIGN KEY (tenant_id, campus_id) REFERENCES campuses (tenant_id, id),
    CONSTRAINT users_secret CHECK (
        (role IN ('driver','attendant') AND pin_hash IS NOT NULL) OR
        (role NOT IN ('driver','attendant') AND password_hash IS NOT NULL))
);

-- One-time "set your password" links for portal staff (owner, admin, transport_manager): issued by Previa in the
-- onboarding wizard (M1.4b) and later by the school (M1.5). The link carries `<tenant_id>.<token>`; only sha256(token)
-- is stored. 72 h, single use; a new link for the user spends the older ones (docs/05 §10).
CREATE TABLE user_secret_links (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id    bigint  NOT NULL REFERENCES tenants(id),
    user_id      bigint  NOT NULL,
    token_hash   bytea   NOT NULL UNIQUE,
    expires_at   timestamptz NOT NULL,
    used_at      timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id)
);
CREATE INDEX user_secret_links_live ON user_secret_links (tenant_id, user_id) WHERE used_at IS NULL;

CREATE TABLE crew (                               -- driver / attendant compliance record (1:1 with a crew user)
    id                    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id             bigint  NOT NULL REFERENCES tenants(id),
    user_id               bigint  NOT NULL,
    kind                  text    NOT NULL CHECK (kind IN ('driver','attendant')),
    licence_no            text,
    licence_expiry        date,
    badge_no              text,
    police_verified_on    date,
    medical_checked_on    date,
    photo_file_id         bigint,
    status                text    NOT NULL DEFAULT 'active' CHECK (status IN ('active','on_leave','left')),
    created_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, user_id),
    FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id),
    CONSTRAINT crew_driver_licence CHECK (kind <> 'driver' OR (licence_no IS NOT NULL AND licence_expiry IS NOT NULL))
);

CREATE TABLE vehicles (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id           bigint  NOT NULL REFERENCES tenants(id),
    campus_id           bigint  NOT NULL,
    reg_no              text    NOT NULL CHECK (reg_no ~ '^[A-Z]{2}[0-9]{1,2}[A-Z]{0,3}[0-9]{1,4}$'),
    label               text    NOT NULL,                 -- "Bus 12"
    make_model          text    NOT NULL DEFAULT '',
    seats               smallint NOT NULL CHECK (seats BETWEEN 4 AND 80),
    ownership           text    NOT NULL DEFAULT 'school' CHECK (ownership IN ('school','contractor')),
    contractor_name     text,
    fitness_expiry      date,
    insurance_expiry    date,
    permit_expiry       date,
    puc_expiry          date,
    vltd_vendor         text,                             -- the bus's AIS-140 tracker (kept for the RTO), if known
    status              text    NOT NULL DEFAULT 'active' CHECK (status IN ('active','off_road','retired')),
    version             integer NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, reg_no),
    FOREIGN KEY (tenant_id, campus_id) REFERENCES campuses (tenant_id, id)
);

-- Bus units (our hardware). Rows exist from manufacturing (tenant_id null = Previa stock). A school sees only
-- its allocated units (RLS; SELECT only). Ingest authenticates units against secret_hash (docs/13 §1).
CREATE TABLE bus_units (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uid              text    NOT NULL UNIQUE CHECK (uid ~ '^SB-[0-9A-F]{6}$'),    -- from the ESP32 MAC
    hw_rev           text    NOT NULL,
    imei             text    UNIQUE CHECK (imei ~ '^[0-9]{15}$'),
    iccid            text    UNIQUE,
    secret_hash      text    NOT NULL,                     -- argon2id of the MQTT secret
    fw_version       text,
    fw_channel       text    NOT NULL DEFAULT 'stable' CHECK (fw_channel IN ('dev','beta','stable')),
    tenant_id        bigint  REFERENCES tenants(id),
    vehicle_id       bigint,
    status           text    NOT NULL DEFAULT 'stock' CHECK (status IN ('stock','allocated','installed','repair','retired')),
    installed_at     timestamptz,
    last_seen_at     timestamptz,
    last_status      jsonb,                                -- last status/birth message (docs/13 §4)
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, vehicle_id) REFERENCES vehicles (tenant_id, id),
    CONSTRAINT bus_units_alloc CHECK (status = 'stock' OR status = 'retired' OR tenant_id IS NOT NULL)
);
CREATE UNIQUE INDEX bus_units_one_per_vehicle ON bus_units (vehicle_id) WHERE vehicle_id IS NOT NULL AND status = 'installed';

CREATE TABLE unit_commands (                     -- cloud → unit commands, sent by ingest, acked by the unit
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    unit_id       bigint  NOT NULL REFERENCES bus_units(id),
    cmd           jsonb   NOT NULL,                 -- command.v1.json
    status        text    NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','sent','acked','failed','expired')),
    created_by    bigint  REFERENCES platform_users(id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    sent_at       timestamptz,
    acked_at      timestamptz,
    result        jsonb
);

CREATE TABLE routes (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id           bigint  NOT NULL REFERENCES tenants(id),
    campus_id           bigint  NOT NULL,
    code                text    NOT NULL CHECK (code ~ '^[A-Z0-9-]{2,12}$'),   -- R-01
    name                text    NOT NULL,
    direction           text    NOT NULL CHECK (direction IN ('to_school','from_school')),
    shift               text    NOT NULL DEFAULT 'regular' CHECK (shift IN ('regular','half_day','activity')),
    vehicle_id          bigint,
    driver_crew_id      bigint,
    attendant_crew_id   bigint,
    planned_start       time    NOT NULL,
    polyline            text,                       -- encoded polyline, computed once when stops change
    distance_m          integer,
    is_active           boolean NOT NULL DEFAULT true,
    version             integer NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, code, direction, shift),
    FOREIGN KEY (tenant_id, campus_id)         REFERENCES campuses (tenant_id, id),
    FOREIGN KEY (tenant_id, vehicle_id)        REFERENCES vehicles (tenant_id, id),
    FOREIGN KEY (tenant_id, driver_crew_id)    REFERENCES crew (tenant_id, id),
    FOREIGN KEY (tenant_id, attendant_crew_id) REFERENCES crew (tenant_id, id)
);

CREATE TABLE stops (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id        bigint  NOT NULL REFERENCES tenants(id),
    route_id         bigint  NOT NULL,
    seq              smallint NOT NULL CHECK (seq > 0),
    name             text    NOT NULL,
    lat              double precision NOT NULL CHECK (lat BETWEEN -90 AND 90),
    lon              double precision NOT NULL CHECK (lon BETWEEN -180 AND 180),
    radius_m         smallint NOT NULL DEFAULT 60 CHECK (radius_m BETWEEN 20 AND 300),
    planned_offset_s integer NOT NULL DEFAULT 0 CHECK (planned_offset_s >= 0),   -- from route start
    learned_offset_s integer,                     -- median of the last 10 trips (ETA, docs/05 §5)
    is_active        boolean NOT NULL DEFAULT true,
    UNIQUE (tenant_id, id),
    UNIQUE (route_id, seq),
    FOREIGN KEY (tenant_id, route_id) REFERENCES routes (tenant_id, id)
);

CREATE TABLE students (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id           bigint  NOT NULL REFERENCES tenants(id),
    campus_id           bigint  NOT NULL,
    admission_no        text    NOT NULL,
    name                text    NOT NULL,
    class_name          text    NOT NULL,             -- "Grade 5"
    section             text    NOT NULL DEFAULT '',
    photo_file_id       bigint,
    transport           boolean NOT NULL DEFAULT true,  -- billed while true (docs/05 §11)
    to_route_id         bigint,
    to_stop_id          bigint,
    from_route_id       bigint,
    from_stop_id        bigint,
    special_needs_note  text,                       -- shown to crew only; not shared with other parents
    status              text    NOT NULL DEFAULT 'active' CHECK (status IN ('active','left')),
    left_on             date,
    version             integer NOT NULL DEFAULT 1,
    created_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, admission_no),
    FOREIGN KEY (tenant_id, campus_id)     REFERENCES campuses (tenant_id, id),
    FOREIGN KEY (tenant_id, to_route_id)   REFERENCES routes (tenant_id, id),
    FOREIGN KEY (tenant_id, to_stop_id)    REFERENCES stops (tenant_id, id),
    FOREIGN KEY (tenant_id, from_route_id) REFERENCES routes (tenant_id, id),
    FOREIGN KEY (tenant_id, from_stop_id)  REFERENCES stops (tenant_id, id)
);
CREATE INDEX students_name_trgm ON students USING gin (name gin_trgm_ops);
CREATE INDEX students_to_route ON students (tenant_id, to_route_id) WHERE status = 'active' AND transport;
CREATE INDEX students_from_route ON students (tenant_id, from_route_id) WHERE status = 'active' AND transport;

CREATE TABLE rfid_cards (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id     bigint  NOT NULL REFERENCES tenants(id),
    student_id    bigint  NOT NULL,
    uid           text    NOT NULL CHECK (uid ~ '^[0-9A-F]{8,20}$'),   -- card UID, hex
    issued_at     timestamptz NOT NULL DEFAULT now(),
    revoked_at    timestamptz,
    revoke_reason text,
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, student_id) REFERENCES students (tenant_id, id)
);
CREATE UNIQUE INDEX rfid_cards_active_uid ON rfid_cards (tenant_id, uid) WHERE revoked_at IS NULL;
CREATE UNIQUE INDEX rfid_cards_one_active ON rfid_cards (tenant_id, student_id) WHERE revoked_at IS NULL;

CREATE TABLE guardians (                          -- parents / authorised pickup persons (mobile app users)
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id        bigint  NOT NULL REFERENCES tenants(id),
    name             text    NOT NULL,
    mobile           text    NOT NULL CHECK (mobile ~ '^[6-9][0-9]{9}$'),
    language         text    NOT NULL DEFAULT 'en' CHECK (language IN ('en','gu','hi')),
    app_enabled      boolean NOT NULL DEFAULT true,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, mobile)
);

CREATE TABLE student_guardians (
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    student_id     bigint  NOT NULL,
    guardian_id    bigint  NOT NULL,
    relation       text    NOT NULL CHECK (relation IN ('mother','father','guardian','grandparent','sibling','other')),
    is_primary     boolean NOT NULL DEFAULT false,
    can_pickup     boolean NOT NULL DEFAULT true,
    PRIMARY KEY (tenant_id, student_id, guardian_id),
    FOREIGN KEY (tenant_id, student_id)  REFERENCES students (tenant_id, id),
    FOREIGN KEY (tenant_id, guardian_id) REFERENCES guardians (tenant_id, id)
);
CREATE INDEX student_guardians_guardian ON student_guardians (tenant_id, guardian_id);

-- ═══════════════════════════════ Sessions and app identity ═══════════════════════════════

CREATE TABLE sessions (                            -- portal, driver app and parent app sessions
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id          bigint  NOT NULL REFERENCES tenants(id),
    token_hash         bytea   NOT NULL UNIQUE,
    user_id            bigint,
    guardian_id        bigint,
    platform_user_id   bigint  REFERENCES platform_users(id),    -- Previa support view (read-only)
    client             text    NOT NULL CHECK (client IN ('web','driver_app','parent_app')),
    install_id         text,                                     -- random id per app install (push + device binding)
    read_only          boolean NOT NULL DEFAULT false,
    created_at         timestamptz NOT NULL DEFAULT now(),
    last_seen_at       timestamptz NOT NULL DEFAULT now(),
    expires_at         timestamptz NOT NULL,
    ip                 inet,
    user_agent         text,
    revoked_at         timestamptz,
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, user_id)     REFERENCES users (tenant_id, id),
    FOREIGN KEY (tenant_id, guardian_id) REFERENCES guardians (tenant_id, id),
    CONSTRAINT sessions_one_principal CHECK (num_nonnulls(user_id, guardian_id, platform_user_id) = 1)
);
CREATE INDEX sessions_guardian ON sessions (tenant_id, guardian_id) WHERE revoked_at IS NULL;
CREATE INDEX sessions_user ON sessions (tenant_id, user_id, install_id) WHERE user_id IS NOT NULL; -- known driver install

CREATE TABLE otp_challenges (                      -- parent login (SMS OTP); one live challenge per mobile
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    mobile         text    NOT NULL CHECK (mobile ~ '^[6-9][0-9]{9}$'),
    code_hash      bytea   NOT NULL,                -- HMAC-SHA256(HKDF(SECRETS_KEY), tenant|mobile|code) — internal/shared/otp
    expires_at     timestamptz NOT NULL,            -- 5 min
    attempts       smallint NOT NULL DEFAULT 0 CHECK (attempts <= 5),
    consumed_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    ip             inet,
    UNIQUE (tenant_id, id)
);
CREATE INDEX otp_live ON otp_challenges (tenant_id, mobile, created_at DESC) WHERE consumed_at IS NULL;
CREATE INDEX otp_recent ON otp_challenges (tenant_id, mobile, created_at);   -- per-mobile hourly limit (live or not)
CREATE INDEX otp_expiry ON otp_challenges (created_at);                      -- worker purge after 24 h

CREATE TABLE push_tokens (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    session_id     bigint  NOT NULL,
    platform       text    NOT NULL CHECK (platform IN ('android','ios')),
    fcm_token      text    NOT NULL,
    app_version    text    NOT NULL,
    updated_at     timestamptz NOT NULL DEFAULT now(),
    invalid_at     timestamptz,                     -- FCM said UNREGISTERED
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, fcm_token),
    FOREIGN KEY (tenant_id, session_id) REFERENCES sessions (tenant_id, id)
);

CREATE TABLE files (                                -- photos and documents on local disk (docs/02 A11)
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    kind           text    NOT NULL CHECK (kind IN ('boarding_photo','incident_photo','student_photo','crew_photo','document')),
    storage_key    text    NOT NULL UNIQUE,         -- "<tenant_id>/<kind>/<yyyy-mm-dd>/<uuid>.jpg"
    content_type   text    NOT NULL,
    bytes          integer NOT NULL CHECK (bytes > 0),
    sha256         bytea   NOT NULL,
    expires_at     timestamptz,                     -- boarding photos: now + photo_retention_days
    deleted_at     timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id)
);
CREATE INDEX files_expiring ON files (expires_at) WHERE deleted_at IS NULL AND expires_at IS NOT NULL;

-- ═══════════════════════════════ Trips, attendance, positions ═══════════════════════════════

CREATE TABLE trips (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id          bigint  NOT NULL REFERENCES tenants(id),
    route_id           bigint  NOT NULL,
    service_date       date    NOT NULL,
    vehicle_id         bigint  NOT NULL,
    unit_id            bigint,
    driver_crew_id     bigint,
    attendant_crew_id  bigint,
    status             text    NOT NULL DEFAULT 'scheduled'
                       CHECK (status IN ('scheduled','started','completed','cancelled')),
    started_at         timestamptz,
    started_by         text    CHECK (started_by IN ('driver_app','unit','auto','admin')),
    ended_at           timestamptz,
    inspection_ok      boolean,
    distance_m         integer,
    path               text,                         -- simplified encoded polyline, kept after raw positions expire
    cancel_reason      text,
    version            integer NOT NULL DEFAULT 1,
    created_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, route_id, service_date),
    FOREIGN KEY (tenant_id, route_id)          REFERENCES routes (tenant_id, id),
    FOREIGN KEY (tenant_id, vehicle_id)        REFERENCES vehicles (tenant_id, id),
    FOREIGN KEY (tenant_id, unit_id)           REFERENCES bus_units (tenant_id, id),
    FOREIGN KEY (tenant_id, driver_crew_id)    REFERENCES crew (tenant_id, id),
    FOREIGN KEY (tenant_id, attendant_crew_id) REFERENCES crew (tenant_id, id)
);
CREATE INDEX trips_today ON trips (tenant_id, service_date, status);

CREATE TABLE trip_stops (                          -- arrival/departure per stop (ETA learning, delays)
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    trip_id        bigint  NOT NULL,
    stop_id        bigint  NOT NULL,
    arrived_at     timestamptz,
    departed_at    timestamptz,
    skipped        boolean NOT NULL DEFAULT false,
    PRIMARY KEY (tenant_id, trip_id, stop_id),
    FOREIGN KEY (tenant_id, trip_id) REFERENCES trips (tenant_id, id),
    FOREIGN KEY (tenant_id, stop_id) REFERENCES stops (tenant_id, id)
);

CREATE TABLE inspections (                         -- driver pre-trip checklist (demo: Inspection)
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    trip_id        bigint  NOT NULL,
    crew_id        bigint  NOT NULL,
    items          jsonb   NOT NULL,                -- {"brakes":"ok","first_aid":"ok","fire_extinguisher":"missing",…}
    passed         boolean NOT NULL,
    notes          text    NOT NULL DEFAULT '',
    at             timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, trip_id),
    FOREIGN KEY (tenant_id, trip_id) REFERENCES trips (tenant_id, id),
    FOREIGN KEY (tenant_id, crew_id) REFERENCES crew (tenant_id, id)
);

CREATE TABLE boardings (                           -- attendance events: board / alight (docs/05 §3)
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id        bigint  NOT NULL REFERENCES tenants(id),
    trip_id          bigint  NOT NULL,
    student_id       bigint  NOT NULL,
    kind             text    NOT NULL CHECK (kind IN ('board','alight')),
    method           text    NOT NULL CHECK (method IN ('rfid','manual')),
    at               timestamptz NOT NULL,
    stop_id          bigint,                         -- nearest stop within its radius, else null (off-stop)
    lat              double precision,
    lon              double precision,
    photo_file_id    bigint,
    unit_id          bigint,
    source_key       text    NOT NULL,              -- "<unit uid>:<seq>" or "app:<install_id>:<local id>" — idempotency
    by_user_id       bigint,                         -- manual marks
    voided_at        timestamptz,
    void_reason      text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, source_key),
    FOREIGN KEY (tenant_id, trip_id)       REFERENCES trips (tenant_id, id),
    FOREIGN KEY (tenant_id, student_id)    REFERENCES students (tenant_id, id),
    FOREIGN KEY (tenant_id, stop_id)       REFERENCES stops (tenant_id, id),
    FOREIGN KEY (tenant_id, photo_file_id) REFERENCES files (tenant_id, id),
    FOREIGN KEY (tenant_id, unit_id)       REFERENCES bus_units (tenant_id, id),
    FOREIGN KEY (tenant_id, by_user_id)    REFERENCES users (tenant_id, id),
    CONSTRAINT boardings_manual_by CHECK (method <> 'manual' OR by_user_id IS NOT NULL)
);
CREATE INDEX boardings_trip ON boardings (tenant_id, trip_id, student_id, at);
CREATE INDEX boardings_student ON boardings (tenant_id, student_id, at DESC);

CREATE TABLE absences (                            -- parent says "not travelling" (driver skips the stop)
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    student_id     bigint  NOT NULL,
    on_date        date    NOT NULL,
    direction      text    NOT NULL CHECK (direction IN ('to_school','from_school','both')),
    guardian_id    bigint,
    reason         text    NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    cancelled_at   timestamptz,
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, student_id)  REFERENCES students (tenant_id, id),
    FOREIGN KEY (tenant_id, guardian_id) REFERENCES guardians (tenant_id, id)
);
CREATE UNIQUE INDEX absences_one ON absences (tenant_id, student_id, on_date, direction) WHERE cancelled_at IS NULL;

-- Raw positions: daily partitions, dropped after position_retention_days (worker → drop_old_position_partitions).
CREATE TABLE positions (
    tenant_id     bigint   NOT NULL,
    unit_id       bigint   NOT NULL,
    vehicle_id    bigint,
    trip_id       bigint,
    seq           bigint   NOT NULL,                -- device lifetime sequence (docs/13 §3)
    at            timestamptz NOT NULL,
    lat           double precision NOT NULL,
    lon           double precision NOT NULL,
    speed_kmh     real,
    heading       smallint,
    sats          smallint,
    hdop          real,
    ign           boolean,
    PRIMARY KEY (unit_id, at, seq)
) PARTITION BY RANGE (at);
CREATE INDEX positions_vehicle_at ON positions (tenant_id, vehicle_id, at);

CREATE TABLE vehicle_live (                        -- latest state per bus (written by ingest at most every 30 s)
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    vehicle_id     bigint  NOT NULL,
    unit_id        bigint,
    trip_id        bigint,
    at             timestamptz NOT NULL,
    lat            double precision NOT NULL,
    lon            double precision NOT NULL,
    speed_kmh      real,
    heading        smallint,
    next_stop_id   bigint,
    eta_next_at    timestamptz,
    online         boolean NOT NULL DEFAULT true,
    PRIMARY KEY (tenant_id, vehicle_id),
    FOREIGN KEY (tenant_id, vehicle_id) REFERENCES vehicles (tenant_id, id)
);

-- ═══════════════════════════════ Safety, notifications, requests ═══════════════════════════════

CREATE TABLE alerts (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id        bigint  NOT NULL REFERENCES tenants(id),
    kind             text    NOT NULL CHECK (kind IN ('sos','overspeed','route_deviation','late_start','delay',
                             'unit_offline','power_cut','tamper','not_dropped','unknown_card','inspection_failed')),
    severity         text    NOT NULL CHECK (severity IN ('critical','high','normal')),
    vehicle_id       bigint,
    trip_id          bigint,
    student_id       bigint,
    at               timestamptz NOT NULL,
    lat              double precision,
    lon              double precision,
    details          jsonb   NOT NULL DEFAULT '{}',
    source           text    NOT NULL CHECK (source IN ('unit','driver_app','system','admin')),
    source_key       text,                           -- idempotency for unit events
    status           text    NOT NULL DEFAULT 'open' CHECK (status IN ('open','acknowledged','resolved')),
    acknowledged_by  bigint,
    acknowledged_at  timestamptz,
    resolved_by      bigint,
    resolved_at      timestamptz,
    resolution       text,
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, source_key),
    FOREIGN KEY (tenant_id, vehicle_id)      REFERENCES vehicles (tenant_id, id),
    FOREIGN KEY (tenant_id, trip_id)         REFERENCES trips (tenant_id, id),
    FOREIGN KEY (tenant_id, student_id)      REFERENCES students (tenant_id, id),
    FOREIGN KEY (tenant_id, acknowledged_by) REFERENCES users (tenant_id, id),
    FOREIGN KEY (tenant_id, resolved_by)     REFERENCES users (tenant_id, id)
);
CREATE INDEX alerts_open ON alerts (tenant_id, at DESC) WHERE status <> 'resolved';

CREATE TABLE incidents (                           -- reported by crew or admin (demo: Incidents)
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    trip_id        bigint,
    vehicle_id     bigint,
    kind           text    NOT NULL CHECK (kind IN ('breakdown','accident','medical','student_issue','traffic','harassment','other')),
    description    text    NOT NULL,
    photo_file_ids bigint[] NOT NULL DEFAULT '{}',
    status         text    NOT NULL DEFAULT 'open' CHECK (status IN ('open','in_progress','resolved')),
    reported_by    bigint  NOT NULL,
    at             timestamptz NOT NULL DEFAULT now(),
    resolution     text,
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, trip_id)     REFERENCES trips (tenant_id, id),
    FOREIGN KEY (tenant_id, vehicle_id)  REFERENCES vehicles (tenant_id, id),
    FOREIGN KEY (tenant_id, reported_by) REFERENCES users (tenant_id, id)
);

CREATE TABLE notifications (                       -- outbox for push/SMS (worker sends; docs/05 §7)
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id        bigint  NOT NULL REFERENCES tenants(id),
    guardian_id      bigint,
    user_id          bigint,
    event            text    NOT NULL CHECK (event IN ('boarded','alighted','approaching','delay','trip_started',
                             'arrived_school','sos','absence_ack','announcement','not_boarded','otp')),
    channel          text    NOT NULL CHECK (channel IN ('push','sms')),
    title            text    NOT NULL,
    body             text    NOT NULL,
    data             jsonb   NOT NULL DEFAULT '{}',   -- deep link: {"screen":"tracking","trip_id":…}
    dedupe_key       text    NOT NULL,               -- "boarded:<boarding id>:<guardian id>"
    status           text    NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','sent','failed','skipped')),
    attempts         smallint NOT NULL DEFAULT 0,
    next_attempt_at  timestamptz NOT NULL DEFAULT now(),
    sent_at          timestamptz,
    error            text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, dedupe_key),
    FOREIGN KEY (tenant_id, guardian_id) REFERENCES guardians (tenant_id, id),
    FOREIGN KEY (tenant_id, user_id)     REFERENCES users (tenant_id, id),
    CONSTRAINT notifications_one_recipient CHECK (num_nonnulls(guardian_id, user_id) = 1)
);
CREATE INDEX notifications_due ON notifications (next_attempt_at) WHERE status = 'queued';
CREATE INDEX notifications_otp ON notifications (tenant_id, created_at) WHERE event = 'otp';  -- per-school SMS limit, purge

CREATE TABLE announcements (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    campus_id      bigint,
    route_id       bigint,                           -- null = whole campus/school
    title          text    NOT NULL,
    body           text    NOT NULL,
    created_by     bigint  NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, campus_id)  REFERENCES campuses (tenant_id, id),
    FOREIGN KEY (tenant_id, route_id)   REFERENCES routes (tenant_id, id),
    FOREIGN KEY (tenant_id, created_by) REFERENCES users (tenant_id, id)
);

CREATE TABLE requests (                            -- parent → school (stop change, pickup change, other)
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    student_id     bigint  NOT NULL,
    guardian_id    bigint  NOT NULL,
    kind           text    NOT NULL CHECK (kind IN ('stop_change','pickup_person','contact','other')),
    details        jsonb   NOT NULL,
    status         text    NOT NULL DEFAULT 'open' CHECK (status IN ('open','approved','rejected','closed')),
    handled_by     bigint,
    handled_at     timestamptz,
    response       text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, student_id)  REFERENCES students (tenant_id, id),
    FOREIGN KEY (tenant_id, guardian_id) REFERENCES guardians (tenant_id, id),
    FOREIGN KEY (tenant_id, handled_by)  REFERENCES users (tenant_id, id)
);

CREATE TABLE audit_log (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id      bigint  NOT NULL REFERENCES tenants(id),
    at             timestamptz NOT NULL DEFAULT now(),
    user_id        bigint,
    guardian_id    bigint,
    platform_user_id bigint,
    action         text    NOT NULL,
    entity         text    NOT NULL,
    entity_id      bigint,
    summary        text    NOT NULL,
    changes        jsonb,
    ip             inet,
    request_id     text
);
CREATE INDEX audit_entity ON audit_log (tenant_id, entity, entity_id, at);
CREATE TRIGGER audit_log_append_only BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION forbid_update_delete();

-- ═══════════════════════════════ Row-Level Security ═══════════════════════════════

-- Every base table with a tenant_id column (incl. the partitioned `positions` parent): enable + force RLS.
DO $$
DECLARE t text;
BEGIN
    FOR t IN
        SELECT c.relname FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
        JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped
        WHERE c.relkind IN ('r','p') AND NOT c.relispartition
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = app_tenant()) WITH CHECK (tenant_id = app_tenant())', t);
    END LOOP;
END $$;

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_self ON tenants USING (id = app_tenant());

-- ═══════════════════════════════ Partition maintenance (SECURITY DEFINER, owner-run) ═══════════════════════════════
-- The worker runs as sf_platform, which can't create or drop partitions of a table it doesn't own; these two
-- functions are the narrow door. Both are idempotent.

CREATE FUNCTION ensure_position_partitions(p_days_ahead int DEFAULT 7)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE d date;
BEGIN
    FOR d IN SELECT generate_series(current_date - 1, current_date + p_days_ahead, interval '1 day')::date LOOP
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF positions FOR VALUES FROM (%L) TO (%L)',
                       'positions_' || to_char(d, 'YYYYMMDD'), d::timestamptz, (d + 1)::timestamptz);
    END LOOP;
END $$;

CREATE FUNCTION drop_old_position_partitions(p_keep_days int)
RETURNS int LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE r record; n int := 0;
BEGIN
    IF p_keep_days < 7 THEN RAISE EXCEPTION 'keep at least 7 days'; END IF;
    FOR r IN
        SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
        WHERE i.inhparent = 'positions'::regclass
          AND c.relname ~ '^positions_[0-9]{8}$'
          AND to_date(substr(c.relname, 11), 'YYYYMMDD') < current_date - p_keep_days
    LOOP
        EXECUTE format('DROP TABLE %I', r.relname);
        n := n + 1;
    END LOOP;
    RETURN n;
END $$;

SELECT ensure_position_partitions(7);

-- ═══════════════════════════════ Grants ═══════════════════════════════

REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO sf_app, sf_platform;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO sf_platform;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO sf_platform;

-- sf_app (school portal + apps): no DELETE anywhere. Units, positions, live state and alerts from units are
-- written by ingest (sf_platform); the app reads them.
GRANT SELECT, INSERT, UPDATE ON
    tenant_settings, campuses, users, user_secret_links, crew, vehicles, routes, stops, students, rfid_cards, guardians,
    student_guardians, sessions, otp_challenges, push_tokens, files, trips, trip_stops, inspections, boardings,
    absences, alerts, incidents, notifications, announcements, requests
    TO sf_app;
GRANT SELECT ON tenants, subscriptions, platform_invoices, positions, vehicle_live TO sf_app;
-- bus_units: column grant, never secret_hash (the MQTT secret's argon2id hash is read by ingest only).
GRANT SELECT (id, uid, hw_rev, imei, iccid, fw_version, fw_channel, tenant_id, vehicle_id, status,
              installed_at, last_seen_at, last_status, created_at) ON bus_units TO sf_app;
GRANT SELECT, INSERT ON audit_log TO sf_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO sf_app;
-- Deliberately NOT granted to sf_app: platform_*, agents, agent_commissions, firmware_releases, unit_commands,
-- system_state.

REVOKE ALL ON FUNCTION ensure_position_partitions(int), drop_old_position_partitions(int) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION ensure_position_partitions(int), drop_old_position_partitions(int) TO sf_platform;

-- ═══════════════════════════════ Pre-tenant lookup (SECURITY DEFINER) ═══════════════════════════════

CREATE FUNCTION tenant_by_code(p_code text)
RETURNS TABLE (tenant_id bigint, status text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
    SELECT t.id, t.status FROM tenants t WHERE t.code = lower(p_code)
$$;
REVOKE ALL ON FUNCTION tenant_by_code(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION tenant_by_code(text) TO sf_app;
