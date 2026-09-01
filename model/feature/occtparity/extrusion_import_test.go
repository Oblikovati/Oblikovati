// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// extrusionSolidFixtures are the corpus inputs swept from an oblique conic (a STEP
// SURFACE_OF_LINEAR_EXTRUSION of an ELLIPSE or CIRCLE). Before the importer mapped that surface they
// imported as OPEN shells with a dropped elliptical-cylinder face, yet flagged IsSolid=true — the
// fillet then inherited an unclosed shell ("not a valid solid []"). See the STEP importer fix.
var extrusionSolidFixtures = []string{"F6", "T6", "T7", "U3", "U4"}

// TestExtrusionFixturesImportClosedSolid pins the import fix: each fixture must come in as a genuinely
// closed B-rep solid — every edge used by two faces, no boundary edges — not merely flagged solid.
func TestExtrusionFixturesImportClosedSolid(t *testing.T) {
	t.Parallel()
	for _, id := range extrusionSolidFixtures {
		b, err := importInput("fixtures/simple/" + id + ".step")
		if err != nil {
			t.Fatalf("%s: import: %v", id, err)
		}
		if !b.IsSolid() {
			t.Errorf("%s: IsSolid()=false, want a solid", id)
		}
		if bnd := boundaryEdgeCount(b); bnd != 0 {
			t.Errorf("%s: %d boundary (one-sided) edges, want 0 — a dropped face left the shell open", id, bnd)
		}
	}
}

// TestExtrusionFixturesTessellateWatertight pins the tessellation fix: the imported bodies (which now
// carry geom.EllipticalCylinder faces) must mesh watertight — every mesh edge shared by exactly two
// triangles — and integrate to a positive, quality-stable volume. Guards CLAUDE.md's tessellation
// priority for the newly-wired elliptical-cylinder mesh paths.
func TestExtrusionFixturesTessellateWatertight(t *testing.T) {
	t.Parallel()
	for _, id := range extrusionSolidFixtures {
		b, err := importInput("fixtures/simple/" + id + ".step")
		if err != nil {
			t.Fatalf("%s: import: %v", id, err)
		}
		m, _ := tessellate.TessellateBody(b, ops.Quality{})
		if open := openMeshEdgeCount(m); open != 0 {
			t.Errorf("%s: %d non-manifold mesh edges, want 0 (watertight)", id, open)
		}
		if v := query.BodyGeometryProperties(b, ops.PropertyQuality()).Volume; v <= 0 {
			t.Errorf("%s: mesh volume = %g, want > 0", id, v)
		}
	}
}

// boundaryEdgeCount counts B-rep edges used by fewer than two faces (an open shell).
func boundaryEdgeCount(b *topo.Body) int {
	n := 0
	for _, e := range b.Edges() {
		if len(e.Uses()) < 2 {
			n++
		}
	}
	return n
}

// openMeshEdgeCount counts triangle edges not shared by exactly two triangles, welding vertices at a
// 1e-6 model tolerance (the same vertex reached via two faces differs in the last ULP, so exact-float
// welding would report spurious cracks).
func openMeshEdgeCount(m *ops.Mesh) int {
	type pk [3]int64
	round := func(p [3]float64) pk {
		return pk{int64(p[0]*1e6 + 0.5), int64(p[1]*1e6 + 0.5), int64(p[2]*1e6 + 0.5)}
	}
	key := func(i int) pk {
		p := m.Positions[i]
		return round([3]float64{float64(p.X), float64(p.Y), float64(p.Z)})
	}
	use := map[[2]pk]int{}
	for t := 0; t+2 < len(m.Indices); t += 3 {
		tri := [3]int{m.Indices[t], m.Indices[t+1], m.Indices[t+2]}
		for k := range 3 {
			a, c := key(tri[k]), key(tri[(k+1)%3])
			if less(c, a) {
				a, c = c, a
			}
			use[[2]pk{a, c}]++
		}
	}
	open := 0
	for _, u := range use {
		if u != 2 {
			open++
		}
	}
	return open
}

// less orders two rounded positions lexicographically for a canonical undirected edge key.
func less(a, b [3]int64) bool {
	for i := range 3 {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}
