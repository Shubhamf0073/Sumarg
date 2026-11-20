// Package isolation holds the parent-scope walker (CLAUDE.md invariant 3, docs/07 S2) used by the isolation suite.
// It is test support: nothing in the running binary imports it.
//
// The walker reads the huma OpenAPI document, finds every parent operation and, for each id the operation takes,
// calls it as one guardian with that single id swapped for another guardian's. Every such call must answer the
// one 404 body. The same call with the guardian's own ids must not, so a route that 404s for everyone can't pass.
//
// It refuses to guess: an id it can't map to a Kind, a request body with ids but no sample, or an operation with no
// ids that isn't on the reviewed NoIDs list is a problem, not a skip. New parent routes therefore can't land
// without being walked.
package isolation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// Kind is the thing a parent-facing id names.
type Kind string

// The kinds parent routes take (docs/04 "Parent app").
const (
	Student  Kind = "student"
	Route    Kind = "route"
	Trip     Kind = "trip"
	Boarding Kind = "boarding"
	Absence  Kind = "absence"
	Request  Kind = "request"
)

// segmentKinds maps the path segment before an `{id}` to what the id names: /parent/children/{id} is a student.
var segmentKinds = map[string]Kind{
	"children": Student, "students": Student,
	"routes": Route, "live": Route,
	"trips":     Trip,
	"boardings": Boarding,
	"absences":  Absence,
	"requests":  Request,
}

// IDs is one guardian's id of each kind.
type IDs map[Kind]int64

// Principal is a signed-in guardian and the ids that are theirs.
type Principal struct {
	Name     string // for messages: "A", "B (same school)", "C (other school)"
	Tenant   int64
	Guardian int64
	IDs      IDs
}

// Body is the sample request body for an operation whose body carries ids. Kinds lists the ids it uses; Make builds
// a valid body from ids (own ids, with one swapped for a foreign one when walking).
type Body struct {
	Kinds []Kind
	Make  func(IDs) any
}

// Walk is one run of the walker.
type Walk struct {
	API      huma.API     // the API the operations are registered on (its OpenAPI document)
	Handler  http.Handler // serves the requests; carries the real auth and tx middleware
	Prefixes []string     // parent path prefixes; default "/parent/" and "/api/parent/"
	// Auth returns the headers that sign p in (bearer `<tenant_id>.<token>`).
	Auth func(p Principal) http.Header
	Me   Principal // the caller
	// Others are guardians whose ids Me must never reach: at least one in Me's school and one in another school.
	Others []Principal
	// NoIDs lists operations (by OperationID) that take no id at all, with the reviewed reason they are scoped
	// anyway (e.g. "lists only the caller's children"). Their own tests cover what they return.
	NoIDs map[string]string
	// Bodies gives a sample body for every operation whose JSON body has `*_id` fields.
	Bodies map[string]Body
	// OwnStatus is the status expected with the caller's own ids when it isn't 2xx (e.g. live → 409 no_trip).
	OwnStatus map[string]int
}

// Op is one parent operation and the ids it takes.
type Op struct {
	ID, Method, Path string
	path             map[string]Kind // path param → kind
	query            map[string]Kind // query param → kind
	body             bool            // JSON body has *_id fields
}

// Kinds is every kind the operation takes, sorted.
func (o Op) Kinds(bodies map[string]Body) []Kind {
	set := map[Kind]bool{}
	for _, k := range o.path {
		set[k] = true
	}
	for _, k := range o.query {
		set[k] = true
	}
	if b, ok := bodies[o.ID]; ok {
		for _, k := range b.Kinds {
			set[k] = true
		}
	}
	return slices.Sorted(maps.Keys(set))
}

// Operations finds the parent operations in the API and maps their ids. Problems are ids it can't map.
func (w Walk) Operations() (ops []Op, problems []string) {
	doc := w.API.OpenAPI()
	for _, p := range slices.Sorted(maps.Keys(doc.Paths)) {
		if !w.isParent(p) {
			continue
		}
		item := doc.Paths[p]
		for _, mo := range []struct {
			method string
			op     *huma.Operation
		}{
			{http.MethodGet, item.Get}, {http.MethodPost, item.Post}, {http.MethodPut, item.Put},
			{http.MethodPatch, item.Patch}, {http.MethodDelete, item.Delete},
		} {
			if mo.op == nil {
				continue
			}
			op, probs := w.mapOp(doc, mo.method, p, mo.op)
			ops = append(ops, op)
			problems = append(problems, probs...)
		}
	}
	seen := map[string]bool{}
	for _, op := range ops {
		seen[op.ID] = true
	}
	for _, id := range slices.Sorted(maps.Keys(w.NoIDs)) {
		if !seen[id] {
			problems = append(problems, fmt.Sprintf("NoIDs lists %s, which is not a parent operation", id))
		}
	}
	for _, id := range slices.Sorted(maps.Keys(w.Bodies)) {
		if !seen[id] {
			problems = append(problems, fmt.Sprintf("Bodies has a sample for %s, which is not a parent operation", id))
		}
	}
	// DELETEs last: their positive control changes the fixture other operations' controls read.
	slices.SortStableFunc(ops, func(a, b Op) int {
		return boolInt(a.Method == http.MethodDelete) - boolInt(b.Method == http.MethodDelete)
	})
	return ops, problems
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (w Walk) isParent(path string) bool {
	prefixes := w.Prefixes
	if len(prefixes) == 0 {
		prefixes = []string{"/parent/", "/api/parent/"}
	}
	for _, pre := range prefixes {
		if strings.HasPrefix(path, pre) || path+"/" == pre {
			return true
		}
	}
	return false
}

func (w Walk) mapOp(doc *huma.OpenAPI, method, path string, o *huma.Operation) (Op, []string) {
	op := Op{ID: o.OperationID, Method: method, Path: path, path: map[string]Kind{}, query: map[string]Kind{}}
	var problems []string
	name := op.ID + " (" + method + " " + path + ")"
	segs := strings.Split(strings.Trim(path, "/"), "/")
	for i, s := range segs {
		if !strings.HasPrefix(s, "{") {
			continue
		}
		param := strings.Trim(s, "{}")
		var k Kind
		var ok bool
		if param == "id" && i > 0 {
			k, ok = segmentKinds[segs[i-1]]
		} else {
			k, ok = kindOfField(param)
		}
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: path param {%s}: the walker can't tell what it names — "+
				"teach internal/isolation (segmentKinds or the Kind list)", name, param))
			continue
		}
		op.path[param] = k
	}
	for _, prm := range o.Parameters {
		if prm == nil || prm.In != "query" {
			continue
		}
		if strings.HasSuffix(prm.Name, "_ids") {
			problems = append(problems, fmt.Sprintf("%s: query param %s: id lists aren't walked yet — "+
				"teach internal/isolation", name, prm.Name))
			continue
		}
		if !strings.HasSuffix(prm.Name, "_id") {
			continue
		}
		k, ok := kindOfField(prm.Name)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: query param %s: unknown kind — teach internal/isolation",
				name, prm.Name))
			continue
		}
		op.query[prm.Name] = k
	}
	if fields := bodyIDFields(doc, o); len(fields) > 0 {
		op.body = true
		b, ok := w.Bodies[op.ID]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: body has id fields %v: add a sample to Walk.Bodies",
				name, fields))
		} else if want := fieldKinds(fields); !slices.Equal(want, sortedKinds(b.Kinds)) {
			problems = append(problems, fmt.Sprintf("%s: body id fields %v name kinds %v but the sample's "+
				"Kinds are %v: every id in the body must be walked", name, fields, want, sortedKinds(b.Kinds)))
		}
	}
	if len(op.path) == 0 && len(op.query) == 0 && !op.body && len(problems) == 0 {
		if _, ok := w.NoIDs[op.ID]; !ok {
			problems = append(problems, fmt.Sprintf("%s: takes no id: list it in NoIDs with the reason its "+
				"result is still the caller's own (and test that directly)", name))
		}
	}
	return op, problems
}

// kindOfField maps `student_id` / `route_id` (or `route`) to a Kind.
func kindOfField(name string) (Kind, bool) {
	k := Kind(strings.TrimSuffix(name, "_id"))
	switch k {
	case Student, Route, Trip, Boarding, Absence, Request:
		return k, true
	}
	return "", false
}

// fieldKinds maps body id fields to their kinds, sorted and deduplicated; an unknown one maps to its own name so
// the comparison with Body.Kinds fails and says which.
func fieldKinds(fields []string) []Kind {
	set := map[Kind]bool{}
	for _, f := range fields {
		k, ok := kindOfField(strings.TrimSuffix(f, "s"))
		if !ok {
			k = Kind(f)
		}
		set[k] = true
	}
	return slices.Sorted(maps.Keys(set))
}

func sortedKinds(ks []Kind) []Kind {
	out := slices.Clone(ks)
	slices.Sort(out)
	return slices.Compact(out)
}

// bodyIDFields lists the top-level `*_id` / `*_ids` properties of the operation's JSON body. Ids nested in objects
// or arrays aren't seen: an operation with those needs its sample body (and Kinds) written with them in mind.
func bodyIDFields(doc *huma.OpenAPI, o *huma.Operation) []string {
	if o.RequestBody == nil {
		return nil
	}
	mt := o.RequestBody.Content["application/json"]
	if mt == nil || mt.Schema == nil {
		return nil
	}
	s := mt.Schema
	if s.Ref != "" && doc.Components != nil && doc.Components.Schemas != nil {
		s = doc.Components.Schemas.SchemaFromRef(s.Ref)
	}
	if s == nil {
		return nil
	}
	var out []string
	for _, f := range slices.Sorted(maps.Keys(s.Properties)) {
		if strings.HasSuffix(f, "_id") || strings.HasSuffix(f, "_ids") {
			out = append(out, f)
		}
	}
	return out
}

// Check runs the walk and returns every problem found; empty means every parent operation is scoped.
func (w Walk) Check(ctx context.Context) []string {
	ops, problems := w.Operations()
	if len(ops) == 0 {
		return problems
	}
	if w.Auth == nil || w.Handler == nil {
		return append(problems, fmt.Sprintf("%d parent operations but no Auth/Handler wired", len(ops)))
	}
	if p := w.checkOthers(); len(p) > 0 {
		return append(problems, p...)
	}
	for _, op := range ops {
		if _, ok := w.NoIDs[op.ID]; ok && !op.body && len(op.path) == 0 && len(op.query) == 0 {
			continue
		}
		kinds := op.Kinds(w.Bodies)
		// Foreign first: a DELETE with own ids would change the fixture.
		for _, k := range kinds {
			for _, other := range w.Others {
				ids, err := swap(w.Me.IDs, other.IDs, k)
				if err != nil {
					problems = append(problems, fmt.Sprintf("%s: fixture: %v for %s", op.ID, err, other.Name))
					continue
				}
				status, code, err := w.do(ctx, op, ids)
				switch {
				case err != nil:
					problems = append(problems, fmt.Sprintf("%s with %s's %s: %v", op.ID, other.Name, k, err))
				case status != http.StatusNotFound || code != "not_found":
					problems = append(problems, fmt.Sprintf("%s %s as %s with %s's %s id %d: got %d %q, want 404 not_found",
						op.Method, op.Path, w.Me.Name, other.Name, k, ids[k], status, code))
				}
			}
		}
		if err := missing(w.Me.IDs, kinds); err != nil {
			problems = append(problems, fmt.Sprintf("%s: fixture for %s: %v", op.ID, w.Me.Name, err))
			continue
		}
		status, code, err := w.do(ctx, op, w.Me.IDs)
		want, fixed := w.OwnStatus[op.ID]
		switch {
		case err != nil:
			problems = append(problems, fmt.Sprintf("%s with own ids: %v", op.ID, err))
		case fixed && status != want, !fixed && (status < 200 || status > 299):
			problems = append(problems, fmt.Sprintf("%s %s as %s with own ids: got %d %q — the positive control "+
				"must pass, or the 404s above prove nothing (set OwnStatus if non-2xx is right)",
				op.Method, op.Path, w.Me.Name, status, code))
		}
	}
	return problems
}

// checkOthers: the walk needs another guardian in the caller's school and one in another school, none sharing an
// id with the caller (a shared id would make the "foreign" call legitimately the caller's own).
func (w Walk) checkOthers() []string {
	var same, cross bool
	var problems []string
	for _, o := range w.Others {
		same = same || (o.Tenant == w.Me.Tenant && o.Guardian != w.Me.Guardian)
		cross = cross || o.Tenant != w.Me.Tenant
		for k, id := range o.IDs {
			if w.Me.IDs[k] == id {
				problems = append(problems, fmt.Sprintf("%s shares %s id %d with %s", o.Name, k, id, w.Me.Name))
			}
		}
	}
	if !same {
		problems = append(problems, "Others needs a guardian in the caller's school")
	}
	if !cross {
		problems = append(problems, "Others needs a guardian in another school")
	}
	return problems
}

// swap is own with kind k replaced by the other guardian's.
func swap(own, other IDs, k Kind) (IDs, error) {
	if err := missing(own, []Kind{k}); err != nil {
		return nil, err
	}
	id, ok := other[k]
	if !ok {
		return nil, fmt.Errorf("no %s id", k)
	}
	out := maps.Clone(own)
	out[k] = id
	return out, nil
}

func missing(ids IDs, kinds []Kind) error {
	for _, k := range kinds {
		if _, ok := ids[k]; !ok {
			return fmt.Errorf("no %s id", k)
		}
	}
	return nil
}

// do sends one request as Me and returns the status and the error body's code.
func (w Walk) do(ctx context.Context, op Op, ids IDs) (int, string, error) {
	path := op.Path
	for param, k := range op.path {
		path = strings.ReplaceAll(path, "{"+param+"}", strconv.FormatInt(ids[k], 10))
	}
	q := url.Values{}
	for param, k := range op.query {
		q.Set(param, strconv.FormatInt(ids[k], 10))
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var body *bytes.Reader
	if b, ok := w.Bodies[op.ID]; ok {
		raw, err := json.Marshal(b.Make(ids))
		if err != nil {
			return 0, "", fmt.Errorf("sample body: %w", err)
		}
		body = bytes.NewReader(raw)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequestWithContext(ctx, op.Method, path, body)
	if body.Len() > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for h, vs := range w.Auth(w.Me) {
		req.Header[h] = vs
	}
	rec := httptest.NewRecorder()
	w.Handler.ServeHTTP(rec, req)
	var e struct {
		Code string `json:"code"`
	}
	if rec.Code >= 400 {
		_ = json.Unmarshal(rec.Body.Bytes(), &e)
	}
	return rec.Code, e.Code, nil
}
