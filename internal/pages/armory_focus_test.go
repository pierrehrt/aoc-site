package pages_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

var idAttr = regexp.MustCompile(` id="([^"]+)"`)

// AOC-071. Following a pill's ×, "clear all", a pager link or a sort link with the keyboard dropped
// focus to the page (measured in Chrome): htmx gives focus back after a swap only to an element with
// the id of the one that had it, and none had one. This pins the ids, by position for pills, on both
// answers (the page and htmx's update); a disabled or current pager control is a span htmx can focus;
// the removable pills name their fallback; and no id repeats, or focus would go to the wrong twin.
// Where focus then lands was measured in Chrome (the ticket's Log); this holds what makes it so.
func TestTheListsControlsKeepIdsFocusCanReturnTo(t *testing.T) {
	h := router(t)
	for _, hdr := range []map[string]string{nil, {"HX-Request": "true"}} {
		body := get(t, h, http.MethodGet, "/armory?rarity=epic&ilvl_min=1&p=2", hdr, "").Body.String()
		for _, want := range []string{
			`id="pill-0" data-pill data-refocus="#armory-chips [data-pill], #pills-clear" class=`,
			`id="pill-1" data-pill data-refocus="#armory-chips [data-pill], #pills-clear" class=`,
			`id="pills-clear" class=`,
			`hx-push-url="true" id="pager-prev" class=`, `hx-push-url="true" id="pager-next" class=`,
			`hx-push-url="true" id="pager-1" class=`, `<span aria-current="page" id="pager-2" tabindex="-1" class=`,
			`id="sort-toggle" class=`, `id="sort-col-name" class=`, `id="sort-col-ilvl" class=`,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("HX-Request %v: the page has no %s", hdr != nil, want)
			}
		}
		seen := map[string]int{}
		for _, m := range idAttr.FindAllStringSubmatch(body, -1) {
			seen[m[1]]++
		}
		for id, n := range seen {
			if n > 1 {
				t.Errorf("HX-Request %v: id %q appears %d times", hdr != nil, id, n)
			}
		}
	}
	// At the ends, the arrow that leads nowhere is a span with the same id, so focus stays on it.
	first := get(t, h, http.MethodGet, "/armory", nil, "").Body.String()
	if !strings.Contains(first, `<span id="pager-prev" tabindex="-1" aria-disabled="true"`) {
		t.Errorf("page 1: Prev is not a focusable span with its id")
	}
	last := get(t, h, http.MethodGet, "/armory?p=3", nil, "").Body.String()
	if !strings.Contains(last, `<span id="pager-next" tabindex="-1" aria-disabled="true"`) {
		t.Errorf("the last page: Next is not a focusable span with its id")
	}
}
