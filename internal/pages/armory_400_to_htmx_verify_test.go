package pages_test

import (
	"net/http"
	"strings"
	"testing"
)

// AOC-080 verify round 2. docs/api-routes.md § Conventions: a page's own 400 is a small HTML page;
// to htmx, a refused search or build is the armory_invalid fragment instead (invalidSearch), while
// a bad page number is the small 400 page to htmx too (httpx.RejectHTML). The fragment was pinned
// (TestALiveRequestWithABadRangeSaysWhy, TestAServiceRefusalIsA400OnTheArmory); what a bad page
// number answers to htmx was not: TestArmoryRejectsBadInputHonestly sends no HX-Request.
func TestToHTMXARefusedSearchIsTheFragmentAndABadPageNumberIsThePage(t *testing.T) {
	h := router(t)
	htmx := map[string]string{"HX-Request": "true"}

	isSmallPage := func(t *testing.T, label, body string) {
		t.Helper()
		if !strings.HasPrefix(body, "<!DOCTYPE html>") || !strings.Contains(body, "<title>400 — AoC Codex</title>") {
			t.Errorf("%s: not the small 400 page: %.120q", label, body)
		}
		if strings.Contains(body, `role="alert"`) {
			t.Errorf("%s: the small page carries the armory_invalid fragment's alert", label)
		}
	}

	// The doc's own examples of a refused search or build: the parser's refusals.
	for _, path := range []string{"/armory?sort=bogus", "/armory?ilvl_min=80&ilvl_max=60", "/armory?gear=bogus"} {
		rr := get(t, h, http.MethodGet, path, htmx, "")
		body := rr.Body.String()
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s to htmx: status %d, want 400", path, rr.Code)
		}
		if strings.Contains(body, "<html") || !strings.Contains(body, `role="alert"`) || !strings.Contains(body, "That search is not valid.") {
			t.Errorf("%s to htmx: not the armory_invalid fragment: %.120q", path, body)
		}
		if got := rr.Header().Get("HX-Push-Url"); got != "false" {
			t.Errorf("%s to htmx: HX-Push-Url = %q, want \"false\"", path, got)
		}
		// To a browser, the same refusal is the small page.
		rr = get(t, h, http.MethodGet, path, nil, "")
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s to a browser: status %d, want 400", path, rr.Code)
		}
		isSmallPage(t, path+" to a browser", rr.Body.String())
	}

	// A bad page number: the small page whoever asks, and no URL instruction for htmx.
	for _, path := range []string{"/armory?p=0", "/armory?p=abc", "/armory?p=-1"} {
		for _, hdr := range []map[string]string{nil, htmx} {
			label := path + " to a browser"
			if hdr != nil {
				label = path + " to htmx"
			}
			rr := get(t, h, http.MethodGet, path, hdr, "")
			if rr.Code != http.StatusBadRequest {
				t.Errorf("%s: status %d, want 400", label, rr.Code)
			}
			if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
				t.Errorf("%s: Content-Type %q, want text/html", label, ct)
			}
			isSmallPage(t, label, rr.Body.String())
			if !strings.Contains(rr.Body.String(), "That page number is not valid") {
				t.Errorf("%s: the page does not say the page number is the problem", label)
			}
			if got, ok := rr.Header()["Hx-Push-Url"]; ok {
				t.Errorf("%s: HX-Push-Url %q set; only the fragment sets it", label, got)
			}
		}
	}
}
