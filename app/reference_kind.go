// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"slices"
	"strings"
)

// selectionRefKind returns the kind prefix ("face"/"edge"/"vertex") of a selection reference
// string formatted "<kind>/<base64key>" (see model/feature/work_surface_ref.go); "" if malformed.
func selectionRefKind(ref string) string {
	i := strings.IndexByte(ref, '/')
	if i <= 0 {
		return ""
	}
	return ref[:i]
}

// filterRefsByAccepts keeps refs whose kind is in accepts; an empty accepts keeps all.
func filterRefsByAccepts(refs, accepts []string) []string {
	if len(accepts) == 0 {
		return refs
	}
	kept := make([]string, 0, len(refs))
	for _, r := range refs {
		if slices.Contains(accepts, selectionRefKind(r)) {
			kept = append(kept, r)
		}
	}
	return kept
}
