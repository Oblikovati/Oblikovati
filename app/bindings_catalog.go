// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"sort"
	"strings"
)

// The Customize Keyboard window's read model: the bindable catalog sorted and filtered for
// display (issue #1232: the raw Catalog is in registration order and offers no way to find a
// command by name). Kept in the app layer so the sort/filter is unit-tested headlessly; the
// head window is a thin renderer over CatalogFiltered.

// CatalogFiltered returns the bindable catalog sorted alphabetically by command name and,
// when query is non-empty, narrowed to the actions whose name or alias contains it
// (case-insensitive). An empty query returns the whole catalog, still sorted.
//
//	s.Bindings().CatalogFiltered("dim") // every dimension command, A→Z
func (b *Bindings) CatalogFiltered(query string) []Binding {
	q := strings.ToLower(strings.TrimSpace(query))
	full := b.Catalog()
	out := make([]Binding, 0, len(full))
	for _, bd := range full {
		if q == "" || bindingMatchesQuery(bd, q) {
			out = append(out, bd)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].DisplayName) < strings.ToLower(out[j].DisplayName)
	})
	return out
}

// bindingMatchesQuery reports whether the binding's command name or alias contains the
// (already lower-cased) query — the Customize Keyboard filter predicate.
func bindingMatchesQuery(b Binding, lowerQuery string) bool {
	return strings.Contains(strings.ToLower(b.DisplayName), lowerQuery) ||
		strings.Contains(strings.ToLower(b.Alias), lowerQuery)
}
