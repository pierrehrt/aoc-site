package pages_test

import (
	"net/http"
	"testing"

	"github.com/pierrehrt/aoc-api/internal/assets"
	"github.com/pierrehrt/aoc-api/internal/builds"
	"github.com/pierrehrt/aoc-api/internal/httpx"
	"github.com/pierrehrt/aoc-api/internal/items"
	"github.com/pierrehrt/aoc-api/internal/pages"
	"github.com/pierrehrt/aoc-api/internal/templates"
)

// AOC-069 on the page: a NUL byte or invalid UTF-8 in a filter is the page's 400, as plain HTML and
// as the fragment to htmx with no URL pushed, and in an item's address a 404. Both were a 500 and an
// ERROR line ("page render failed", measured). The rule is the parser's and the service's, shared
// with /v1 (internal/items/text_test.go); this pins that the page answers with it.
func TestAByteNoTextCanHoldIsA400OnTheArmoryAndA404ForAnItem(t *testing.T) {
	set, err := assets.Load()
	if err != nil {
		t.Fatalf("assets.Load: %v", err)
	}
	tpl, err := templates.New(set)
	if err != nil {
		t.Fatalf("templates.New: %v", err)
	}
	q := newFakeItems(3)
	site := pages.New(tpl, set, base, items.NewService(q), builds.NewService(q))
	h := httpx.NewRouterWithSite(httpx.Build{Version: "1.2.3", Commit: "abc1234", Env: "test"}, site.Routes, set.Handler())
	for _, bad := range []string{"%00", "%FF"} {
		for _, hdr := range []map[string]string{nil, {"HX-Request": "true"}} {
			rr := get(t, h, http.MethodGet, "/armory?q=a"+bad+"b", hdr, "")
			if rr.Code != http.StatusBadRequest {
				t.Errorf("q=a%sb, HX-Request %v: status %d, want 400", bad, hdr != nil, rr.Code)
				continue
			}
			if hdr != nil && rr.Header().Get("HX-Push-Url") != "false" {
				t.Errorf("q=a%sb, HX-Request: HX-Push-Url = %q, want \"false\"", bad, rr.Header().Get("HX-Push-Url"))
			}
		}
		if rr := get(t, h, http.MethodGet, "/armory/a"+bad+"b", nil, ""); rr.Code != http.StatusNotFound {
			t.Errorf("/armory/a%sb: status %d, want 404", bad, rr.Code)
		}
	}
}
