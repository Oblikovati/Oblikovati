// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
)

// verticalEdgeKey returns a vertical edge (start/end share X,Y) of a box.
func verticalEdgeKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			return e.ReferenceKey()
		}
	}
	t.Fatal("no vertical edge")
	return nil
}

// hasCylinderFace reports whether the body has at least n cylindrical faces.
func hasCylinderFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// filletNotch is the cross-section area a fillet of radius r removes at a convex 90° edge:
// the square corner r² minus the quarter disc πr²/4.
func filletNotch(r float64) float64 { return r*r - stdmath.Pi*r*r/4 }

// TestFilletOneEdge rounds one vertical edge of a 2×2×2 box (r=0.5): a valid solid with one
// cylinder face, volume 8 − (r²−πr²/4)·L (L=2). The headline rolling-ball-fillet acceptance.
func TestFilletOneEdge(t *testing.T) {
	box := shellBox(2, 2, 2)
	res, err := ops.FilletEdges(box, [][]byte{verticalEdgeKey(t, box)}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("filleted box not a valid solid: %+v", r)
	}
	if n := hasCylinderFaces(res); n != 1 {
		t.Errorf("filleted box has %d cylinder faces, want 1", n)
	}
	// The cylinder face is exact; the measured volume is its faceted approximation, so check
	// at a fine tessellation tolerance where it has converged to the analytic value.
	want := 8 - filletNotch(0.5)*2
	if got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume; stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("fillet volume = %g, want ≈ %g", got, want)
	}
}

// TestFilletFourVerticalEdges rounds all four vertical edges of a 2×2×2 box in one pass:
// a validated manifold solid with four quarter-cylinder faces of radius 0.5 (the F03
// acceptance). Volume 8 − 4·(r²−πr²/4)·L.
func TestFilletFourVerticalEdges(t *testing.T) {
	box := shellBox(2, 2, 2)
	var keys [][]byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 4 {
		t.Fatalf("found %d vertical edges, want 4", len(keys))
	}
	res, err := ops.FilletEdges(box, keys, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("filleted box not a valid solid: %+v", r)
	}
	if n := hasCylinderFaces(res); n != 4 {
		t.Errorf("filleted box has %d cylinder faces, want 4", n)
	}
	want := 8 - 4*filletNotch(0.5)*2
	if got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume; stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("four-edge fillet volume = %g, want ≈ %g", got, want)
	}
}

// TestFilletLostKeyErrors checks a non-convex (concave) edge is rejected rather than
// producing garbage. A box has no concave edges, so a lost key stands in for the error path.
func TestFilletLostKeyErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.FilletEdges(box, [][]byte{[]byte("ghost")}, 0.5); err == nil {
		t.Error("fillet with a lost key should error")
	}
}

// cornerEdgeKeys returns the reference keys of the edges meeting at the (0,0,0) box corner.
func cornerEdgeKeys(t *testing.T, b *topo.Body) [][]byte {
	t.Helper()
	var keys [][]byte
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if (a.X == 0 && a.Y == 0 && a.Z == 0) || (c.X == 0 && c.Y == 0 && c.Z == 0) {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

// hasSphereFaces counts a body's spherical faces.
func hasSphereFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			n++
		}
	}
	return n
}

// TestFilletCornerBlend rounds the three edges meeting at a box corner: the three cylinder
// fillets are joined by a spherical corner patch into a valid solid (3 cylinders + 1 sphere),
// with material removed (volume < 8). The corner-blend acceptance.
func TestFilletCornerBlend(t *testing.T) {
	box := shellBox(2, 2, 2)
	keys := cornerEdgeKeys(t, box)
	if len(keys) != 3 {
		t.Fatalf("found %d edges at the corner, want 3", len(keys))
	}
	res, err := ops.FilletEdges(box, keys, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("corner-blended box not a valid solid: %+v", r)
	}
	if c, s := hasCylinderFaces(res), hasSphereFaces(res); c != 3 || s != 1 {
		t.Errorf("got %d cylinder + %d sphere faces, want 3 + 1", c, s)
	}
	if v := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume; v <= 7.5 || v >= 8 {
		t.Errorf("corner-blend volume = %g, want material removed (7.5 < v < 8)", v)
	}
}

// meshOpenEdges counts the welded mesh edges not shared by exactly two triangles — i.e. the
// cracks. A watertight tessellation has zero.
func meshOpenEdges(m *ops.Mesh) int {
	grid := func(x float64) int64 { return int64(stdmath.Round(x / 1e-5)) }
	id := map[[3]int64]int{}
	vid := func(i int) int {
		p := m.Positions[i]
		k := [3]int64{grid(p.X), grid(p.Y), grid(p.Z)}
		if v, ok := id[k]; ok {
			return v
		}
		v := len(id)
		id[k] = v
		return v
	}
	use := map[[2]int]int{}
	for t := 0; t+2 < len(m.Indices); t += 3 {
		vs := [3]int{vid(m.Indices[t]), vid(m.Indices[t+1]), vid(m.Indices[t+2])}
		for _, e := range [][2]int{{vs[0], vs[1]}, {vs[1], vs[2]}, {vs[2], vs[0]}} {
			if e[0] > e[1] {
				e[0], e[1] = e[1], e[0]
			}
			use[e]++
		}
	}
	open := 0
	for _, c := range use {
		if c != 2 {
			open++
		}
	}
	return open
}

// TestFilletCornerBlendMeshWatertight checks the corner blend's TESSELLATION is watertight at
// a fine quality — the spherical patch's lat/long UV is degenerate where a box corner lands on
// the pole, so it must be flattened onto a tangent plane (else the ear-clip stalls and cracks).
func TestFilletCornerBlendMeshWatertight(t *testing.T) {
	box := shellBox(4, 3, 5)
	keys := cornerEdgeKeys(t, box)
	res, err := ops.FilletEdges(box, keys, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	for _, tol := range []float64{0.05, 1e-2, 1e-3} {
		m, _ := ops.TessellateBody(res, ops.Quality{ChordTolerance: tol})
		if open := meshOpenEdges(m); open != 0 {
			t.Errorf("corner-blend mesh at tol %g has %d open edges, want watertight", tol, open)
		}
	}
}

// TestFilletAllBoxEdges rounds every edge of a 2×2×2 box: 12 cylinder fillets joined by 8
// spherical corner patches into a valid solid (a fully-rounded box), with material removed.
func TestFilletAllBoxEdges(t *testing.T) {
	box := shellBox(2, 2, 2)
	var keys [][]byte
	for _, e := range box.Edges() {
		keys = append(keys, e.ReferenceKey())
	}
	res, err := ops.FilletEdges(box, keys, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("fully-rounded box not a valid solid: %+v", r)
	}
	if c, s := hasCylinderFaces(res), hasSphereFaces(res); c != 12 || s != 8 {
		t.Errorf("got %d cylinder + %d sphere faces, want 12 + 8", c, s)
	}
	if v := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume; v >= 8 {
		t.Errorf("fully-rounded box volume = %g, want < 8", v)
	}
}

// TestFilletTwoEdgeCornerErrors checks an unsupported corner config (two of the three edges
// at a vertex) errors clearly rather than producing a broken solid.
func TestFilletTwoEdgeCornerErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	keys := cornerEdgeKeys(t, box)[:2] // two of the three edges meeting at the corner
	_, err := ops.FilletEdges(box, keys, 0.3)
	if err == nil {
		t.Fatal("filleting only two of the three edges at a corner should error")
	}
	if !strings.Contains(err.Error(), "corner") {
		t.Errorf("error %q should mention the corner limitation", err)
	}
}

// TestFilletRadiusMustBePositive guards the radius.
func TestFilletRadiusMustBePositive(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.FilletEdges(box, [][]byte{verticalEdgeKey(t, box)}, 0); err == nil {
		t.Error("zero radius should error")
	}
}
