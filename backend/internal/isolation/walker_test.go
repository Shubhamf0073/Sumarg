package isolation

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// The walker has to catch a leaky route, not just pass scoped ones. These fake parent operations answer from an
// in-memory ownership table keyed by the X-Guardian header (stand-in for the session).

var (
	me    = Principal{Name: "A", Tenant: 1, Guardian: 10, IDs: IDs{Student: 100, Trip: 200, Route: 300}}
	other = Principal{Name: "B (same school)", Tenant: 1, Guardian: 11, IDs: IDs{Student: 101, Trip: 201, Route: 301}}
	far   = Principal{Name: "C (other school)", Tenant: 2, Guardian: 20, IDs: IDs{Student: 102, Trip: 202, Route: 302}}
)

var owns = map[int64]IDs{me.Guardian: me.IDs, other.Guardian: other.IDs, far.Guardian: far.IDs}

func caller(ctx context.Context) int64 {
	g, _ := ctx.Value(guardianKey{}).(int64)
	return g
}

type guardianKey struct{}

func withGuardian(api huma.API) {
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		g, _ := strconv.ParseInt(ctx.Header("X-Guardian"), 10, 64)
		next(huma.WithValue(ctx, guardianKey{}, g))
	})
}

type idIn struct {
	ID int64 `path:"id"`
}
type out struct{ Body struct{ OK bool } }

func ownsID(g int64, k Kind, id int64) bool { return owns[g][k] == id }

func newAPI(t *testing.T, leakTrips bool, extra func(huma.API)) (huma.API, http.Handler) {
	t.Helper()
	r := chi.NewRouter()
	api := httpx.NewAPI(r, "test", "1")
	withGuardian(api)
	huma.Register(api, huma.Operation{OperationID: "parent-child", Method: http.MethodGet, Path: "/parent/children/{id}"},
		func(ctx context.Context, in *idIn) (*out, error) {
			if !ownsID(caller(ctx), Student, in.ID) {
				return nil, httpx.ErrNotFound
			}
			return &out{}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "parent-trip", Method: http.MethodGet, Path: "/parent/trips/{id}"},
		func(ctx context.Context, in *idIn) (*out, error) {
			if !leakTrips && !ownsID(caller(ctx), Trip, in.ID) {
				return nil, httpx.ErrNotFound
			}
			return &out{}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "parent-home", Method: http.MethodGet, Path: "/parent/home"},
		func(context.Context, *struct{}) (*out, error) { return &out{}, nil })
	if extra != nil {
		extra(api)
	}
	return api, r
}

func walk(api huma.API, h http.Handler) Walk {
	return Walk{
		API: api, Handler: h, Me: me, Others: []Principal{other, far},
		Auth:  func(p Principal) http.Header { return http.Header{"X-Guardian": {strconv.FormatInt(p.Guardian, 10)}} },
		NoIDs: map[string]string{"parent-home": "lists only the caller's children"},
	}
}

func TestWalkerPassesScopedRoutes(t *testing.T) {
	api, h := newAPI(t, false, nil)
	ops, _ := walk(api, h).Operations()
	if len(ops) != 3 {
		t.Fatalf("found %d parent operations, want 3", len(ops))
	}
	if p := walk(api, h).Check(t.Context()); len(p) != 0 {
		t.Fatalf("scoped routes flagged:\n%s", strings.Join(p, "\n"))
	}
}

func TestWalkerCatchesLeakyRoute(t *testing.T) {
	api, h := newAPI(t, true, nil)
	p := walk(api, h).Check(t.Context())
	if len(p) != 2 { // B's trip and C's trip
		t.Fatalf("want 2 problems (leaky trip × 2 other guardians), got:\n%s", strings.Join(p, "\n"))
	}
	for _, s := range p {
		if !strings.Contains(s, "/parent/trips/{id}") || !strings.Contains(s, "got 200") {
			t.Errorf("unexpected problem: %s", s)
		}
	}
}

func TestWalkerRefusesToGuess(t *testing.T) {
	type bodyIn struct {
		Body struct {
			StudentID int64  `json:"student_id"`
			Date      string `json:"date"`
		}
	}
	type stopIn struct {
		ID int64 `path:"id"`
	}
	api, h := newAPI(t, false, func(api huma.API) {
		huma.Register(api, huma.Operation{OperationID: "parent-absence-create", Method: http.MethodPost, Path: "/parent/absences"},
			func(context.Context, *bodyIn) (*out, error) { return &out{}, nil })
		huma.Register(api, huma.Operation{OperationID: "parent-stop", Method: http.MethodGet, Path: "/parent/stops/{id}"},
			func(context.Context, *stopIn) (*out, error) { return &out{}, nil })
		huma.Register(api, huma.Operation{OperationID: "parent-notifications", Method: http.MethodGet, Path: "/parent/notifications"},
			func(context.Context, *struct{}) (*out, error) { return &out{}, nil })
	})
	_, problems := walk(api, h).Operations()
	want := []string{
		"parent-absence-create (POST /parent/absences): body has id fields [student_id]",
		"parent-stop (GET /parent/stops/{id}): path param {id}",
		"parent-notifications (GET /parent/notifications): takes no id",
	}
	if len(problems) != len(want) {
		t.Fatalf("want %d problems, got:\n%s", len(want), strings.Join(problems, "\n"))
	}
	for _, w := range want {
		found := false
		for _, p := range problems {
			found = found || strings.HasPrefix(p, w)
		}
		if !found {
			t.Errorf("missing problem %q in:\n%s", w, strings.Join(problems, "\n"))
		}
	}
}

func TestWalkerWalksBodyIDs(t *testing.T) {
	type bodyIn struct {
		Body struct {
			StudentID int64 `json:"student_id"`
		}
	}
	extra := func(leak bool) func(huma.API) {
		return func(api huma.API) {
			huma.Register(api, huma.Operation{OperationID: "parent-absence-create", Method: http.MethodPost,
				Path: "/parent/absences", DefaultStatus: http.StatusCreated},
				func(ctx context.Context, in *bodyIn) (*out, error) {
					if !leak && !ownsID(caller(ctx), Student, in.Body.StudentID) {
						return nil, httpx.ErrNotFound
					}
					return &out{}, nil
				})
		}
	}
	bodies := map[string]Body{"parent-absence-create": {
		Kinds: []Kind{Student},
		Make:  func(ids IDs) any { return map[string]any{"student_id": ids[Student]} },
	}}
	for _, leak := range []bool{false, true} {
		api, h := newAPI(t, false, extra(leak))
		w := walk(api, h)
		w.Bodies = bodies
		p := w.Check(t.Context())
		if wantN := map[bool]int{false: 0, true: 2}[leak]; len(p) != wantN {
			t.Errorf("leak=%v: want %d problems, got:\n%s", leak, wantN, strings.Join(p, "\n"))
		}
	}
}

func TestWalkerNeedsAPositiveControl(t *testing.T) {
	api, h := newAPI(t, false, nil)
	w := walk(api, h)
	w.Me.IDs = IDs{Student: 999, Trip: 998} // ids nobody owns: every call 404s, which must not count as a pass
	p := w.Check(t.Context())
	if len(p) != 2 || !strings.Contains(p[0], "positive control") {
		t.Fatalf("want 2 positive-control problems, got:\n%s", strings.Join(p, "\n"))
	}
}

func TestWalkerChecksItsOwnSetup(t *testing.T) {
	type bodyIn struct {
		Body struct {
			StudentID int64 `json:"student_id"`
			TripID    int64 `json:"trip_id"`
		}
	}
	body := func(api huma.API) {
		huma.Register(api, huma.Operation{OperationID: "parent-request-create", Method: http.MethodPost, Path: "/parent/requests"},
			func(context.Context, *bodyIn) (*out, error) { return &out{}, nil })
	}
	cases := []struct {
		name string
		edit func(*Walk)
		want string
	}{
		{"body sample misses an id", func(w *Walk) {
			w.Bodies = map[string]Body{"parent-request-create": {Kinds: []Kind{Student}, Make: func(IDs) any { return nil }}}
		}, "every id in the body must be walked"},
		{"stale NoIDs entry", func(w *Walk) { w.NoIDs["parent-gone"] = "removed" }, "NoIDs lists parent-gone"},
		{"no other school", func(w *Walk) { w.Others = []Principal{other} }, "guardian in another school"},
		{"no same school", func(w *Walk) { w.Others = []Principal{far} }, "guardian in the caller's school"},
		{"shared id", func(w *Walk) {
			o := other
			o.IDs = IDs{Student: me.IDs[Student], Trip: 201, Route: 301}
			w.Others = []Principal{o, far}
		}, "shares student id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api, h := newAPI(t, false, body)
			w := walk(api, h)
			w.NoIDs = map[string]string{"parent-home": "lists only the caller's children"}
			w.Bodies = map[string]Body{"parent-request-create": {Kinds: []Kind{Student, Trip},
				Make: func(ids IDs) any { return map[string]any{"student_id": ids[Student], "trip_id": ids[Trip]} }}}
			tc.edit(&w)
			p := strings.Join(w.Check(t.Context()), "\n")
			if !strings.Contains(p, tc.want) {
				t.Fatalf("want a problem containing %q, got:\n%s", tc.want, p)
			}
		})
	}
}
