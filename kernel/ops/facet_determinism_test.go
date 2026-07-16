// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	"sort"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestFacetIsDeterministic pins #23: re-faceting the SAME analytic body must yield the same
// topology every time, not merely the same vertices.
//
// Facet unifies its triangle cage back into maximal coplanar faces, and that pass chains boundary
// half-edges into face loops through Go maps. Map iteration is randomized, so where two coplanar
// boundary loops touched at a vertex the pairing — hence the LOOPS themselves — varied run to run.
// Nothing downstream could recover: the feature layer re-facets every analytic operand before the
// planar boolean, so each rebuild fed a differently-connected tool into the boolean, and a real part
// (ReelToReel BigChunkyPlate) alternated between solid and surface with its volume wandering ~300
// mm^3 across identical runs. The vertex set matched throughout, which is why this asserts
// CONNECTIVITY.
func TestFacetIsDeterministic(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	var want string
	for i := 0; i < 25; i++ {
		faceted := Facet(cyl, "f")
		if faceted == nil {
			t.Fatalf("run %d: Facet returned nil", i)
		}
		got := faceTopologyKey(faceted)
		if i == 0 {
			want = got
			continue
		}
		if got != want {
			t.Fatalf("run %d faceted the same body differently:\n got %s\nwant %s", i, got, want)
		}
	}
}

// faceTopologyKey fingerprints a body's face CONNECTIVITY — each face's surface type with the
// bounding vertex positions of its loops — not just its vertex set, which stays equal even when the
// faces are wired differently.
func faceTopologyKey(b *topo.Body) string {
	keys := make([]string, 0, len(b.Faces()))
	for _, f := range b.Faces() {
		var loop []string
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				p := u.Edge().StartVertex().Point()
				q := u.Edge().EndVertex().Point()
				loop = append(loop, fmt.Sprintf("%.6f,%.6f,%.6f/%.6f,%.6f,%.6f", p.X, p.Y, p.Z, q.X, q.Y, q.Z))
			}
		}
		sort.Strings(loop)
		keys = append(keys, fmt.Sprintf("%T:%v", f.Geometry(), loop))
	}
	sort.Strings(keys)
	return fmt.Sprintf("f=%d e=%d v=%d h=%08x", len(b.Faces()), len(b.Edges()), len(b.Vertices()), fnv32(fmt.Sprint(keys)))
}

// fnv32 folds a key string into a compact fingerprint for comparison across runs.
func fnv32(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
