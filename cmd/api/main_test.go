package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
// re-route them as GET. This walks every GET route of the application as run() composes it, plus each
// mount's bare path, and asks the same thing of each: HEAD answers GET's status and Content-Type, with
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
		if len(p) > 1 && strings.HasSuffix(p, "/") {
			paths = append(paths, strings.TrimSuffix(p, "/")) // a mount's bare path: the one that broke
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
	// The walk must have seen the routes this ticket is about, or it proved nothing.
	for _, must := range []string{"/v1/items", "/v1/taxonomies", "/v1/items/x", "/health", "/armory"} {
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
