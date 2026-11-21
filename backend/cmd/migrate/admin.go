package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/platformq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
)

// adminConfig is what create-admin reads: the owner URL, and the portal host to build the link on.
type adminConfig struct {
	config.Base
	AppDomain string `env:"APP_DOMAIN"`
}

type adminArgs struct {
	email, name, role string
}

var emailRE = regexp.MustCompile(`^[a-z0-9._%+-]+@[a-z0-9-]+(\.[a-z0-9-]+)+$`)

func parseAdminArgs(args []string) (adminArgs, error) {
	fs := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var a adminArgs
	fs.StringVar(&a.email, "email", "", "staff e-mail (the passkey's account name)")
	fs.StringVar(&a.name, "name", "", "full name (new users; default: the e-mail's local part)")
	fs.StringVar(&a.role, "role", adminauth.RoleSuperadmin, "superadmin | support | billing | installer (new users)")
	if err := fs.Parse(args); err != nil {
		return a, err
	}
	if fs.NArg() != 0 {
		return a, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	a.email = strings.ToLower(strings.TrimSpace(a.email))
	if !emailRE.MatchString(a.email) {
		return a, errors.New("--email must be an e-mail address")
	}
	switch a.role {
	case adminauth.RoleSuperadmin, adminauth.RoleSupport, adminauth.RoleBilling, adminauth.RoleInstaller:
	default:
		return a, fmt.Errorf("--role %q is not one of superadmin, support, billing, installer", a.role)
	}
	a.name = strings.TrimSpace(a.name)
	if a.name == "" {
		a.name = a.email[:strings.IndexByte(a.email, '@')]
	}
	return a, nil
}

// enrolBase is where the enrolment page is served: the portal host, or the Vite dev server locally.
func enrolBase(cfg adminConfig) (string, error) {
	if cfg.AppDomain != "" {
		return "https://" + cfg.AppDomain, nil
	}
	if cfg.AppEnv == config.EnvLocal {
		return "http://localhost:5173", nil
	}
	return "", fmt.Errorf("APP_DOMAIN is required when APP_ENV=%s (the enrolment link is built on it)", cfg.AppEnv)
}

// createAdmin makes (or re-activates) the staff member and issues a one-time enrolment link, valid 15 minutes.
// Earlier unused links for them stop working. The token is in the #fragment, so it never reaches a server log;
// only its sha256 is stored. Returns the link to print and the user as stored (an existing user keeps their name
// and role, whatever the flags say).
func createAdmin(ctx context.Context, conn *pgx.Conn, a adminArgs, base string) (string, platformq.UpsertPlatformUserRow, error) {
	var (
		link string
		user platformq.UpsertPlatformUserRow
	)
	err := pgx.BeginTxFunc(ctx, conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		u, err := q.UpsertPlatformUser(ctx, a.name, a.email, a.role)
		if err != nil {
			return fmt.Errorf("save platform user: %w", err)
		}
		user = u
		token, hash, err := adminauth.NewToken()
		if err != nil {
			return err
		}
		e, err := q.CreateEnrolment(ctx, u.ID, hash, float64(adminauth.EnrolmentMinute*60), nil)
		if err != nil {
			return fmt.Errorf("save enrolment: %w", err)
		}
		uid := u.ID
		if err := q.InsertPlatformAudit(ctx, platformq.InsertPlatformAuditParams{
			ActorName: "migrate create-admin", Action: "enrolment.issued", Entity: "platform_user", EntityID: &uid,
			Summary: fmt.Sprintf("Passkey enrolment link issued for %s (%s), enrolment %d", u.Email, u.Role, e.ID),
		}); err != nil {
			return fmt.Errorf("audit: %w", err)
		}
		link = base + "/admin/enrol#" + token
		return nil
	})
	return link, user, err
}

func runCreateAdmin(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	a, err := parseAdminArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "migrate create-admin: %v\nusage: migrate create-admin --email=... [--name=...] [--role=superadmin]\n", err)
		return 2
	}
	cfg, err := config.Load[adminConfig]()
	if err != nil {
		fmt.Fprintf(stderr, "migrate: %v\n", err)
		return 1
	}
	if cfg.ServiceName != "migrate" {
		fmt.Fprintf(stderr, "migrate: SVC_NAME is %q, want \"migrate\" (wrong env?)\n", cfg.ServiceName)
		return 1
	}
	base, err := enrolBase(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "migrate create-admin: %v\n", err)
		return 1
	}
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL.Reveal())
	if err != nil {
		fmt.Fprintln(stderr, "migrate create-admin: connect to the database (URL not shown)")
		return 1
	}
	defer func() { _ = conn.Close(ctx) }()
	var role string
	if err := conn.QueryRow(ctx, "SELECT current_user").Scan(&role); err != nil || role != "sf_owner" {
		fmt.Fprintf(stderr, "migrate create-admin: connected as %q; run it as sf_owner\n", role)
		return 1
	}
	link, u, err := createAdmin(ctx, conn, a, base)
	if err != nil {
		fmt.Fprintf(stderr, "migrate create-admin: %v\n", err)
		return 1
	}
	if u.Name != a.name || u.Role != a.role {
		fmt.Fprintf(stderr, "note: %s already exists; kept name %q and role %q (--name/--role apply to new users only)\n",
			u.Email, u.Name, u.Role)
	}
	// The link is a credential until it is used: it goes to this terminal only, never to the JSON log.
	fmt.Fprintf(stdout, "Passkey enrolment link for %s, role %s (valid %d minutes, one use):\n%s\n",
		u.Email, u.Role, adminauth.EnrolmentMinute, link)
	return 0
}
