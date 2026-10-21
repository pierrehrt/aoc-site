package db_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pierrehrt/aoc-api/internal/db/sqlcgen"
	"github.com/pierrehrt/aoc-api/internal/items"
)

// AOC-081 verify round 1: the source panel's counts honour used_by, as they do every other list
// filter. Nothing tested it: with `AND f.in_used_by` deleted from ListSourceTreeRows' `matches`,
// every test still passed (the corpus tree test has an epic state and a class state, no used_by).
//
// Four fake items in one seeded section: restricted to the class, restricted to nobody and of a type
// the class uses, restricted to another class only, and restricted to nobody with no type and no
// weight. Under used_by the panel must count the first two, and each branch must count what picking
// it lists. The class, the type and the section are read from the seed, never typed here (STEP ZERO).
func TestTheSourcePanelCountsWhatAClassUses(t *testing.T) {
	_, url := migratedDB(t)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	one := func(what, q string, args ...any) string {
		t.Helper()
		var s string
		if err := pool.QueryRow(ctx, q, args...).Scan(&s); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		return s
	}
	class := one("a class that uses a type", `SELECT c.slug FROM classes c
		WHERE EXISTS (SELECT 1 FROM class_item_types x WHERE x.class_id = c.id) ORDER BY c.id LIMIT 1`)
	other := one("another class", `SELECT slug FROM classes WHERE slug <> $1 ORDER BY id LIMIT 1`, class)
	usedType := one("a type the class uses", `SELECT it.slug FROM item_types it
		JOIN class_item_types x ON x.item_type_id = it.id JOIN classes c ON c.id = x.class_id
		WHERE c.slug = $1 ORDER BY it.id LIMIT 1`, class)
	tab := one("a tab drawn by section", `SELECT slug FROM source_tabs WHERE groups = '{section}' ORDER BY sort_order LIMIT 1`)

	for _, x := range []struct {
		id           int
		typ          string // "" for none
		restrictedTo string // "" for nobody
	}{
		{9411, "", class},    // restricted to the class: counted
		{9412, usedType, ""}, // nobody's, a type the class uses: counted
		{9413, usedType, other},
		{9414, "", ""},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO items (item_id, slug, name, rarity_id, item_type_id, confidence_id, source_note)
			VALUES ($1, $2, $3, (SELECT id FROM rarities ORDER BY sort_order LIMIT 1), (SELECT id FROM item_types WHERE slug = $4),
			        (SELECT id FROM confidence_levels WHERE slug = 'unconfirmed'), 'fixture — item_used_by_tree_verify_test.go, not a game fact')`,
			x.id, fmt.Sprintf("test-used-by-tree-%d", x.id), fmt.Sprintf("Test Used By Tree %d", x.id), x.typ); err != nil {
			t.Fatalf("item %d: %v", x.id, err)
		}
		if x.restrictedTo != "" {
			if _, err := pool.Exec(ctx, `INSERT INTO item_classes (item_id, class_id) VALUES ($1, (SELECT id FROM classes WHERE slug = $2))`, x.id, x.restrictedTo); err != nil {
				t.Fatalf("item %d class: %v", x.id, err)
			}
		}
		if _, err := pool.Exec(ctx, `INSERT INTO item_sources (item_id, acquisition_type_id, section_id, confidence_id, source_note)
			VALUES ($1, (SELECT id FROM acquisition_types WHERE slug = 'drop'),
			        (SELECT sc.id FROM sections sc JOIN source_tabs t ON t.id = sc.tab_id WHERE t.slug = $2 ORDER BY sc.sort_order LIMIT 1),
			        (SELECT id FROM confidence_levels WHERE slug = 'unconfirmed'), 'fixture — item_used_by_tree_verify_test.go')`,
			x.id, tab); err != nil {
			t.Fatalf("source of %d: %v", x.id, err)
		}
	}

	s := items.NewService(sqlcgen.New(pool))
	// walk checks every branch against the list it promises and returns the branches it saw.
	walk := func(f items.Filters) (int64, int) {
		t.Helper()
		f.Tab = tab
		tree, err := s.Tree(ctx, f)
		if err != nil {
			t.Fatal(err)
		}
		var nodes int
		var visit func(ns []*items.TreeNode)
		visit = func(ns []*items.TreeNode) {
			for _, n := range ns {
				nodes++
				src, err := items.ParseSource(n.Source)
				if err != nil {
					t.Fatalf("%s: %v", n.Source, err)
				}
				g := f
				g.Source, g.Limit = src, 1
				res, err := s.List(ctx, g)
				if err != nil {
					t.Fatal(err)
				}
				if res.Total != n.Count {
					t.Errorf("used_by=%v %s: the branch says %d, picking it lists %d", f.UsedBy, n.Source, n.Count, res.Total)
				}
				visit(n.Children)
			}
		}
		visit(tree.Nodes)
		return tree.Total, nodes
	}

	allTotal, allNodes := walk(items.Filters{})
	if allTotal != 4 {
		t.Errorf("unfiltered, the panel holds %d items, want the 4 fixtures", allTotal)
	}
	usedTotal, usedNodes := walk(items.Filters{UsedBy: []string{class}})
	if usedTotal != 2 {
		t.Errorf("used_by=%s, the panel holds %d items, want 2 (restricted to it, and nobody's of a type it uses)", class, usedTotal)
	}
	if allNodes == 0 || usedNodes != allNodes {
		t.Errorf("used_by=%s lists %d branches, %d unfiltered: a branch the filter empties must stay listed", class, usedNodes, allNodes)
	}
}
