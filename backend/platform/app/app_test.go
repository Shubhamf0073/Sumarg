package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
)

func baseCfg(name string) func() (config.Base, error) {
	return func() (config.Base, error) {
		return config.LoadFrom[config.Base](map[string]string{
			"APP_ENV": "local", "SVC_NAME": name, "DATABASE_URL": "postgres://u:hunter2@localhost:5432/db",
		})
	}
}

// freeAddr returns a 127.0.0.1 port nothing is listening on (it was free a moment ago).
func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func testService() Service {
	return Service{
		Name: "schoolfleet", PublicAddr: "127.0.0.1:0", InternalAddr: "127.0.0.1:0",
		Modes: []string{ModeAPI, ModeWorker, ModeAll},
		Public: func(r chi.Router) {
			r.Get("/api/healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "ok") })
		},
	}
}

// syncBuffer is a bytes.Buffer safe to write from the server goroutines and read from the test.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *syncBuffer) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

type started struct {
	addrs map[string]string
	exit  chan int
	out   *syncBuffer
	stop  context.CancelFunc
}

// start runs `serve args...` in the background and waits until every expected listener is up.
func start(t *testing.T, s Service, want []string, args ...string) *started {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	var mu sync.Mutex
	st := &started{addrs: map[string]string{}, exit: make(chan int, 1), out: &syncBuffer{}, stop: cancel}
	up := make(chan struct{}, 4)
	go func() {
		st.exit <- run(ctx, s, append([]string{"serve"}, args...), st.out, st.out, baseCfg(s.Name),
			func(name, addr string) { mu.Lock(); st.addrs[name] = addr; mu.Unlock(); up <- struct{}{} })
	}()
	for range want {
		select {
		case <-up:
		case code := <-st.exit:
			t.Fatalf("serve exited with %d before listening:\n%s", code, st.out)
		case <-time.After(5 * time.Second):
			t.Fatalf("serve did not start:\n%s", st.out)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	for _, name := range want {
		if st.addrs[name] == "" {
			t.Fatalf("listener %q not started: %v", name, st.addrs)
		}
	}
	return st
}

func (st *started) shutdown(t *testing.T) {
	t.Helper()
	st.stop()
	select {
	case code := <-st.exit:
		if code != 0 {
			t.Fatalf("exit %d after shutdown:\n%s", code, st.out)
		}
	case <-time.After(ShutdownGrace + time.Second):
		t.Fatal("serve did not shut down")
	}
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestServeAndHealthcheck(t *testing.T) {
	st := start(t, testService(), []string{"internal", "public"})

	if code, body := get(t, "http://"+st.addrs["public"]+"/api/healthz"); code != 200 || body != "ok" {
		t.Fatalf("public /api/healthz: %d %q", code, body)
	}
	if code, _ := get(t, "http://"+st.addrs["public"]+"/readyz"); code != 404 {
		t.Fatalf("readiness must not be on the public listener: %d", code)
	}
	if code, _ := get(t, "http://"+st.addrs["internal"]+"/api/healthz"); code != 404 {
		t.Fatalf("public routes must not be on the internal listener: %d", code)
	}
	if code := run(t.Context(), Service{Name: "schoolfleet", InternalAddr: st.addrs["internal"]}, []string{"healthcheck"},
		io.Discard, io.Discard, nil, nil); code != 0 {
		t.Fatalf("healthcheck against a running service exited %d", code)
	}

	st.shutdown(t)
	out := st.out.String()
	if strings.Contains(out, "hunter2") {
		t.Fatalf("database password in the logs:\n%s", out)
	}
	if !strings.Contains(out, `"service":"schoolfleet"`) || !strings.Contains(out, "shut down cleanly") {
		t.Fatalf("unexpected logs:\n%s", out)
	}
}

func TestWorkerModeHasNoPublicListener(t *testing.T) {
	st := start(t, testService(), []string{"internal"}, "--mode=worker")
	if st.addrs["public"] != "" {
		t.Fatalf("worker mode opened a public listener at %s", st.addrs["public"])
	}
	st.shutdown(t)
}

func TestNoPublicAddrMeansInternalOnly(t *testing.T) {
	s := Service{Name: "migrate", InternalAddr: "127.0.0.1:0"}
	st := start(t, s, []string{"internal"})
	if len(st.addrs) != 1 {
		t.Fatalf("listeners: %v", st.addrs)
	}
	st.shutdown(t)
}

func TestHealthcheckFails(t *testing.T) {
	// Nothing listening.
	if code := run(t.Context(), Service{Name: "schoolfleet", InternalAddr: freeAddr(t)}, []string{"healthcheck"},
		io.Discard, io.Discard, nil, nil); code != 1 {
		t.Fatalf("exit %d with nothing listening, want 1", code)
	}
	// Listening but not ready, and a redirect is not followed.
	for _, h := range []http.HandlerFunc{
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
		},
		func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/healthz", http.StatusFound) },
	} {
		srv := httptest.NewServer(h)
		_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
		err := Healthcheck(t.Context(), ":"+port)
		srv.Close()
		if err == nil {
			t.Fatal("healthcheck passed against a service that isn't ready")
		}
	}
}

func TestHealthcheckTimesOut(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-block:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(block)
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	start := time.Now()
	err := Healthcheck(t.Context(), ":"+port)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > HealthcheckTimeout+time.Second {
		t.Fatalf("err %v after %v", err, time.Since(start))
	}
}

func TestServeRefusesBadStarts(t *testing.T) {
	s := testService()
	cases := []struct {
		name string
		args []string
		load func() (config.Base, error)
		want int
		msg  string
	}{
		{"no command", nil, baseCfg("schoolfleet"), 2, "usage"},
		{"unknown command", []string{"migrate"}, baseCfg("schoolfleet"), 2, "usage"},
		{"bad mode", []string{"serve", "--mode=everything"}, baseCfg("schoolfleet"), 2, "--mode must be"},
		{"extra arg", []string{"serve", "now"}, baseCfg("schoolfleet"), 2, "unexpected argument"},
		{"wrong env file", []string{"serve"}, baseCfg("migrate"), 1, "wrong env file"},
		{"config error", []string{"serve"}, func() (config.Base, error) {
			return config.LoadFrom[config.Base](map[string]string{"APP_ENV": "local", "SVC_NAME": "schoolfleet"})
		}, 1, "DATABASE_URL"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			if code := run(t.Context(), s, c.args, &out, &out, c.load, nil); code != c.want ||
				!strings.Contains(out.String(), c.msg) {
				t.Fatalf("exit %d, want %d; output %q", code, c.want, out.String())
			}
		})
	}
}

func TestPortInUseFailsAndStopsOthers(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	s := testService()
	s.PublicAddr = busy.Addr().String()
	var out syncBuffer
	done := make(chan int, 1)
	go func() { done <- run(t.Context(), s, []string{"serve"}, &out, &out, baseCfg("schoolfleet"), nil) }()
	select {
	case code := <-done:
		if code != 1 || !strings.Contains(out.String(), "listen failed") {
			t.Fatalf("exit %d:\n%s", code, out.String())
		}
	case <-time.After(ShutdownGrace + time.Second):
		t.Fatal("serve hung after a listen failure")
	}
}
