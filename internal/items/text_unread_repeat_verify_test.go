package items

import (
	"net/http"
	"testing"
)

// AOC-069 verify round 2. A parameter that takes one value reads the first (q.Get, vs[0]), so on
// 0.8.1 a bad byte in a REPEAT of it was never read: `q=x&q=%FF`, `limit=5&limit=a%00b` answered 200
// (measured, local build of v0.8.1 against the dev database), where the same byte in the value read
// was a 500 (a text filter) or a 400 (the others). The check covers the whole query string, so the
// repeat is a 400 now and the database is never asked. The first value alone answers 200 here, so
// the 400 is the repeat's.
func TestAByteNoTextCanHoldInARepeatNothingReadsIsA400(t *testing.T) {
	first := map[string]string{
		"item_type": "x", "region": "x", "tier": "x", "q": "x", "sort": "name", "tab": "x",
		"source": "s:x", "get": "x", "pvp": "true", "unchained": "true", "price": "true",
		"ilvl_min": "1", "ilvl_max": "300", "reqlvl_min": "1", "reqlvl_max": "80", "facets": "true",
		"limit": "5", "offset": "0",
	}
	for p, v := range first {
		pre := ""
		if p == "get" { // a half is of a source: `get` alone is refused for another reason
			pre = "source=s:x&"
		}
		if rec, _ := get(t, &fakeQ{}, "/v1/items?"+pre+p+"="+v); rec.Code != http.StatusOK {
			t.Errorf("%s%s=%s alone -> %d, want 200", pre, p, v, rec.Code)
			continue
		}
		for _, bad := range []string{"%00", "%FF"} {
			path := "/v1/items?" + pre + p + "=" + v + "&" + p + "=a" + bad + "b"
			q := &fakeQ{}
			rec, body := get(t, q, path)
			if rec.Code != http.StatusBadRequest || body["error"] == nil {
				t.Errorf("%s -> %d %v, want 400 with a JSON error", path, rec.Code, body)
			}
			if q.calls != 0 || q.facetCalled != 0 {
				t.Errorf("%s reached the database (%d list calls, %d facet calls)", path, q.calls, q.facetCalled)
			}
		}
	}
}
