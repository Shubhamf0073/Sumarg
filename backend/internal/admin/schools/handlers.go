// Package schools is the /admin onboarding wizard and board (docs/04 "Admin", onboarding/01 "Setup in /admin"):
// one save creates a school with its campuses, settings, owner login (a one-time set-password link) and first
// package; the board moves schools through onboarding stages (onboarding/README.md).
//
// Runs on the platform pool (sf_platform, BYPASSRLS) behind the admin passkey session: every query names the tenant.
package schools

import (
	"context"
	"net/http"
	"net/netip"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/money"
)

// Paths (docs/04 "Admin").
const (
	PathSchools    = "/api/admin/schools"
	PathSchool     = "/api/admin/schools/{id}"
	PathOnboarding = "/api/admin/schools/{id}/onboarding"
	PathUserLink   = "/api/admin/schools/{id}/users/{user_id}/password-link"
)

// Who may do what. Superadmin always may (adminauth.Require).
var (
	writers = []string{adminauth.RoleSupport}
	readers = []string{adminauth.RoleSupport, adminauth.RoleBilling, adminauth.RoleInstaller}
)

// Register adds the schools operations.
func Register(api huma.API) {
	tags := []string{"admin-schools"}
	huma.Register(api, huma.Operation{
		OperationID: "schools-create", Method: http.MethodPost, Path: PathSchools, Tags: tags,
		Summary:       "Onboarding wizard: school, campuses, settings, owner login and first package in one save",
		DefaultStatus: http.StatusCreated,
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict,
			http.StatusUnprocessableEntity},
	}, create)
	huma.Register(api, huma.Operation{
		OperationID: "schools-list", Method: http.MethodGet, Path: PathSchools, Tags: tags,
		Summary: "Onboarding board: schools with their stage and go-live date",
		Errors:  []int{http.StatusUnauthorized, http.StatusForbidden},
	}, list)
	huma.Register(api, huma.Operation{
		OperationID: "schools-get", Method: http.MethodGet, Path: PathSchool, Tags: tags,
		Summary: "One school: details, campuses, owner/admin logins, current package",
		Errors:  []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, get)
	huma.Register(api, huma.Operation{
		OperationID: "schools-set-stage", Method: http.MethodPatch, Path: PathOnboarding, Tags: tags,
		Summary: "Move a school to another onboarding stage; `live` with no go-live date sets today (IST)",
		Errors:  []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict},
	}, setStage)
	huma.Register(api, huma.Operation{
		OperationID: "schools-password-link", Method: http.MethodPost, Path: PathUserLink, Tags: tags,
		Summary:       "A new one-time set-password link for a school's owner or admin (older links stop working)",
		DefaultStatus: http.StatusCreated,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, passwordLink)
}

// ── shapes ──

// SchoolIn is the wizard's first step.
type SchoolIn struct {
	Code          string  `json:"code" pattern:"^[a-z0-9-]{3,30}$" doc:"school code, typed at login and printed on invites"`
	LegalName     string  `json:"legal_name" minLength:"1" maxLength:"200"`
	DisplayName   string  `json:"display_name" minLength:"1" maxLength:"120"`
	ShortName     string  `json:"short_name" minLength:"1" maxLength:"40" doc:"signs notifications"`
	Board         *string `json:"board,omitempty" enum:"CBSE,ICSE,GSEB,IB,IGCSE,other"`
	GSTIN         *string `json:"gstin,omitempty" pattern:"^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$"`
	Address       string  `json:"address" maxLength:"500"`
	City          string  `json:"city" maxLength:"100"`
	StateCode     string  `json:"state_code" pattern:"^[0-9]{2}$" doc:"GST state code, 24 = Gujarat"`
	ContactName   string  `json:"contact_name" minLength:"1" maxLength:"120"`
	ContactMobile string  `json:"contact_mobile" pattern:"^[6-9][0-9]{9}$"`
	ContactEmail  *string `json:"contact_email,omitempty" format:"email" maxLength:"200"`
	GoLiveOn      *string `json:"go_live_on,omitempty" format:"date" doc:"planned go-live (IST date)"`
	Notes         string  `json:"notes" maxLength:"2000"`
}

// CampusIn is one campus.
type CampusIn struct {
	Code      string  `json:"code" pattern:"^[A-Z0-9]{2,8}$"`
	Name      string  `json:"name" minLength:"1" maxLength:"120"`
	Address   string  `json:"address" maxLength:"500"`
	Lat       float64 `json:"lat" minimum:"-90" maximum:"90"`
	Lon       float64 `json:"lon" minimum:"-180" maximum:"180"`
	GeofenceM int32   `json:"geofence_m" minimum:"30" maximum:"1000" default:"150" required:"false"`
}

// SettingsIn is the school's settings (docs/schema.sql tenant_settings; the rest keep their defaults).
type SettingsIn struct {
	ApproachMinutes       int16 `json:"approach_minutes" minimum:"1" maximum:"30" default:"5" required:"false"`
	DelayAlertMinutes     int16 `json:"delay_alert_minutes" minimum:"3" maximum:"60" default:"10" required:"false"`
	SpeedLimitKmph        int16 `json:"speed_limit_kmph" minimum:"20" maximum:"80" default:"40" required:"false"`
	PhotoOnTap            bool  `json:"photo_on_tap" default:"true" required:"false"`
	PhotoRetentionDays    int16 `json:"photo_retention_days" minimum:"1" maximum:"90" default:"30" required:"false"`
	PositionRetentionDays int16 `json:"position_retention_days" minimum:"7" maximum:"90" default:"30" required:"false"`
	SMSFallback           bool  `json:"sms_fallback" default:"false" required:"false"`
}

// OwnerIn is the school's owner login.
type OwnerIn struct {
	Name   string  `json:"name" minLength:"1" maxLength:"120"`
	Mobile string  `json:"mobile" pattern:"^[6-9][0-9]{9}$"`
	Email  *string `json:"email,omitempty" format:"email" maxLength:"200"`
}

// PackageIn is the first 12-month package (docs/05 §11).
type PackageIn struct {
	PeriodStart          string      `json:"period_start" format:"date"`
	StudentsCommitted    int32       `json:"students_committed" minimum:"0" maximum:"100000"`
	PricePerStudentPaise money.Paise `json:"price_per_student_paise" minimum:"0" maximum:"100000000" doc:"integer paise; ₹1,000 = 100000"`
	PriceIncludesGST     bool        `json:"price_includes_gst" default:"true" required:"false"`
	AgentID              *int64      `json:"agent_id,omitempty" minimum:"1"`
}

// CreateBody is the whole wizard, saved at once.
type CreateBody struct {
	School   SchoolIn   `json:"school"`
	Campuses []CampusIn `json:"campuses" minItems:"1" maxItems:"20"`
	Settings SettingsIn `json:"settings"`
	Owner    OwnerIn    `json:"owner"`
	Package  PackageIn  `json:"package"`
}

// LinkBody is a one-time set-password link: shown once, never stored (only its hash).
type LinkBody struct {
	UserID    int64     `json:"user_id"`
	URL       string    `json:"url" doc:"give it to the person over a channel they control; it works once"`
	ExpiresAt time.Time `json:"expires_at"`
}

type createIn struct {
	Body CreateBody
	ip   *netip.Addr
}

func (in *createIn) Resolve(ctx huma.Context) []error { in.ip = adminauth.ClientIP(ctx); return nil }

type createOut struct {
	Body struct {
		ID        int64    `json:"id"`
		Code      string   `json:"code"`
		Stage     string   `json:"onboarding_stage"`
		PeriodEnd string   `json:"period_end" format:"date"`
		OwnerLink LinkBody `json:"owner_link"`
		CampusIDs []int64  `json:"campus_ids"`
	}
}

func create(ctx context.Context, in *createIn) (*createOut, error) {
	p, err := adminauth.Require(ctx, writers...)
	if err != nil {
		return nil, err
	}
	res, err := createSchool(ctx, p, in.ip, in.Body)
	if err != nil {
		return nil, err
	}
	out := &createOut{}
	out.Body.ID, out.Body.Code, out.Body.Stage = res.id, in.Body.School.Code, res.stage
	out.Body.PeriodEnd, out.Body.OwnerLink, out.Body.CampusIDs = res.periodEnd, res.link, res.campusIDs
	return out, nil
}

// SchoolRow is one school on the board.
type SchoolRow struct {
	ID       int64     `json:"id"`
	Code     string    `json:"code"`
	Name     string    `json:"display_name"`
	Status   string    `json:"status" enum:"trial,active,past_due,suspended,cancelled"`
	Stage    string    `json:"onboarding_stage" enum:"setup,data,survey,install,training,live"`
	GoLiveOn *string   `json:"go_live_on" format:"date"`
	IsDemo   bool      `json:"is_demo"`
	Created  time.Time `json:"created_at"`
}

type listIn struct {
	After int64 `query:"after" minimum:"0" doc:"cursor: the last id of the previous page"`
}

type listOut struct {
	Body struct {
		Schools []SchoolRow `json:"schools"`
		Next    *int64      `json:"next" doc:"pass as ?after= for the next page; null = last page"`
	}
}

func list(ctx context.Context, in *listIn) (*listOut, error) {
	if _, err := adminauth.Require(ctx, readers...); err != nil {
		return nil, err
	}
	rows, next, err := listSchools(ctx, in.After)
	if err != nil {
		return nil, err
	}
	out := &listOut{}
	out.Body.Schools, out.Body.Next = rows, next
	return out, nil
}

// SchoolDetail is the school page.
type SchoolDetail struct {
	SchoolRow
	LegalName        string      `json:"legal_name"`
	ShortName        string      `json:"short_name"`
	Board            *string     `json:"board"`
	GSTIN            *string     `json:"gstin"`
	City             string      `json:"city"`
	StateCode        string      `json:"state_code"`
	ContactName      string      `json:"contact_name"`
	ContactMobile    string      `json:"contact_mobile"`
	PriceIncludesGST bool        `json:"price_includes_gst"`
	AgentID          *int64      `json:"agent_id"`
	Campuses         []CampusRow `json:"campuses"`
	Logins           []LoginRow  `json:"logins"`
	Package          *PackageRow `json:"package" doc:"null = no active package"`
}

// CampusRow is a campus on the school page.
type CampusRow struct {
	ID        int64   `json:"id"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	GeofenceM int32   `json:"geofence_m"`
	IsActive  bool    `json:"is_active"`
}

// LoginRow is an owner/admin login; password_set is false until the one-time link is used.
type LoginRow struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Mobile      string `json:"mobile"`
	Role        string `json:"role" enum:"owner,admin"`
	PasswordSet bool   `json:"password_set"`
	IsActive    bool   `json:"is_active"`
}

// PackageRow is the current package.
type PackageRow struct {
	ID                   int64       `json:"id"`
	PeriodStart          string      `json:"period_start" format:"date"`
	PeriodEnd            string      `json:"period_end" format:"date"`
	StudentsCommitted    int32       `json:"students_committed"`
	PricePerStudentPaise money.Paise `json:"price_per_student_paise"`
}

type idIn struct {
	ID int64 `path:"id" minimum:"1"`
}

type getOut struct{ Body SchoolDetail }

func get(ctx context.Context, in *idIn) (*getOut, error) {
	if _, err := adminauth.Require(ctx, readers...); err != nil {
		return nil, err
	}
	d, err := getSchool(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	return &getOut{Body: *d}, nil
}

type stageIn struct {
	ID   int64 `path:"id" minimum:"1"`
	Body struct {
		FromStage string  `json:"from_stage" enum:"setup,data,survey,install,training,live" doc:"the stage you saw (409 if it changed)"`
		Stage     string  `json:"stage" enum:"setup,data,survey,install,training,live"`
		GoLiveOn  *string `json:"go_live_on,omitempty" format:"date" doc:"omit to keep; moving to live with none set uses today (IST)"`
	}
	ip *netip.Addr
}

func (in *stageIn) Resolve(ctx huma.Context) []error { in.ip = adminauth.ClientIP(ctx); return nil }

type stageOut struct {
	Body struct {
		Stage    string  `json:"onboarding_stage"`
		GoLiveOn *string `json:"go_live_on" format:"date"`
	}
}

func setStage(ctx context.Context, in *stageIn) (*stageOut, error) {
	p, err := adminauth.Require(ctx, writers...)
	if err != nil {
		return nil, err
	}
	stage, goLive, err := moveStage(ctx, p, in.ip, in.ID, in.Body.FromStage, in.Body.Stage, in.Body.GoLiveOn)
	if err != nil {
		return nil, err
	}
	out := &stageOut{}
	out.Body.Stage, out.Body.GoLiveOn = stage, goLive
	return out, nil
}

type linkIn struct {
	ID     int64 `path:"id" minimum:"1"`
	UserID int64 `path:"user_id" minimum:"1"`
	ip     *netip.Addr
}

func (in *linkIn) Resolve(ctx huma.Context) []error { in.ip = adminauth.ClientIP(ctx); return nil }

type linkOut struct{ Body LinkBody }

func passwordLink(ctx context.Context, in *linkIn) (*linkOut, error) {
	p, err := adminauth.Require(ctx, writers...)
	if err != nil {
		return nil, err
	}
	l, err := reissueLink(ctx, p, in.ip, in.ID, in.UserID)
	if err != nil {
		return nil, err
	}
	return &linkOut{Body: *l}, nil
}
