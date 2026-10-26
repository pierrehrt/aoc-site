//go:build corpus

// A corpus test: compiled only with `-tags corpus` (AOC-044) — see item_read_endpoints_test.go.

package db_test

import (
	"context"
	"testing"

	"github.com/pierrehrt/aoc-api/internal/items"
)

// AOC-081 on the real corpus: for every class, the used_by list's total is a count made WITHOUT the
// view, straight from item_classes and the two tables Pierre's table seeded. And no class's list
// holds an item restricted only to other classes. The classes come from the data, none is typed here.
func TestUsedByMatchesAnIndependentCountOnTheRealCorpus(t *testing.T) {
	s := svc(t)
	pool := readPool(t)
	ctx := context.Background()
	rows, err := pool.Query(ctx, `
		SELECT c.slug, (SELECT count(*) FROM items i WHERE
		   EXISTS (SELECT 1 FROM item_classes ic WHERE ic.item_id = i.item_id AND ic.class_id = c.id)
		   OR (NOT EXISTS (SELECT 1 FROM item_classes ic WHERE ic.item_id = i.item_id)
		       AND (i.armour_weight_id IN (SELECT armour_weight_id FROM class_armour_weights WHERE class_id = c.id)
		            OR i.item_type_id IN (SELECT item_type_id FROM class_item_types WHERE class_id = c.id))))
		FROM classes c ORDER BY c.id`)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int64{}
	for rows.Next() {
		var slug string
		var n int64
		if err := rows.Scan(&slug, &n); err != nil {
			t.Fatal(err)
		}
		want[slug] = n
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 {
		t.Fatal("no classes: the corpus is not loaded")
	}
	for slug, n := range want {
		if got := total(t, s, items.Filters{UsedBy: []string{slug}}); got != n {
			t.Errorf("used_by=%s: the list has %d items, the independent count %d", slug, got, n)
		}
		if n == 0 {
			t.Errorf("used_by=%s: nothing at all — the seed or the import is missing", slug)
		}
	}
	var stray int64
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM item_used_by u
		WHERE EXISTS (SELECT 1 FROM item_classes ic WHERE ic.item_id = u.item_id)
		  AND NOT EXISTS (SELECT 1 FROM item_classes ic WHERE ic.item_id = u.item_id AND ic.class_id = u.class_id)`).Scan(&stray); err != nil {
		t.Fatal(err)
	}
	if stray != 0 {
		t.Errorf("%d (item, class) pairs give a class an item restricted only to others", stray)
	}
}
