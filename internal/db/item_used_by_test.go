package db_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pierrehrt/aoc-api/internal/db/sqlcgen"
	"github.com/pierrehrt/aoc-api/internal/items"
)

// AOC-081: what a class USES is the items restricted to it, plus the items restricted to nobody whose
// armour weight or item type the class uses (the view item_used_by, over Pierre's table). One fake
// item per branch of the rule. The class, the weights and the types it uses and does not are read
// from the seed, never typed here (CLAUDE.md STEP ZERO): the test pins the rule, not the table.
func TestUsedByIsTheRestrictedItemsPlusWhatTheClassUses(t *testing.T) {
	d, url := migratedDB(t)
	ctx := context.Background()
	one := func(what, q string, args ...any) string {
		t.Helper()
		var s string
		if err := d.QueryRowContext(ctx, q, args...).Scan(&s); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		return s
	}
	class := one("a class that uses a weight and a type", `SELECT c.slug FROM classes c
		WHERE EXISTS (SELECT 1 FROM class_armour_weights w WHERE w.class_id = c.id)
		  AND EXISTS (SELECT 1 FROM class_item_types it WHERE it.class_id = c.id)
		ORDER BY c.id LIMIT 1`)
	other := one("another class", `SELECT slug FROM classes WHERE slug <> $1 ORDER BY id LIMIT 1`, class)
	uses := `EXISTS (SELECT 1 FROM %s x JOIN classes c ON c.id = x.class_id WHERE c.slug = $1 AND x.%s = t.id)`
	pick := func(what, table, link, order string, used bool) string {
		cond := "NOT "
		if used {
			cond = ""
		}
		// #nosec G201 -- table, column and order are literals from this file.
		return one(what, "SELECT t.slug FROM "+table+" t WHERE "+cond+fmt.Sprintf(uses, "class_"+table, link)+" ORDER BY "+order+" LIMIT 1", class)
	}
	usedWeight := pick("a weight the class uses", "armour_weights", "armour_weight_id", "t.sort_order", true)
	unusedWeight := pick("a weight it does not", "armour_weights", "armour_weight_id", "t.sort_order", false)
	usedType := pick("a type the class uses", "item_types", "item_type_id", "t.id", true)
	unusedType := pick("a type it does not", "item_types", "item_type_id", "t.id", false)

	type fx struct {
		id           int
		weight, typ  string // slugs, "" for none
		restrictedTo []string
	}
	fixture := []fx{
		{9301, unusedWeight, "", []string{class}},       // restricted to it: in, whatever it weighs
		{9302, usedWeight, "", []string{other}},         // restricted to another: never, even of a used weight
		{9303, usedWeight, "", nil},                     // nobody's, a used weight: in
		{9304, "", usedType, nil},                       // nobody's, a used type: in
		{9305, unusedWeight, "", nil},                   // nobody's, an unused weight: out
		{9306, "", unusedType, nil},                     // nobody's, an unused type: out
		{9307, "", "", nil},                             // nobody's, no weight, no type: out
		{9308, "", unusedType, []string{class, other}},  // restricted to it and another: in
		{9309, unusedWeight, usedType, []string{other}}, // another's, of a used type: out
		{9310, unusedWeight, usedType, nil},             // nobody's, a used type, an unused weight: in
	}
	for _, x := range fixture {
		if _, err := d.ExecContext(ctx, `INSERT INTO items (item_id, slug, name, rarity_id, armour_weight_id, item_type_id, confidence_id, source_note)
			VALUES ($1, $2, $3, (SELECT id FROM rarities ORDER BY sort_order LIMIT 1),
			        (SELECT id FROM armour_weights WHERE slug = $4), (SELECT id FROM item_types WHERE slug = $5),
			        (SELECT id FROM confidence_levels WHERE slug = 'unconfirmed'), 'fixture — item_used_by_test.go, not a game fact')`,
			x.id, fmt.Sprintf("test-used-by-%d", x.id), fmt.Sprintf("Test Used By %d", x.id), x.weight, x.typ); err != nil {
			t.Fatalf("item %d: %v", x.id, err)
		}
		for _, c := range x.restrictedTo {
			if _, err := d.ExecContext(ctx, `INSERT INTO item_classes (item_id, class_id) VALUES ($1, (SELECT id FROM classes WHERE slug = $2))`, x.id, c); err != nil {
				t.Fatalf("item %d class %s: %v", x.id, c, err)
			}
		}
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	s := items.NewService(sqlcgen.New(pool))
	ids := func(f items.Filters) []int32 {
		t.Helper()
		f.Limit, f.Sort = items.MaxLimit, items.SortID
		res, err := s.List(ctx, f)
		if err != nil {
			t.Fatalf("List(%+v): %v", f, err)
		}
		var out []int32
		for _, it := range res.Items {
			out = append(out, it.ID)
		}
		return out
	}

	if got, want := ids(items.Filters{UsedBy: []string{class}}), []int32{9301, 9303, 9304, 9308, 9310}; !slices.Equal(got, want) {
		t.Errorf("used_by=%s lists %v, want %v", class, got, want)
	}
	// class= keeps its meaning: restricted to.
	if got, want := ids(items.Filters{Classes: []string{class}}), []int32{9301, 9308}; !slices.Equal(got, want) {
		t.Errorf("class=%s lists %v, want %v (restricted to it, unchanged)", class, got, want)
	}
	// The other class: its own restricted items, never this one's.
	got := ids(items.Filters{UsedBy: []string{other}})
	for _, id := range []int32{9302, 9308, 9309} {
		if !slices.Contains(got, id) {
			t.Errorf("used_by=%s lacks %d, restricted to it", other, id)
		}
	}
	if slices.Contains(got, 9301) {
		t.Errorf("used_by=%s lists 9301, restricted to %s only", other, class)
	}
	// The facet counts the same list.
	res, err := s.List(ctx, items.Filters{WithFacets: true, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range res.Facets.UsedBy.Values {
		if v.Slug == class && v.Count != 5 {
			t.Errorf("the used_by facet counts %d for %s, want the 5 it lists", v.Count, class)
		}
	}
}
