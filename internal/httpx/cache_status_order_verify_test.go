package httpx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pierrehrt/aoc-api/internal/httpx"
)

// AOC-080 verify round 2. docs/api-routes.md § Conventions: "a write or an HTMX request is
// `no-store` whatever its status, a 404 included: `cachePolicy` decides those before the status".
// TestCachePolicy pins writes and HTMX on a 200 only, so a policy that read the status first (a
// 404 a minute, a 301 the page's hour) passed it. Every status that has its own rule is tried here,
// on each path class that has one.
func TestAWriteOrAnHTMXRequestIsNoStoreWhateverItsStatus(t *testing.T) {
	type req struct {
		method string
		htmx   bool
	}
	reqs := []req{
		{http.MethodPost, false}, {http.MethodPut, false}, {http.MethodDelete, false},
		{http.MethodPatch, false}, {http.MethodOptions, false},
		{http.MethodGet, true}, {http.MethodHead, true}, {http.MethodPost, true},
	}
	statuses := []int{
		http.StatusOK, http.StatusMovedPermanently, http.StatusNotModified, http.StatusPermanentRedirect,
		http.StatusNotFound, http.StatusGone, http.StatusBadRequest, http.StatusInternalServerError,
	}
	paths := []string{"/", "/items/test-item-alpha", "/v1/items", "/assets/app.0123abcd.css"}
	for _, rq := range reqs {
		for _, status := range statuses {
			for _, path := range paths {
				h := httpx.Cache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
				r := httptest.NewRequestWithContext(context.Background(), rq.method, path, nil)
				if rq.htmx {
					r.Header.Set("HX-Request", "true")
				}
				rr := httptest.NewRecorder()
				h.ServeHTTP(rr, r)
				if got := rr.Result().Header.Values("Cache-Control"); len(got) != 1 || got[0] != wantNoStore {
					t.Errorf("%s %s (htmx %v) answered %d: Cache-Control = %q, want exactly %q",
						rq.method, path, rq.htmx, status, got, wantNoStore)
				}
			}
		}
	}
}

// The same through the production router, on the two cases the doc names: an htmx 404 and a POST
// 404, beside the GET 404 that keeps its minute.
func TestTheRoutersFourOhFourIsNoStoreToAWriteAndToHTMX(t *testing.T) {
	site := func(r chi.Router) {
		r.Get("/test-page", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("test page")) })
	}
	r := httpx.NewRouterWithSite(httpx.Build{Version: "dev", Commit: "none", Env: "test"}, site, nil)
	for _, tc := range []struct {
		method string
		htmx   bool
		want   string
	}{
		{http.MethodGet, false, wantMissing},
		{http.MethodGet, true, wantNoStore},
		{http.MethodPost, false, wantNoStore},
	} {
		for _, path := range []string{"/no-such-page", "/v1/no-such-thing"} {
			req := httptest.NewRequestWithContext(context.Background(), tc.method, path, nil)
			if tc.htmx {
				req.Header.Set("HX-Request", "true")
			}
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Errorf("%s %s (htmx %v): status %d, want 404", tc.method, path, tc.htmx, rr.Code)
			}
			if got := rr.Result().Header.Get("Cache-Control"); got != tc.want {
				t.Errorf("%s %s (htmx %v): Cache-Control = %q, want %q", tc.method, path, tc.htmx, got, tc.want)
			}
		}
	}
}
