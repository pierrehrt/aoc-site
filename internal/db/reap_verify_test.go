package db_test

// AOC-070 verify round 1: what TestOnlyAStaleThrowawayDatabaseIsReaped does not reach. Five mutants
// passed that table and fail a test below: reapOrphans with its age check deleted (main's reaper,
// and the rest of the suite passes with it alone), `>` turned to `>=`, staleAfter set to one minute
// or to thirty minutes, and freshDatabase naming by Unix seconds instead of nanoseconds.

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
)

// The rule's edges, in absolute durations so that the two hours the comment states are pinned too.
func TestStaleRuleEdgesVerify(t *testing.T) {
	now := time.Date(2026, 10, 13, 3, 0, 0, 0, time.UTC)
	at := func(age time.Duration) string { return fmt.Sprintf("aoc_test_%d_%d", now.Add(-age).UnixNano(), 7) }
	for _, tc := range []struct {
		name string
		want bool
	}{
		{at(2 * time.Hour), false},                // exactly the age is not older than it
		{at(2*time.Hour + time.Nanosecond), true}, // one nanosecond older is
		{at(2*time.Hour - time.Nanosecond), false},
		{at(40 * time.Minute), false},             // a run still alive under go test -timeout=40m
		{at(-time.Nanosecond), false},             // a clock a nanosecond ahead
		{"aoc_test_0_0", true},                    // freshDatabase's shape, dated 1970
		{"aoc_test_9223372036854775807_1", false}, // the largest int64: 2262, not old
		{"aoc_testx1_1", false},                   // LIKE 'aoc_test_%' returns it: _ is a wildcard there
		{"", false},
		{"aoc_test_", false},
		{"aoc_test__1", false},
		{"aoc_test_1_", false},
		{"aoc_test_-1_1", false},
		{"aoc_test_+1_1", false},
		{"aoc_test_1_1 ", false},
		{"aoc_test_1_1\n", false},
		{"AOC_TEST_1_1", false},
		{"aoc_test_1_1_aoc_test_1_1", false},
	} {
		if got := isStaleTestDB(tc.name, now); got != tc.want {
			t.Errorf("isStaleTestDB(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// The reaper itself on a real server: it drops a planted database its name dates older than
// staleAfter, and leaves a fresh one and one it cannot date. Then the name freshDatabase really
// makes: live now, stale once staleAfter has passed.
func TestReaperDropsOnlyTheStalePlantedDatabaseVerify(t *testing.T) {
	admin := adminURL(t)
	ctx := context.Background()
	d, err := sql.Open("pgx", admin)
	if err != nil {
		t.Fatalf("connecting to Postgres: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	now := time.Now()
	stale := fmt.Sprintf("aoc_test_%d_%d", now.Add(-staleAfter-time.Hour).UnixNano(), 1)
	fresh := fmt.Sprintf("aoc_test_%d_%d", now.Add(-time.Minute).UnixNano(), 2)
	undatable := fmt.Sprintf("aoc_test_verify_%d", now.UnixNano())
	for _, n := range []string{stale, fresh, undatable} {
		if _, err := d.ExecContext(ctx, "CREATE DATABASE "+n); err != nil {
			t.Fatalf("planting %s: %v", n, err)
		}
		t.Cleanup(func() { _, _ = d.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+n+" WITH (FORCE)") })
	}

	reapOrphans(t, d, ctx)

	exists := func(n string) bool {
		t.Helper()
		var c int
		if err := d.QueryRowContext(ctx, `SELECT count(*) FROM pg_database WHERE datname = $1`, n).Scan(&c); err != nil {
			t.Fatalf("looking for %s: %v", n, err)
		}
		return c == 1
	}
	if exists(stale) {
		t.Errorf("%s is dated %v ago and was not reaped", stale, staleAfter+time.Hour)
	}
	if !exists(fresh) {
		t.Errorf("%s is a minute old and was reaped", fresh)
	}
	if !exists(undatable) {
		t.Errorf("%s cannot be dated and was reaped", undatable)
	}

	u, err := url.Parse(freshDatabase(t))
	if err != nil {
		t.Fatalf("parsing freshDatabase's URL: %v", err)
	}
	name := strings.TrimPrefix(u.Path, "/")
	if isStaleTestDB(name, time.Now()) {
		t.Errorf("freshDatabase made %s and the rule calls it stale already", name)
	}
	if !isStaleTestDB(name, time.Now().Add(staleAfter+time.Minute)) {
		t.Errorf("freshDatabase made %s and the rule would not reap it once staleAfter has passed", name)
	}
}
