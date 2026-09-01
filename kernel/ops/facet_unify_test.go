// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// Regression for Oblikovati#1693. A sequential pattern boolean re-facets the whole running body
// whenever one analytic cylinder wall survives (a drilled hole), and ops.Facet used to emit one
// planar face per tessellation triangle — shattering the untouched top-frame annulus of a shelled
// tray into a diagonal-laced fan. A later rolling-ball fillet on that rim then produced a
// non-manifold solid. ops.Facet now unifies coplanar faces, so the flat frame comes back as one
// face and the fillet stays a valid manifold solid.

// trayTopZ is the tray's outer top-plane height; trayVolume its exact material volume.
const (
	trayW, trayD, trayH = 6.0, 4.0, 2.5
	trayWall            = 0.2
	trayTopZ            = trayH
)

// shelledTray builds an open-top tray: a trayW×trayD×trayH box shelled to trayWall, top removed.
func shelledTray(t *testing.T) *topo.Body {
	t.Helper()
	box := brepfixture.Box(math.P3(0, 0, 0), trayW, trayD, trayH)
	tray, err := ops.Shell(box, [][]byte{topFaceKey(t, box)}, trayWall)
	if err != nil {
		t.Fatalf("shell: %v", err)
	}
	return tray
}

// countTopFrameFaces counts the +Z faces sitting in the outer top plane — the annular top frame,
// which must be a SINGLE face once coplanar triangles are unified.
func countTopFrameFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if ok && pl.Normal().Z > 0.99 && pl.Origin.Z > trayTopZ-1e-4 {
			n++
		}
	}
	return n
}

// topRimEdges returns the reference keys of every edge lying in the tray's outer top plane — the
// outer and inner rim of the frame (8 clean, axis-aligned edges once coplanar faces are unified).
func topRimEdges(b *topo.Body) [][]byte {
	var out [][]byte
	for _, e := range b.Edges() {
		bb := e.RangeBox()
		if bb.Min.Z > trayTopZ-1e-6 && bb.Max.Z < trayTopZ+1e-6 {
			out = append(out, e.ReferenceKey())
		}
	}
	return out
}

// TestFacetUnifiesCoplanarFrame faceting a shelled tray must NOT shatter its flat top frame: the
// annulus stays one face, the body stays a valid solid with χ and volume preserved.
func TestFacetUnifiesCoplanarFrame(t *testing.T) {
	t.Parallel()
	tray := shelledTray(t)
	wantVol := query.BodyGeometryProperties(tray, ops.DefaultQuality()).Volume

	faceted := ops.Facet(tray, "facet")
	if faceted == nil {
		t.Fatal("Facet returned nil")
	}
	r := ops.Validate(faceted)
	if !r.Valid || !faceted.IsSolid() {
		t.Fatalf("faceted tray not a valid solid: manifold=%v orient=%v closed=%v issues=%v",
			r.Manifold, r.OrientationOK, r.Closed, r.Issues)
	}
	if r.EulerCharacteristic != 2 {
		t.Errorf("faceted tray χ = %d, want 2 (genus-0 shell)", r.EulerCharacteristic)
	}
	if got := countTopFrameFaces(faceted); got != 1 {
		t.Errorf("top frame split into %d faces, want 1 (coplanar triangles must merge)", got)
	}
	if got := len(topRimEdges(faceted)); got != 8 {
		t.Errorf("top rim has %d edges, want 8 (outer+inner rect, no diagonals)", got)
	}
	if got := query.BodyGeometryProperties(faceted, ops.DefaultQuality()).Volume; stdmath.Abs(got-wantVol) > 1e-6 {
		t.Errorf("faceted tray volume = %g, want %g", got, wantVol)
	}
}

// TestFilletOnFacetedRimStaysManifold is the end-to-end guard: filleting the top rim of a FACETED
// tray (the state the #1693 pattern cascade leaves) must yield a valid manifold solid, not the
// "non-manifold edge used by 4 faces" the shattered rim produced.
func TestFilletOnFacetedRimStaysManifold(t *testing.T) {
	t.Parallel()
	faceted := ops.Facet(shelledTray(t), "facet")
	before := query.BodyGeometryProperties(faceted, ops.DefaultQuality()).Volume

	// r must clear the max-radius bound (#1800): the outer AND inner rim edges are both filleted
	// and their bands recede toward each other across the 0.2-wide top frame, so each must stay
	// under 0.1 (0.08 leaves a 0.04 margin). r=0.15 self-intersected but only faceting hid it.
	res, err := blend.FilletEdges(faceted, topRimEdges(faceted), 0.08)
	if err != nil {
		t.Fatalf("rim fillet on faceted tray failed: %v", err)
	}
	r := ops.Validate(res)
	if !r.Valid || !res.IsSolid() {
		t.Fatalf("rim-filleted faceted tray not a valid solid: manifold=%v orient=%v closed=%v issues=%v",
			r.Manifold, r.OrientationOK, r.Closed, r.Issues)
	}
	if r.EulerCharacteristic != 2 {
		t.Errorf("filleted tray χ = %d, want 2", r.EulerCharacteristic)
	}
	// The rim round removes a sliver of material, so the volume drops a little but stays close.
	if after := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; after >= before || after < before-1.0 {
		t.Errorf("filleted volume = %g, want a little under %g", after, before)
	}
}
