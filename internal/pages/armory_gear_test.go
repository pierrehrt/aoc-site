package pages_test

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/pierrehrt/aoc-api/internal/assets"
	"github.com/pierrehrt/aoc-api/internal/builds"
	"github.com/pierrehrt/aoc-api/internal/db/sqlcgen"
	"github.com/pierrehrt/aoc-api/internal/httpx"
	"github.com/pierrehrt/aoc-api/internal/items"
	"github.com/pierrehrt/aoc-api/internal/pages"
	"github.com/pierrehrt/aoc-api/internal/templates"
)

// The gear builder on the Armory page (AOC-051), over a fake corpus: item 1 goes on the head, item 2
// in the main hand, item 3 in either hand; items 4 onwards go in no slot. Item 1 is worn only by
// "test-other". Obviously fake (CLAUDE.md STEP ZERO).
func gearCorpus() *fakeItems {
	f := newFakeItems(10)
	f.slots = []sqlcgen.ListItemPageEquipLocationsRow{
		{ItemID: 1, Slug: "test-head", Name: "Test Head"},
		{ItemID: 2, Slug: "test-main", Name: "Test Main"},
		{ItemID: 3, Slug: "test-main", Name: "Test Main"}, {ItemID: 3, Slug: "test-off", Name: "Test Off"},
	}
	f.gearClasses = map[int32][]string{1: {"test-other"}}
	f.gearStats = map[int32][]sqlcgen.ListBuildItemStatsRow{
		1: {{ItemID: 1, Stat: "Test Might", Unit: "flat", Centi: 1000}, {ItemID: 1, Stat: "Test Grace", Unit: "flat", Centi: -150}},
		3: {{ItemID: 3, Stat: "Test Might", Unit: "flat", Centi: 250}},
	}
	return f
}

// gearRouter is the site and /v1 over one fake, as one database stands behind both in production.
func gearRouter(t *testing.T, q *fakeItems) http.Handler {
	t.Helper()
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := templates.New(set)
	if err != nil {
		t.Fatal(err)
	}
	gear := builds.NewService(q)
	return httpx.NewRouterWithAPI(httpx.Build{Version: "1.2.3", Commit: "abc1234", Env: "test"}, pages.New(tpl, set, base, items.NewService(q), gear).Routes, set.Handler(),
		func(v1 chi.Router) { v1.Mount("/builds", builds.NewHandler(gear).Routes()) })
}

// slotRow is one builder row's markup, by its slot.
func slotRow(t *testing.T, body, slot string) string {
	t.Helper()
	m := regexp.MustCompile(`(?s)<li data-gear-slot="` + slot + `" data-status="([a-z_]+)".*?</li>`).FindString(body)
	if m == "" {
		t.Fatalf("no builder row for %s", slot)
	}
	return m
}

// ⭐ THE CRITERION: with JavaScript off, a build's URL renders the whole builder — its slots, its
// items linked to their pages, its class, its sum and its share link.
func TestABuildRendersWholeWithoutJavaScript(t *testing.T) {
	body := get(t, gearRouter(t, gearCorpus()), http.MethodGet, "/armory?gear=test-head:1&gear=test-off:3&gear_class=test-other", nil, "").Body.String()
	for _, want := range []string{
		`<input type="checkbox" id="gear-open" class="sr-only max-lg:hidden" aria-label="Open the gear builder" checked>`, // open on load
		`<input type="hidden" name="gear" value="test-head:1">`, `<input type="hidden" name="gear" value="test-off:3">`,   // in the form
		`<option value="test-other" selected>Test Other · Test Archetype</option>`,
		`href="/armory/test-item-1"`, `href="/armory/test-item-3"`,
		`>&#43;12.5</span> <span class="text-[12.5px] text-paper-2">Test Might</span>`, // 10.00 + 2.50, exact (html/template writes + as &#43;)
		`>-1.5</span> <span class="text-[12.5px] text-paper-2">Test Grace</span>`,
		builds.SumNote,
		`href="` + base + `/armory?gear=test-head:1&amp;gear=test-off:3&amp;gear_class=test-other" data-gear-copy`, // the build alone
		`2/3 slots`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if row := slotRow(t, body, "test-head"); !strings.Contains(row, `data-status="equipped"`) {
		t.Errorf("head: %s", row)
	}
	if row := slotRow(t, body, "test-main"); !strings.Contains(row, `data-status="empty"`) {
		t.Errorf("main: %s", row)
	}
}

func TestEveryRuleShowsOnItsRow(t *testing.T) {
	body := get(t, gearRouter(t, gearCorpus()), http.MethodGet, "/armory?gear=test-head:99&gear=test-main:1&gear=test-off:3&gear_class=test-class", nil, "").Body.String()
	for slot, want := range map[string][]string{
		"test-head": {`data-status="unknown"`, "unknown item 99", `aria-label="Empty the Test Head slot"`},
		"test-main": {`data-status="wrong_slot"`, "⚠ does not go here"},
		"test-off":  {`data-status="equipped"`}, // item 3 lists no class: anyone wears it
	} {
		row := slotRow(t, body, slot)
		for _, w := range want {
			if !strings.Contains(row, w) {
				t.Errorf("%s row lacks %q:\n%s", slot, w, row)
			}
		}
	}
	body = get(t, gearRouter(t, gearCorpus()), http.MethodGet, "/armory?gear=test-head:1&gear_class=test-class", nil, "").Body.String()
	if row := slotRow(t, body, "test-head"); !strings.Contains(row, `data-status="conflict"`) || !strings.Contains(row, "⚠ not Test Class") {
		t.Errorf("a conflict is marked: %s", row)
	}
	if !strings.Contains(body, `>remove 1 conflicting</a>`) || !strings.Contains(body, `href="/armory?gear_class=test-class"`) {
		t.Error("remove-conflicting must link to the build without them, the class kept")
	}
	if strings.Contains(body, "Test Might") {
		t.Error("a conflicting item is summed")
	}
	// The list dims the rows the class cannot wear: item 1, worn only by test-other.
	if !regexp.MustCompile(`<tr data-row data-gear-id="1"[^>]*opacity-\[\.34\]`).MatchString(body) {
		t.Error("item 1 is not dimmed for test-class")
	}
	if regexp.MustCompile(`<tr data-row data-gear-id="3"[^>]*opacity-\[\.34\]`).MatchString(body) {
		t.Error("item 3 lists no class and is dimmed")
	}
}

// An add without a script is a 303 to the state it made: the address bar never keeps an action.
func TestAnAddIsA303ToTheStatesOwnURL(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	rr := get(t, h, http.MethodGet, "/armory?q=Test&add=1", nil, "")
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/armory?gear=test-head:1&q=Test" {
		t.Fatalf("add=1: %d → %q, want 303 → /armory?gear=test-head:1&q=Test", rr.Code, rr.Header().Get("Location"))
	}
	rr = get(t, h, http.MethodGet, "/armory?gear=test-head:1&add=test-off:3", nil, "")
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/armory?gear=test-head:1&gear=test-off:3" {
		t.Errorf("a drop on the off hand: %d → %q", rr.Code, rr.Header().Get("Location"))
	}
	// With htmx: the update, and the state's own URL pushed — never the one with add=.
	rr = get(t, h, http.MethodGet, "/armory?add=1", map[string]string{"HX-Request": "true", "HX-Current-URL": "http://x/armory"}, "")
	if rr.Code != http.StatusOK || rr.Header().Get("HX-Push-Url") != "/armory?gear=test-head:1" {
		t.Errorf("htmx add: %d, push %q", rr.Code, rr.Header().Get("HX-Push-Url"))
	}
	if !strings.Contains(rr.Body.String(), `hx-swap-oob="innerHTML:#gear-body"`) {
		t.Error("the update does not carry the builder")
	}
}

func TestARefusedAddSaysWhyAndChangesNothing(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	rr := get(t, h, http.MethodGet, "/armory?gear=test-head:1&add=test-off:1", nil, "")
	body := rr.Body.String()
	if rr.Code != http.StatusOK || !strings.Contains(body, `<p role="status"`) || !strings.Contains(body, "Test Item 1 does not go in the Test Off slot.") {
		t.Fatalf("%d, refusal shown: %v", rr.Code, strings.Contains(body, "does not go in"))
	}
	if !strings.Contains(body, `<input type="hidden" name="gear" value="test-head:1">`) || strings.Contains(body, `value="test-off:1"`) {
		t.Error("a refused add changed the build")
	}
	if !strings.Contains(body, `aria-label="Open the gear builder" checked>`) {
		t.Error("a refusal must open the builder, or nobody sees it")
	}
	// htmx: the same state, so the entry is replaced, not pushed — however the address orders its build.
	rr = get(t, h, http.MethodGet, "/armory?gear=test-off:3&gear=test-head:1&add=test-off:1", map[string]string{"HX-Request": "true", "HX-Current-URL": "http://x/armory?gear=test-off:3&gear=test-head:1"}, "")
	if rr.Header().Get("HX-Replace-Url") != "/armory?gear=test-head:1&gear=test-off:3" {
		t.Errorf("a hand-ordered build is another state: replace %q push %q", rr.Header().Get("HX-Replace-Url"), rr.Header().Get("HX-Push-Url"))
	}
	rr = get(t, h, http.MethodGet, "/armory?gear=test-head:1&add=test-off:1", map[string]string{"HX-Request": "true", "HX-Current-URL": "http://x/armory?gear=test-head:1"}, "")
	if rr.Header().Get("HX-Replace-Url") != "/armory?gear=test-head:1" || rr.Header().Get("HX-Push-Url") != "" {
		t.Errorf("refused htmx add: replace %q push %q", rr.Header().Get("HX-Replace-Url"), rr.Header().Get("HX-Push-Url"))
	}
}

// ⭐ Every link on the page carries the build, so no piece of it falls out of a pager, a sort, a
// pill or a tab — and the canonical never does: build URLs are not indexed apart from the list.
func TestTheBuildRidesOnEveryLinkButTheCanonical(t *testing.T) {
	body := get(t, gearRouter(t, newFakeItems(120)), http.MethodGet, "/armory?rarity=epic&gear=test-head:1", nil, "").Body.String()
	var links, without []string
	for _, m := range regexp.MustCompile(`href="(/armory\?[^"]*)"`).FindAllStringSubmatch(body, -1) {
		u := html.UnescapeString(m[1])
		links = append(links, u)
		// The builder's own clear and × empty the build and keep it (gear=); no other link drops it.
		if q, _ := url.ParseQuery(strings.SplitN(u, "?", 2)[1]); q.Get("rarity") == "epic" && !q.Has("gear") {
			without = append(without, u)
		}
	}
	if len(links) < 10 || len(without) > 0 {
		t.Errorf("%d links; these keep the filters and drop the build: %v", len(links), without)
	}
	if !strings.Contains(body, `<link rel="canonical" href="`+base+`/armory?rarity=epic">`) {
		t.Error("the canonical must be the list state without the build")
	}
	if !strings.Contains(body, "<title>Gear build — Armory</title>") {
		t.Error("a build's title names it")
	}
}

// The browser's id for a build rides in the form and on every link, and never in the share link or
// the canonical (verify round 2, F4).
func TestTheBuildsIDRidesOnTheLinksButNotTheShareLink(t *testing.T) {
	body := get(t, gearRouter(t, gearCorpus()), http.MethodGet, "/armory?gear=test-head:1&gear_id=abc123", nil, "").Body.String()
	if !strings.Contains(body, `<input type="hidden" name="gear_id" value="abc123">`) {
		t.Error("the id does not ride in the form")
	}
	// A "+" posts to the address it is on (AOC-101), so the id rides along in that address.
	if loc := get(t, gearRouter(t, gearCorpus()), http.MethodPost, "/armory?gear=test-head:1&gear_id=abc123", nil, "add=3").Header().Get("Location"); !strings.Contains(loc, "gear_id=abc123") {
		t.Errorf("a + must keep the id: → %q", loc)
	}
	if !strings.Contains(body, `href="`+base+`/armory?gear=test-head:1" data-gear-copy`) {
		t.Error("the share link must not carry the sender's id")
	}
	if !strings.Contains(body, `<link rel="canonical" href="`+base+`/armory">`) {
		t.Error("the canonical must not carry the build")
	}
	// "Any class" on a build of a class alone: the id keeps it a build, so nothing stale comes back (F6).
	rr := get(t, gearRouter(t, gearCorpus()), http.MethodGet, "/armory?gear_id=abc123&gear_class=", map[string]string{"HX-Request": "true", "HX-Current-URL": "http://x/armory?gear_class=test-class&gear_id=abc123"}, "")
	if push := rr.Header().Get("HX-Push-Url"); push != "/armory?gear_id=abc123" {
		t.Errorf("Any class pushes %q, want the build with its id and no class", push)
	}
}

// A state with no build is the list as it was: no builder in its URLs, the pane folded.
func TestAStateWithNoBuildIsTheListAsItWas(t *testing.T) {
	body := get(t, gearRouter(t, gearCorpus()), http.MethodGet, "/armory?rarity=epic", nil, "").Body.String()
	if strings.Contains(body, `name="gear"`) || strings.Contains(body, `aria-label="Open the gear builder" checked`) {
		t.Error("no build: no gear input, the builder folded")
	}
	for _, m := range regexp.MustCompile(`href="(/armory\?[^"]*)"`).FindAllStringSubmatch(body, -1) {
		if u := html.UnescapeString(m[1]); strings.Contains(u, "gear") && !strings.Contains(u, "add=") {
			t.Errorf("a list link names the builder: %s", u)
		}
	}
	if !strings.Contains(body, `<link rel="canonical" href="`+base+`/armory?rarity=epic">`) {
		t.Error("canonical changed")
	}
	// A filter change sends the form, class picker included, empty: still no build.
	rr := get(t, gearRouter(t, gearCorpus()), http.MethodGet, "/armory?rarity=epic&gear_class=", map[string]string{"HX-Request": "true", "HX-Current-URL": "http://x/armory"}, "")
	if push := rr.Header().Get("HX-Push-Url"); push != "/armory?rarity=epic" {
		t.Errorf("a filter change with no class picked pushes %q, want /armory?rarity=epic", push)
	}
}

// Each row that goes in a slot carries its "+" (a link, so it works without a script) and what the
// island reads to drag it; a row with no slot carries neither.
func TestRowsWithASlotCarryThePlusAndTheDragData(t *testing.T) {
	body := get(t, gearRouter(t, gearCorpus()), http.MethodGet, "/armory?gear=test-head:1", nil, "").Body.String()
	if !strings.Contains(body, `<tr data-row data-gear-id="3" data-gear-slots="test-main test-off"`) {
		t.Error("item 3's row lacks its drag data")
	}
	if !strings.Contains(body, `<button type="submit" form="gear-add" name="add" value="3" title="Add to the gear builder"`) {
		t.Error("item 3's + must be a button of the gear-add form, posting add=3")
	}
	if regexp.MustCompile(`data-gear-id="4"|name="add" value="4"`).MatchString(body) {
		t.Error("item 4 goes in no slot and has a + or drag data")
	}
}

// ⭐ AOC-101: no page offers an add as a URL. A "+" link was an action at its own address, and every
// answer was a build page with 50 more: crawlers walked them without end (up to 37 GB a day). The
// "+" is a button now, its add in the body of a POST, so nothing on a page names an add.
func TestNoPageOffersAnAddAsAURL(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	for _, path := range []string{"/armory", "/armory?gear=test-head:1", "/armory?gear=test-head:1&gear_class=test-other&gear_id=abc123&q=Test"} {
		body := get(t, h, http.MethodGet, path, nil, "").Body.String()
		if strings.Contains(body, "add=") {
			t.Errorf("%s: the page names an add=: %q", path, body[max(0, strings.Index(body, "add=")-120):strings.Index(body, "add=")+20])
		}
		// One "+" per row that goes in a slot (items 1, 2 and 3 of the corpus), and none for the rest.
		if n := strings.Count(body, `<button type="submit" form="gear-add" name="add" value="`); n != 3 {
			t.Errorf("%s: %d + buttons, want 3", path, n)
		}
		// One form, with no action (it posts to the address the reader is on), after the filter form:
		// forms do not nest, and the rows are inside the filter form.
		if n := strings.Count(body, `<form id="gear-add" method="post" hidden></form>`); n != 1 {
			t.Fatalf("%s: %d gear-add forms, want 1", path, n)
		}
		if !regexp.MustCompile(`(?s)</form>\s*<form id="gear-add"`).MatchString(body) || strings.Count(body, "<form") != 2 {
			t.Errorf("%s: the gear-add form must stand after the page's one filter form, not inside it", path)
		}
	}
}

// Without JavaScript the "+" posts add=<id> to the address it is on. The answer is the 303 a GET
// add on that address gives — the same state, at its own URL, so a reload never adds twice.
func TestAPostedAddIsA303ToTheSameStateAsAGetAdd(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	for _, c := range []struct{ state, add string }{
		{"/armory", "1"},
		{"/armory?q=Test", "1"},
		{"/armory?gear=test-head:1", "3"},
		{"/armory?gear=test-head:1", "test-off:3"},
		{"/armory?gear=test-head:1&gear_class=test-other&p=1&rarity=epic", "2"},
		{"/armory?gear=test-head:1&gear_id=abc123", "test-main:3"},
	} {
		sep := "?"
		if strings.Contains(c.state, "?") {
			sep = "&"
		}
		want := get(t, h, http.MethodGet, c.state+sep+"add="+url.QueryEscape(c.add), nil, "")
		got := get(t, h, http.MethodPost, c.state, nil, "add="+url.QueryEscape(c.add))
		if want.Code != http.StatusSeeOther || got.Code != http.StatusSeeOther || got.Header().Get("Location") != want.Header().Get("Location") {
			t.Errorf("POST %s add=%s: %d → %q; GET add: %d → %q", c.state, c.add, got.Code, got.Header().Get("Location"), want.Code, want.Header().Get("Location"))
		}
		if cc := got.Header().Get("Cache-Control"); cc != "no-store" {
			t.Errorf("POST %s: Cache-Control %q, want no-store", c.state, cc)
		}
	}
}

// A refused add says why, on POST as on GET, and changes nothing.
func TestARefusedPostedAddSaysWhy(t *testing.T) {
	rr := get(t, gearRouter(t, gearCorpus()), http.MethodPost, "/armory?gear=test-head:1", nil, "add=test-off:1")
	body := rr.Body.String()
	if rr.Code != http.StatusOK || !strings.Contains(body, "Test Item 1 does not go in the Test Off slot.") {
		t.Fatalf("%d, refusal shown: %v", rr.Code, strings.Contains(body, "does not go in"))
	}
	if !strings.Contains(body, `<input type="hidden" name="gear" value="test-head:1">`) || strings.Contains(body, `value="test-off:1"`) {
		t.Error("a refused add changed the build")
	}
	if cc := rr.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control %q, want no-store", cc)
	}
}

// A POST exists only to add. One that cannot be read, or has nothing to add, or a bad add, is a 400
// naming it, never a 500. The add is the body's: one in a POST's query string is not read.
func TestABadPostedAddIsA400(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	for name, body := range map[string]string{
		"no add":           "other=1",
		"an empty add":     "add=",
		"not a number":     "add=abc",
		"a NUL byte":       "add=%00",
		"invalid UTF-8":    "add=%FF",
		"a bad slot":       "add=Bad%20Slot:1",
		"a body over 1KiB": "add=1&pad=" + strings.Repeat("x", 1<<10),
	} {
		rr := get(t, h, http.MethodPost, "/armory?gear=test-head:1", nil, body)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", name, rr.Code)
		}
	}
	rr := get(t, h, http.MethodPost, "/armory?gear=test-head:1&add=3", nil, "other=1")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("an add in a POST's query string was read: %d → %q", rr.Code, rr.Header().Get("Location"))
	}
	rr = get(t, h, http.MethodPost, "/armory?add=test-off:1", nil, "add=1")
	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/armory?gear=test-head:1" {
		t.Errorf("the body's add must be the one done: %d → %q", rr.Code, rr.Header().Get("Location"))
	}
}

func TestAMalformedBuildIsA400(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	for _, q := range []string{"gear=test-head", "gear=test-head:x", "gear=test-elbow:1", "gear_class=test-nobody", "gear=test-head:1&gear=test-head:2", "add=x", "add=test-elbow:1"} {
		if rr := get(t, h, http.MethodGet, "/armory?"+q, nil, ""); rr.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", q, rr.Code)
		}
	}
}

// ⭐ THE CRITERION: /v1/builds/compute answers what the page shows — every slot's status and every
// line of the sum — for the same parameters.
func TestThePageAndTheJSONShowTheSameBuild(t *testing.T) {
	h := gearRouter(t, gearCorpus())
	for _, q := range []string{"gear=", "gear=test-head:1&gear=test-off:3", "gear=test-main:1&gear=test-off:3&gear_class=test-class", "gear=test-head:1&gear_class=test-other"} {
		var res builds.Result
		rr := get(t, h, http.MethodGet, "/v1/builds/compute?"+q, nil, "")
		if rr.Code != http.StatusOK {
			t.Fatalf("%s: /v1 %d", q, rr.Code)
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
			t.Fatal(err)
		}
		body := get(t, h, http.MethodGet, "/armory?"+q, nil, "").Body.String()
		for _, s := range res.Slots {
			if row := slotRow(t, body, s.Slot.Slug); !strings.Contains(row, `data-status="`+s.Status+`"`) {
				t.Errorf("%s %s: /v1 says %s, the page %s", q, s.Slot.Slug, s.Status, row[:80])
			}
		}
		var page []string
		for _, m := range regexp.MustCompile(`<li class="grid grid-cols-\[66px_1fr\][^"]*"><span[^>]*>([^<]+)</span> <span[^>]*>([^<]+)</span>`).FindAllStringSubmatch(body, -1) {
			page = append(page, html.UnescapeString(m[1]+" "+m[2]))
		}
		var api []string
		for _, s := range res.Stats {
			api = append(api, templates.GearValue(s)+" "+templates.GearLabel(s))
		}
		if strings.Join(page, " | ") != strings.Join(api, " | ") {
			t.Errorf("%s: the page sums %v, /v1 %v", q, page, api)
		}
	}
	if rr := get(t, h, http.MethodGet, "/v1/builds/compute?gear=test-head:99", nil, ""); rr.Code != http.StatusBadRequest {
		t.Errorf("an unknown id on /v1: %d, want 400", rr.Code)
	}
}
