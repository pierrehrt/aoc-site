package items

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/pierrehrt/aoc-api/internal/httpx"
)

// AOC-069. A NUL byte or invalid UTF-8 in a filter reached Postgres, which refuses both as text
// (`invalid byte sequence for encoding "UTF8": 0x00`, SQLSTATE 22021, measured), and the request
// answered 500 with an ERROR line: 11 text parameters on /v1/items, /v1/sources/tree and /armory.
// It is a malformed request, so a 400, and the database is never asked.
//
// The list names every parameter the parser reads, and one it does not: the check is on the whole
// query string, so a filter added later is covered without a line of its own.
func TestAByteNoTextCanHoldIsA400OnEveryParameter(t *testing.T) {
	params := []string{"rarity", "equip_location", "armour_weight", "class", "currency", "set",
		"item_type", "region", "tier", "q", "sort", "place", "tab", "source", "get", "pvp", "unchained",
		"price", "ilvl_min", "ilvl_max", "reqlvl_min", "reqlvl_max", "facets", "limit", "offset",
		"not_a_filter"}
	// NUL; a byte that starts no UTF-8 sequence; an overlong "/"; a lone surrogate half.
	for _, bad := range []string{"%00", "%FF", "%C0%AF", "%ED%A0%80"} {
		for _, p := range params {
			q := &fakeQ{}
			rec, body := get(t, q, "/v1/items?"+p+"=a"+bad+"b")
			if rec.Code != http.StatusBadRequest || body["error"] == nil {
				t.Errorf("%s=a%sb -> %d %v, want 400 with a JSON error", p, bad, rec.Code, body)
			}
			if q.calls != 0 || q.facetCalled != 0 {
				t.Errorf("%s=a%sb reached the database (%d list calls, %d facet calls)", p, bad, q.calls, q.facetCalled)
			}
		}
		// In a parameter's name too.
		q := &fakeQ{}
		if rec, _ := get(t, q, "/v1/items?a"+bad+"b=1"); rec.Code != http.StatusBadRequest || q.calls != 0 {
			t.Errorf("a parameter NAME holding %s -> %d after %d list calls, want 400 and none", bad, rec.Code, q.calls)
		}
	}
	// Text beyond ASCII is still text: these answer 200, as on 0.8.1.
	for _, ok := range []string{"q=Test%20Caf%C3%A9", "place=test%E2%80%94cave", "q=%F0%9F%97%A1"} {
		if rec, _ := get(t, &fakeQ{}, "/v1/items?"+ok); rec.Code != http.StatusOK {
			t.Errorf("%s -> %d, want 200", ok, rec.Code)
		}
	}
}

// slugAsked answers a slug lookup the way Postgres answers one holding a byte text cannot: with an
// error that is not "no rows", which reached the reader as a 500. It records every slug asked.
type slugAsked struct {
	*fakeQ
	asked []string
}

func (s *slugAsked) GetItemBySlug(_ context.Context, slug string) (int32, error) {
	s.asked = append(s.asked, slug)
	return 0, errors.New(`ERROR: invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)`)
}

// The same bytes in an item's address are a 404, without a query: no item's slug can hold them, and
// /v1/items/{slug} and /armory/{slug} both answered 500 (AOC-069, measured). Service.Get decides it,
// so both surfaces get the answer from one place.
func TestAnAddressNoItemCanHaveIsNotFoundWithoutAsking(t *testing.T) {
	for _, slug := range []string{"a\x00b", "a\xffb", "\xc0\xaf", "\xed\xa0\x80"} {
		q := &slugAsked{fakeQ: &fakeQ{}}
		_, err := NewService(q).Get(context.Background(), slug)
		if !errors.Is(err, httpx.ErrNotFound) {
			t.Errorf("Get(%q): %v, want ErrNotFound", slug, err)
		}
		if len(q.asked) != 0 {
			t.Errorf("Get(%q) asked the database for %q", slug, q.asked)
		}
	}
	// A well-formed address is still looked up.
	q := &slugAsked{fakeQ: &fakeQ{}}
	_, _ = NewService(q).Get(context.Background(), "no-such-item")
	if len(q.asked) != 1 {
		t.Errorf("a well-formed slug was asked %d times, want 1", len(q.asked))
	}
}
