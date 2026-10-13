package main

import (
	"github.com/go-chi/chi/v5"

	"github.com/pierrehrt/aoc-api/internal/assets"
	"github.com/pierrehrt/aoc-api/internal/builds"
	"github.com/pierrehrt/aoc-api/internal/httpx"
	"github.com/pierrehrt/aoc-api/internal/items"
	"github.com/pierrehrt/aoc-api/internal/pages"
	"github.com/pierrehrt/aoc-api/internal/templates"
)

// querier is everything the composed application reads: *sqlcgen.Queries in production.
type querier interface {
	items.Querier
	items.TaxonomyQuerier
	builds.Querier
}

// newHandler composes the whole application over one querier: the pages, the assets and every
// /v1 router. run() serves it; main_test.go walks it, so a route added here is covered there
// without a line of its own (AOC-079).
func newHandler(build httpx.Build, q querier, tpl *templates.Engine, assetSet *assets.Set, baseURL string, opts ...httpx.RouterOption) *chi.Mux {
	// The read surface over the armory (AOC-012). ONE service, built here, handed to the JSON
	// handlers AND to the HTML pages (AOC-047) — never two implementations of the filtering
	// (CLAUDE.md rule 5b).
	itemsSvc := items.NewService(q)
	itemsAPI := items.NewHandler(itemsSvc, items.NewTaxonomyService(q))
	// The gear builder (AOC-051): one service for the Armory page and /v1/builds/compute.
	buildsSvc := builds.NewService(q)
	buildsAPI := builds.NewHandler(buildsSvc)
	site := pages.New(tpl, assetSet, baseURL, itemsSvc, buildsSvc)
	return httpx.NewRouterWithAPI(build, site.Routes, assetSet.Handler(), func(v1 chi.Router) {
		v1.Mount("/items", itemsAPI.Routes())
		v1.Mount("/taxonomies", itemsAPI.TaxonomyRoutes())
		v1.Mount("/sources", itemsAPI.SourceRoutes())
		v1.Mount("/builds", buildsAPI.Routes())
	}, opts...)
}
