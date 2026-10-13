package items

import (
	"net/http"
	"testing"
)

// AOC-069 verify round 1. The seven list filters take several values (AOC-064), and every value
// reaches the query, so a bad byte in the SECOND one was a 500 on main as surely as in the first
// (measured: `rarity=epic&rarity=a%00b` and `place=x&place=a%FFb` answered 500). Every bad request in
// TestAByteNoTextCanHoldIsA400OnEveryParameter carries one value per parameter, so a check that read
// only the first (q.Get) passed it; this pins the later values too.
func TestAByteNoTextCanHoldInALaterValueIsA400Too(t *testing.T) {
	for _, p := range []string{"rarity", "equip_location", "armour_weight", "class", "currency", "set", "place"} {
		for _, bad := range []string{"%00", "%FF"} {
			path := "/v1/items?" + p + "=x&" + p + "=a" + bad + "b"
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
