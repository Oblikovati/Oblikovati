// SPDX-License-Identifier: GPL-2.0-only

package retopo

import (
	"fmt"

	"oblikovati.org/kernel/topo"
)

// ResolveFaceSet turns face reference keys into the set of face IDs they name, erroring if a
// key no longer resolves (the feature must go Sick honestly).
func ResolveFaceSet(solid *topo.Body, keys [][]byte) (map[uint64]bool, error) {
	set := make(map[uint64]bool, len(keys))
	for _, k := range keys {
		// ADR-0043 resolution guard: a key must bind to exactly one face. >1 is a topological-
		// naming collision, surfaced as an honest error rather than a silent first-match.
		match := solid.FacesByKey(k)
		switch len(match) {
		case 1:
			set[match[0].ID()] = true
		case 0:
			return nil, fmt.Errorf("face reference lost: %x", k)
		default:
			return nil, fmt.Errorf("face reference %x is ambiguous — it matches %d faces (a topological-naming collision)", k, len(match))
		}
	}
	return set, nil
}
