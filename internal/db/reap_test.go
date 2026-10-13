package db_test

import (
	"fmt"
	"testing"
	"time"
)

// AOC-070: the reaper drops a throwaway database only when its name says it is older than
// staleAfter, so a run never drops another run's live database. A name it cannot date stays.
func TestOnlyAStaleThrowawayDatabaseIsReaped(t *testing.T) {
	now := time.Date(2026, 10, 13, 3, 0, 0, 0, time.UTC)
	name := func(age time.Duration) string { return fmt.Sprintf("aoc_test_%d_%d", now.Add(-age).UnixNano(), 42) }
	for _, tc := range []struct {
		name string
		want bool
	}{
		{name(0), false},                              // another run, starting now
		{name(staleAfter - time.Minute), false},       // a long run, still alive
		{name(staleAfter + time.Minute), true},        // left behind by a timeout or Ctrl-C
		{name(-time.Hour), false},                     // a clock ahead of ours: not old
		{"aoc_test_123", false},                       // not the shape freshDatabase makes
		{"aoc_test_abc_1", false},                     //
		{"aoc_test_1_2_3", false},                     //
		{"aoc_dev", false},                            // the developer's own database
		{"aoc_test_99999999999999999999999_1", false}, // a number that does not parse
	} {
		if got := isStaleTestDB(tc.name, now); got != tc.want {
			t.Errorf("isStaleTestDB(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
