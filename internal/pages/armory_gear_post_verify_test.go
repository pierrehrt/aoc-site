package pages_test

import (
	"net/http"
	"strings"
	"testing"
)

// AOC-101 verify round 1: what the build's own tests did not reach. TestNoPageOffersAnAddAsAURL reads
// three full GET pages; the answers htmx swaps in, a refusal and a POST's own page are read here.
// TestABadPostedAddIsA400 checks the status, not the cache header. The cap is checked one byte past
// it, not at it. And a body that is not a form was not sent at all.

// No answer the builder can give names an add: not the fragments htmx swaps in, not a refusal, not a
// POST's page. Any one of them would be a "+" a link-follower could walk again.
func TestNoAnswerOfTheBuilderNamesAnAdd(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	hx := map[string]string{"HX-Request": "true"}
	for _, c := range []struct {
		name, method, path, body string
		hdr                      map[string]string
		want                     int
	}{
		{"the htmx fragment, no build", http.MethodGet, "/armory", "", hx, http.StatusOK},
		{"the htmx fragment, a build", http.MethodGet, "/armory?gear=test-head:1&gear_class=test-other", "", hx, http.StatusOK},
		{"an htmx add's answer", http.MethodGet, "/armory?gear=&add=3", "", hx, http.StatusOK},
		{"an htmx refused add", http.MethodGet, "/armory?gear=test-head:1&add=test-off:1", "", hx, http.StatusOK},
		{"a plain refused GET add", http.MethodGet, "/armory?gear=test-head:1&add=test-off:1", "", nil, http.StatusOK},
		{"a refused POST", http.MethodPost, "/armory?gear=test-head:1", "add=test-off:1", nil, http.StatusOK},
		{"a refused POST over htmx", http.MethodPost, "/armory?gear=test-head:1", "add=test-off:1", hx, http.StatusOK},
		{"a POST over htmx", http.MethodPost, "/armory?gear=", "add=3", hx, http.StatusOK},
		{"a bad POST over htmx", http.MethodPost, "/armory?gear=", "add=abc", hx, http.StatusBadRequest},
	} {
		rr := get(t, h, c.method, c.path, c.hdr, c.body)
		body := rr.Body.String()
		if rr.Code != c.want {
			t.Errorf("%s: %d, want %d", c.name, rr.Code, c.want)
		}
		if i := strings.Index(body, "add="); i >= 0 {
			t.Errorf("%s: the answer names an add=: %q", c.name, body[max(0, i-120):i+20])
		}
	}
}

// Every answer to a POST is no-store, the 400s included: an answer to a write is about that request.
func TestEveryAnswerToAPostIsNoStore(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	hx := map[string]string{"HX-Request": "true"}
	for _, c := range []struct {
		name, body string
		hdr        map[string]string
		want       int
	}{
		{"added", "add=3", nil, http.StatusSeeOther},
		{"refused", "add=test-off:1", nil, http.StatusOK},
		{"nothing to add", "other=1", nil, http.StatusBadRequest},
		{"a bad add", "add=abc", nil, http.StatusBadRequest},
		{"over the cap", "add=3&pad=" + strings.Repeat("x", 1<<10), nil, http.StatusBadRequest},
		{"added over htmx", "add=3", hx, http.StatusOK},
		{"a bad add over htmx", "add=abc", hx, http.StatusBadRequest},
	} {
		rr := get(t, h, http.MethodPost, "/armory?gear=test-head:1", c.hdr, c.body)
		if rr.Code != c.want || rr.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: %d with Cache-Control %q, want %d with no-store", c.name, rr.Code, rr.Header().Get("Cache-Control"), c.want)
		}
	}
}

// The cap is 1 KiB exactly: a body of 1024 bytes is read, one of 1025 is not.
func TestThePostedAddsCapIsOneKiBExactly(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	at := "add=3&pad=" + strings.Repeat("x", 1024-len("add=3&pad="))
	if rr := get(t, h, http.MethodPost, "/armory?gear=", nil, at); len(at) != 1024 || rr.Code != http.StatusSeeOther {
		t.Errorf("a %d-byte body: %d, want 303", len(at), rr.Code)
	}
	if rr := get(t, h, http.MethodPost, "/armory?gear=", nil, at+"x"); rr.Code != http.StatusBadRequest {
		t.Errorf("a %d-byte body: %d, want 400", len(at)+1, rr.Code)
	}
}

// Only a form's body is read. A body that is not application/x-www-form-urlencoded has no add in it
// for this handler, whatever its bytes say: a 400, never a 303 and never a 500.
func TestAPostThatIsNotAFormIsA400(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	for name, c := range map[string][2]string{
		"multipart":        {"multipart/form-data; boundary=X", "--X\r\nContent-Disposition: form-data; name=\"add\"\r\n\r\n3\r\n--X--\r\n"},
		"text":             {"text/plain", "add=3"},
		"json":             {"application/json", `{"add":"3"}`},
		"no type":          {"", "add=3"},
		"a malformed type": {"application/x-www-form-urlencoded; charset", "add=3"},
	} {
		rr := get(t, h, http.MethodPost, "/armory?gear=", map[string]string{"Content-Type": c[0]}, c[1])
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: %d → %q, want 400", name, rr.Code, rr.Header().Get("Location"))
		}
	}
}

// A "+" on page 2 lands on page 2, as a GET add on page 2 does.
func TestAPostedAddKeepsItsPage(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	g := get(t, h, http.MethodGet, "/armory?gear=&p=2&add=3", nil, "")
	p := get(t, h, http.MethodPost, "/armory?gear=&p=2", nil, "add=3")
	if p.Code != http.StatusSeeOther || p.Header().Get("Location") != g.Header().Get("Location") || !strings.Contains(p.Header().Get("Location"), "p=2") {
		t.Errorf("POST: %d → %q; GET add: %d → %q", p.Code, p.Header().Get("Location"), g.Code, g.Header().Get("Location"))
	}
}

// ❌ at verify round 1. The "+" posts to the address the reader is on, and the state is read from that
// address as a GET reads it: a pair url.ParseQuery drops (one holding ';', or a malformed escape) is
// read by nothing and answers as if absent (items.parseFilters). The page at such an address is a
// 200, and a GET add there is a 303. The POST must add there too. r.ParseForm parses the URL's query
// as well as the body and returns the query's error, so these were 400 "That form could not be read".
func TestAPostOnAnAddressTheGetReadsAddsAsTheGetDoes(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	for _, state := range []string{"/armory?gear=test-head:1&x=1;y=2", "/armory?gear=test-head:1&q=100%", "/armory?gear=test-head:1&zz=%zz"} {
		page := get(t, h, http.MethodGet, state, nil, "")
		g := get(t, h, http.MethodGet, state+"&add=3", nil, "")
		p := get(t, h, http.MethodPost, state, nil, "add=3")
		if page.Code != http.StatusOK || g.Code != http.StatusSeeOther {
			t.Fatalf("%s: the page is %d and a GET add %d; this test assumes 200 and 303", state, page.Code, g.Code)
		}
		if p.Code != g.Code || p.Header().Get("Location") != g.Header().Get("Location") {
			t.Errorf("%s: POST add=3 → %d %q; GET …&add=3 → %d %q", state, p.Code, p.Header().Get("Location"), g.Code, g.Header().Get("Location"))
		}
	}
}
