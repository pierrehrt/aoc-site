package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/pierrehrt/aoc-api/internal/assets"
	"github.com/pierrehrt/aoc-api/internal/httpx"
	"github.com/pierrehrt/aoc-api/internal/templates"
)

// failingQ answers no query: every call is a nil interface's, so it panics and the router's Recover
// answers 500. The walk below compares HEAD with GET, so it needs no data, only the same answer twice.
type failingQ struct{ querier }

var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// AOC-079. docs/api-routes.md says HEAD is answered on every route, and HEAD /v1/items and HEAD
// /v1/taxonomies were 405 (measured): chi's GetHead found the bare mount path's stub and did not
// re-route them as GET. This walks every GET route of the application as run() composes it, and every
// path prefix of one, with and without its trailing slash (so each mount's bare path: /v1, /v1/items,
// /v1/sources, …), and asks the same thing of each: HEAD answers GET's status and Content-Type, with
// no body. A route added to newHandler is covered without a line here.
//
// Through a real server, not a recorder: net/http is what drops a HEAD body, and a recorder keeps it.
func TestHeadAnswersWhatGetAnswersOnEveryRoute(t *testing.T) {
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // the panics below are the fake's
	t.Cleanup(func() { slog.SetDefault(prev) })

	set, err := assets.Load()
	if err != nil {
		t.Fatalf("assets.Load: %v", err)
	}
	tpl, err := templates.New(set)
	if err != nil {
		t.Fatalf("templates.New: %v", err)
	}
	h := newHandler(httpx.Build{Version: "test", Commit: "test", Env: "test"}, failingQ{}, tpl, set, "https://example.test")
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	var paths []string
	err = chi.Walk(h, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method != http.MethodGet {
			return nil
		}
		p := strings.ReplaceAll(routeParam.ReplaceAllString(route, "x"), "*", "x")
		paths = append(paths, p)
		// Every prefix, with and without its slash: a mount's bare path is where GetHead broke.
		for i := 1; i < len(p); i++ {
			if p[i] == '/' {
				paths = append(paths, p[:i], p[:i+1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	paths = unique(paths)
	// The walk must have seen the routes this ticket is about, or it proved nothing.
	for _, must := range []string{"/v1/items", "/v1/taxonomies", "/v1/items/x", "/health", "/armory",
		"/v1", "/v1/", "/v1/sources", "/v1/builds", "/v1/sources/", "/v1/builds/"} {
		found := false
		for _, p := range paths {
			found = found || p == must
		}
		if !found {
			t.Fatalf("the walk never reached %s (walked %d paths: %v)", must, len(paths), paths)
		}
	}

	for _, p := range paths {
		get := fetch(t, srv.URL+p, http.MethodGet)
		head := fetch(t, srv.URL+p, http.MethodHead)
		if head.status != get.status {
			t.Errorf("%s: HEAD %d, GET %d", p, head.status, get.status)
		}
		if head.contentType != get.contentType {
			t.Errorf("%s: HEAD Content-Type %q, GET %q", p, head.contentType, get.contentType)
		}
		if head.bodyLen != 0 {
			t.Errorf("%s: HEAD carried a %d-byte body", p, head.bodyLen)
		}
	}
}

type answer struct {
	status      int
	contentType string
	bodyLen     int
}

func fetch(t *testing.T, url, method string) answer {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return answer{resp.StatusCode, resp.Header.Get("Content-Type"), len(b)}
}

func unique(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// headAsGet routes every HEAD as GET, so a handler registered for HEAD itself would never be reached:
// a GET route with its own HEAD handler would answer with the GET one, and a HEAD-only route would be
// a 405. The walk above cannot see either (measured at AOC-079 verify round 1: both planted, it
// passed), so this refuses them by registration: on every route Routes() lists, at every level and a
// mount's own /* included, the HEAD handler it lists must be the GET handler it lists (as for the
// assets, mounted with r.Handle for every method), so what HEAD would have reached is what GET
// answers. Routes() lists the handlers as registered, an r.With or r.Group chain included; chi.Walk
// unwraps such a chain to its endpoint and would hide a HEAD's own middleware (verify round 3), and
// skips a mount's own route (round 4). Routes() hides only chi's own Mount stub, which nobody
// registers (read in chi v5.3.2's routes() at verify round 4). They are compared as chi
// v5.3.2 compares handlers (tree.go, equalHandlers): a func by its code pointer, anything else (a
// chain is a pointer) by ==. Its limit: two closures of one function literal, or one method on two
// receivers, count as one handler, so a HEAD handler built from GET's own literal or method passes.
func TestNoRouteHasAHeadHandlerOfItsOwn(t *testing.T) {
	set, err := assets.Load()
	if err != nil {
		t.Fatalf("assets.Load: %v", err)
	}
	tpl, err := templates.New(set)
	if err != nil {
		t.Fatalf("templates.New: %v", err)
	}
	h := newHandler(httpx.Build{Version: "test", Commit: "test", Env: "test"}, failingQ{}, tpl, set, "https://example.test")
	for _, b := range headHandlersOfTheirOwn(t, h) {
		t.Error(b)
	}
}

// The check above, against every way AOC-079's verify rounds planted a HEAD handler that nothing
// reaches: (a)–(e) passed the walk, (c) and (e) round 2's count of methods, (g)–(g3) round 3's
// chi.Walk endpoints, (h)–(h7) round 4's skipped mount route; (f) has handlers that are values rather
// than funcs (compared by ==). And against routes mounted for every method, with and without
// middleware, and mounted routers, which it must let through.
func TestTheHeadCheckRefusesEveryPlant(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	teapot := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Mw", "1")
			next.ServeHTTP(w, r)
		})
	}
	sub := func() chi.Router { s := chi.NewRouter(); s.Get("/", ok); s.Get("/x", ok); return s }
	var others []string // every method chi knows but HEAD, read from a probe route
	probe := chi.NewRouter()
	probe.Handle("/", ok)
	_ = chi.Walk(probe, func(method, _ string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method != http.MethodHead {
			others = append(others, method)
		}
		return nil
	})
	if len(others) < 2 {
		t.Fatalf("chi's every-method set, read from a probe route: %v", others)
	}
	plants := map[string]func(r chi.Router){
		"(a) GET and its own HEAD": func(r chi.Router) { r.Get("/zz", ok); r.Head("/zz", teapot) },
		"(b) HEAD only":            func(r chi.Router) { r.Head("/zz", teapot) },
		"(c) every method, then its own HEAD": func(r chi.Router) {
			r.Handle("/zz", ok)
			r.Head("/zz", teapot)
		},
		"(d) a mounted router's GET and its own HEAD": func(r chi.Router) {
			sub := chi.NewRouter()
			sub.Get("/", ok)
			sub.Head("/", teapot)
			r.Mount("/zz", sub)
		},
		"(e) each other method, and its own HEAD": func(r chi.Router) {
			for _, m := range others {
				r.Method(m, "/zz", ok)
			}
			r.Method(http.MethodHead, "/zz", teapot)
		},
		"(f) as (c), with handlers that are values, not funcs": func(r chi.Router) {
			r.Handle("/zz", http.RedirectHandler("/a", http.StatusFound))
			r.Method(http.MethodHead, "/zz", http.RedirectHandler("/b", http.StatusFound))
		},
		"(g) GET, and HEAD with its own middleware": func(r chi.Router) {
			r.Get("/zz", ok)
			r.With(mw).Head("/zz", ok)
		},
		"(g2) GET, and HEAD in a group with its own middleware": func(r chi.Router) {
			r.Get("/zz", ok)
			r.Group(func(g chi.Router) { g.Use(mw); g.Head("/zz", ok) })
		},
		"(g3) GET with its own middleware, and HEAD": func(r chi.Router) {
			r.With(mw).Get("/zz", ok)
			r.Head("/zz", ok)
		},
		"(h) a mounted router, then HEAD on its /*": func(r chi.Router) {
			r.Mount("/zz", sub())
			r.Head("/zz/*", teapot)
		},
		"(h6) a Route, then HEAD on its /*": func(r chi.Router) {
			r.Route("/zz", func(s chi.Router) { s.Get("/", ok) })
			r.Head("/zz/*", teapot)
		},
		"(h7) a mounted router, then HEAD on its /* with its own middleware": func(r chi.Router) {
			r.Mount("/zz", sub())
			r.With(mw).Head("/zz/*", ok)
		},
	}
	for name, plant := range plants {
		r := chi.NewRouter()
		plant(r)
		if len(headHandlersOfTheirOwn(t, r)) == 0 {
			t.Errorf("%s: not refused", name)
		}
	}
	controls := map[string]func(r chi.Router){
		"Handle":                 func(r chi.Router) { r.Handle("/zz", ok) },
		"With(mw).Handle":        func(r chi.Router) { r.With(mw).Handle("/zz", ok) },
		"Use(mw), then Handle":   func(r chi.Router) { r.Use(mw); r.Handle("/zz", ok) },
		"Group(Use(mw); Handle)": func(r chi.Router) { r.Group(func(g chi.Router) { g.Use(mw); g.Handle("/zz", ok) }) },
		"Mount(sub)":             func(r chi.Router) { r.Mount("/zz", sub()) },
		"With(mw).Mount(sub)":    func(r chi.Router) { r.With(mw).Mount("/zz", sub()) },
		"Route":                  func(r chi.Router) { r.Route("/zz", func(s chi.Router) { s.Get("/", ok) }) },
		"Mount(a plain handler)": func(r chi.Router) { r.Mount("/zz", ok) },
		"Group(Use(mw); Mount(sub))": func(r chi.Router) {
			r.Group(func(g chi.Router) { g.Use(mw); g.Mount("/zz", sub()) })
		},
	}
	for name, control := range controls {
		r := chi.NewRouter()
		control(r)
		if bad := headHandlersOfTheirOwn(t, r); len(bad) > 0 {
			t.Errorf("%s was refused: %v", name, bad)
		}
	}
}

// headHandlersOfTheirOwn lists every route Routes() lists, in r and every router mounted in it, a
// mount's own /* included, whose listed HEAD handler is not its listed GET handler. Read from
// Routes(), not chi.Walk: see above.
func headHandlersOfTheirOwn(t *testing.T, r chi.Routes) []string {
	t.Helper()
	var bad []string
	var visit func(prefix string, r chi.Routes)
	visit = func(prefix string, r chi.Routes) {
		for _, route := range r.Routes() {
			if head, ok := route.Handlers[http.MethodHead]; ok && !sameHandler(head, route.Handlers[http.MethodGet]) {
				bad = append(bad, fmt.Sprintf("%s%s has a HEAD handler that is not its GET handler: headAsGet routes HEAD as GET, so it is never reached", prefix, route.Pattern))
			}
			if route.SubRoutes != nil {
				visit(prefix+strings.TrimSuffix(route.Pattern, "/*"), route.SubRoutes)
			}
		}
	}
	visit("", r)
	return bad
}

// sameHandler is chi v5.3.2's equalHandlers (tree.go), which chi uses to tell its own stub apart.
func sameHandler(a, b http.Handler) bool {
	if a == nil || b == nil {
		return a == b
	}
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	if av.Type() != bv.Type() {
		return false
	}
	if av.Kind() == reflect.Func {
		return av.Pointer() == bv.Pointer()
	}
	if av.Type().Comparable() {
		return a == b
	}
	return false
}
