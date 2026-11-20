package db

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// A resolver error is answered with the standard error body, and the handler never runs (no pool is needed: the
// transaction is never opened).
func TestRequestTxResolverError(t *testing.T) {
	called := false
	h := RequestTx(nil, func(*http.Request) (Resolved, error) {
		return Resolved{}, httpx.NewError(http.StatusUnauthorized, "unauthenticated", "Sign in.")
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/x", nil))

	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), `"unauthenticated"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("handler ran without a tenant")
	}
}

// A resolver's Ctx must derive from the request's context; one that doesn't is a 500 before any transaction (no
// pool is needed), and the handler never runs.
func TestRequestTxResolvedCtxMustDerive(t *testing.T) {
	type k struct{}
	for name, tc := range map[string]struct {
		ctx  func(r *http.Request) context.Context
		want int
	}{
		"background":    {func(*http.Request) context.Context { return context.WithValue(context.Background(), k{}, 1) }, http.StatusInternalServerError},
		"other request": {func(*http.Request) context.Context { return httptest.NewRequest(http.MethodGet, "/", nil).Context() }, http.StatusInternalServerError},
	} {
		t.Run(name, func(t *testing.T) {
			called := false
			h := RequestTx(nil, func(r *http.Request) (Resolved, error) {
				return Resolved{TenantID: 1, Ctx: tc.ctx(r)}, nil
			})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/x", nil))
			if rec.Code != tc.want || called {
				t.Fatalf("got %d (handler ran: %t), want %d", rec.Code, called, tc.want)
			}
		})
	}
}

func TestBufferedResponse(t *testing.T) {
	t.Run("default status is 200", func(t *testing.T) {
		b := &bufferedResponse{header: http.Header{}}
		if b.status() != http.StatusOK {
			t.Fatalf("status %d", b.status())
		}
	})
	t.Run("1xx is ignored, first final status wins", func(t *testing.T) {
		b := &bufferedResponse{header: http.Header{}}
		b.WriteHeader(http.StatusEarlyHints)
		b.WriteHeader(http.StatusConflict)
		b.WriteHeader(http.StatusOK)
		if b.status() != http.StatusConflict {
			t.Fatalf("status %d", b.status())
		}
	})
	t.Run("flush copies headers, status and body", func(t *testing.T) {
		b := &bufferedResponse{header: http.Header{}}
		b.Header().Set("X-Test", "1")
		b.WriteHeader(http.StatusCreated)
		_, _ = b.Write([]byte("hello"))
		rec := httptest.NewRecorder()
		b.flushTo(rec)
		if rec.Code != http.StatusCreated || rec.Body.String() != "hello" || rec.Header().Get("X-Test") != "1" {
			t.Fatalf("got %d %q %v", rec.Code, rec.Body.String(), rec.Header())
		}
	})
	t.Run("oversize write is refused", func(t *testing.T) {
		b := &bufferedResponse{header: http.Header{}}
		if _, err := b.Write(make([]byte, MaxBufferedResponse+1)); err == nil || !b.tooLarge {
			t.Fatalf("want ErrResponseTooLarge, got %v (tooLarge=%t)", err, b.tooLarge)
		}
	})
}

func TestSafeMethod(t *testing.T) {
	for m, want := range map[string]bool{
		http.MethodGet: true, http.MethodHead: true, http.MethodOptions: true,
		http.MethodPost: false, http.MethodPut: false, http.MethodPatch: false, http.MethodDelete: false,
	} {
		if got := safeMethod(m); got != want {
			t.Errorf("safeMethod(%s) = %t, want %t", m, got, want)
		}
	}
}
