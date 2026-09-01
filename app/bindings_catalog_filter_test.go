// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"sort"
	"strings"
	"testing"
)

// TestCatalogFilteredIsAlphabetical guards the Customize Keyboard sort (issue #1232): the
// filtered catalog is ordered by command name case-insensitively, regardless of the
// registration order the raw Catalog preserves.
func TestCatalogFilteredIsAlphabetical(t *testing.T) {
	t.Parallel()
	s := NewSession()
	got := s.Bindings().CatalogFiltered("")
	if len(got) < 2 {
		t.Fatalf("catalog has %d entries, want several built-ins", len(got))
	}
	names := make([]string, len(got))
	for i, b := range got {
		names[i] = strings.ToLower(b.DisplayName)
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("CatalogFiltered is not alphabetical: %v", names)
	}
}

// TestCatalogFilteredNarrowsByName guards the filter (issue #1232): a query returns only the
// actions whose name contains it (case-insensitive), and includes the new Delete action.
func TestCatalogFilteredNarrowsByName(t *testing.T) {
	t.Parallel()
	s := NewSession()
	got := s.Bindings().CatalogFiltered("del")
	if len(got) == 0 {
		t.Fatal("filter \"del\" returned nothing, want the Delete action")
	}
	foundDelete := false
	for _, b := range got {
		if !strings.Contains(strings.ToLower(b.DisplayName), "del") &&
			!strings.Contains(strings.ToLower(b.Alias), "del") {
			t.Errorf("filter \"del\" returned non-matching action %q", b.DisplayName)
		}
		if b.ActionID == ActionDelete {
			foundDelete = true
		}
	}
	if !foundDelete {
		t.Errorf("filter \"del\" did not include the Delete action (%s)", ActionDelete)
	}
}

// TestCatalogFilteredEmptyForNoMatch: a query matching nothing returns an empty slice.
func TestCatalogFilteredEmptyForNoMatch(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if got := s.Bindings().CatalogFiltered("zzz-no-such-command"); len(got) != 0 {
		t.Errorf("filter for a missing command returned %d entries, want 0", len(got))
	}
}
