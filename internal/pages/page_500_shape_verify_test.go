package pages_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/pierrehrt/aoc-api/internal/assets"
	"github.com/pierrehrt/aoc-api/internal/builds"
	"github.com/pierrehrt/aoc-api/internal/db/sqlcgen"
	"github.com/pierrehrt/aoc-api/internal/httpx"
	"github.com/pierrehrt/aoc-api/internal/items"
	"github.com/pierrehrt/aoc-api/internal/pages"
	"github.com/pierrehrt/aoc-api/internal/templates"
)

// brokenItems is a database whose first query on each page either fails or panics: ListItems for
// the Armory, GetItemBySlug for an item page, ItemIDSpan for the sitemap index.
type brokenItems struct {
	*fakeItems
	panics bool
}

func (b brokenItems) broken(query string) error {
	if b.panics {
		panic("verify: " + query + " panicked")
	}
	return fmt.Errorf("verify: %s failed", query)
}

func (b brokenItems) ListItems(context.Context, sqlcgen.ListItemsParams) ([]sqlcgen.ListItemsRow, error) {
	return nil, b.broken("list items")
}

func (b brokenItems) GetItemBySlug(context.Context, string) (int32, error) {
	return 0, b.broken("get item by slug")
}

func (b brokenItems) ItemIDSpan(context.Context) (sqlcgen.ItemIDSpanRow, error) {
	return sqlcgen.ItemIDSpanRow{}, b.broken("item id span")
}

// AOC-080 verify round 1. docs/api-routes.md § Conventions says a page's 500 is not HTML: a failed
// query gets text/plain "internal server error" (pages.Handler.fail), to a browser and to htmx
// alike, and a panic gets the JSON 500 httpx.Recover writes. Until this test, only "a failed RENDER
// is a 500 without <html>" was pinned (TestAFailedRenderIsA500ThroughTheRouter); neither shape the
// doc names was. Both go through the production router, so Cache's no-store is checked as well.
func TestAPagesFiveHundredHasTheShapeTheDocNames(t *testing.T) {
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := templates.New(set)
	if err != nil {
		t.Fatal(err)
	}
	restore := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))
	defer slog.SetDefault(restore)

	paths := []string{"/armory", "/armory/test-item-1", "/sitemap.xml"}
	for _, panics := range []bool{false, true} {
		q := brokenItems{newFakeItems(10), panics}
		h := pages.New(tpl, set, base, items.NewService(q), builds.NewService(q))
		router := httpx.NewRouterWithSite(httpx.Build{Version: "1.2.3", Commit: "abc1234", Env: "test"}, h.Routes, set.Handler())
		for _, path := range paths {
			for _, htmx := range []bool{false, true} {
				hdr := map[string]string{}
				if htmx {
					hdr["HX-Request"] = "true"
				}
				rr := get(t, router, http.MethodGet, path, hdr, "")
				what := fmt.Sprintf("panic=%v %s HX-Request=%v", panics, path, htmx)
				if rr.Code != http.StatusInternalServerError {
					t.Errorf("%s: status %d, want 500", what, rr.Code)
					continue
				}
				if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
					t.Errorf("%s: Cache-Control %q, want no-store", what, cc)
				}
				ct := rr.Header().Get("Content-Type")
				if !panics {
					if ct != "text/plain; charset=utf-8" || rr.Body.String() != "internal server error\n" {
						t.Errorf("%s: %q %q, want text/plain \"internal server error\"", what, ct, rr.Body.String())
					}
					continue
				}
				var body map[string]string
				if ct != "application/json; charset=utf-8" || json.Unmarshal(rr.Body.Bytes(), &body) != nil ||
					len(body) != 2 || body["error"] != "internal error" || body["request_id"] == "" {
					t.Errorf("%s: %q %q, want Recover's JSON 500 {error, request_id}", what, ct, rr.Body.String())
				}
			}
		}
	}
}
