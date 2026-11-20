package auth

import "context"

// Clients (sessions.client).
const (
	ClientWeb       = "web"
	ClientDriverApp = "driver_app"
	ClientParentApp = "parent_app"
)

// Principal is who a request is signed in as, from the session row. Handlers read it with FromContext; it is set
// only by the session resolver, never from request input.
type Principal struct {
	SessionID  int64
	TenantID   int64
	Client     string
	UserID     int64 // staff or crew user; 0 for a guardian
	GuardianID int64 // parent; 0 for staff/crew
	Role       string
	CampusID   *int64 // campus-scoped staff; nil = all campuses
	Name       string
	// MustChangeSecret: the user still has the password/PIN an admin set (M1.5 adds the change screens).
	MustChangeSecret bool
	// ReadOnly: Previa support view or a suspended school. The request transaction is READ ONLY.
	ReadOnly   bool
	TenantCode string
	TenantName string
}

type principalKey struct{}

// FromContext returns the request's principal, or false outside an authenticated request.
func FromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(*Principal)
	return p, ok && p != nil
}

func withPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}
