// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// ellipticRimPerFaceTol is the budget for "our per-face mesh area equals DRAWEXE's own sprops number",
// RELATIVE and per face. The carried elliptic rim is exact geometry, so every face lands within 9.0e-5
// (F7's 2329.06 face) of OCCT 8.0.0 — the residual is mesh quantization, not construction. 2e-4 leaves
// ~2× headroom while staying THREE decades tighter than the chorded rim it replaces (F7's elliptic wall
// measured +50.9% and its top plane −73.4%; F6's top plane +133% and its wall −24.4%).
const ellipticRimPerFaceTol = 2e-4

// ellipticRimOnFaceTol is the budget for "every boundary edge lies on the face it bounds" on these bodies,
// relative to the bounding diagonal — the same 1e-6 the corpus-wide ratchet uses, held here with NO debt
// entry: measured worst 5.8e-10 (F7's fitted-BSpline corner rail) and 1.1e-11 (F6). The chord it replaces
// measures 0.2633 (F7's 89.4426 across its elliptic cylinder), three decades outside.
const ellipticRimOnFaceTol = 1e-6

// TestEllipticSurvivorRimIsCarried is the oracle gate for the elliptic survivor-rim carry
// (fillet_survivor_rim_ellipse.go).
//
// When a fillet corner pulls a survivor wall's rim endpoint back to its tangent point, the retained rim has
// to be re-derived from the PARENT rim's own parameters. Every helper that did so was written for a circle
// (projectOntoArcCircle / arcFrac / subArcOnParent), so an ELLIPTIC rim fell through to nil and the wall
// shipped a straight CHORD across itself — 89.44 off its own elliptic cylinder on F7, the corpus's largest
// per-face gross error (40793 against DRAWEXE). Three assertions, each falsifiable on its own:
//
//	(a) NO straight boundary edge on an EllipticalCylinder face — the direct anti-regression on the chord;
//	(b) every face's mesh area equals DRAWEXE's own, face by face, rank-paired;
//	(c) every boundary edge lies on the face it bounds, with no debt entry.
//
// Both fixtures are OCCT's own `ellipse w1 0 0 0 15 10` + `prism s w 2 0 10` + `tscale SCALE1=10`, i.e. an
// OBLIQUE elliptic prism (major 150, minor 100, extruded along (20,0,100)) blended at r=10 — F6 on one
// edge, F7 on three.
func TestEllipticSurvivorRimIsCarried(t *testing.T) {
	t.Parallel()
	for _, tc := range ellipticRimCases() {
		t.Run(tc.name, func(t *testing.T) {
			body := pinnedBody(t, "simple", tc.name)
			assertNoChordOnEllipticWall(t, tc.name, body)
			assertPerFaceAgainstDrawexe(t, tc, body)
			assertLoopSegmentsOnFaces(t, Record{Grid: "simple", Case: tc.name}, body, ellipticRimOnFaceTol)
		})
	}
}

// ellipticRimCases is the pinned population: the corpus's two elliptic-prism blends whose survivor cap rim
// is a geom.EllipticalArc the corner substitution re-trims. Rank pairing is sound here because both bodies'
// face areas are well separated (the closest pair differs by 1.4%), so a size-ordered pairing cannot
// mis-associate two faces.
func ellipticRimCases() []drawexeFaceCase {
	return []drawexeFaceCase{
		{name: "F6", totalArea: 133725, perFaceTol: ellipticRimPerFaceTol,
			drawexe: []float64{39942.6, 31060.7, 31060.7, 15496.3, 15496.3, 668.339}},
		{name: "F7", totalArea: 132309, perFaceTol: ellipticRimPerFaceTol,
			drawexe: []float64{39896.5, 31060.7, 27897.8, 14044.5, 13848.9, 2565.52, 2329.06, 601.505, 64.3501}},
	}
}

// assertNoChordOnEllipticWall fails when any EllipticalCylinder face is bounded by a straight edge that is
// not ON it — the chord signature the carry replaces. It is stated as "no straight edge OFF the wall"
// rather than "no straight edge at all" because an elliptic prism's own extrusion RULINGS are legitimately
// straight and do lie on the wall.
func assertNoChordOnEllipticWall(t *testing.T, name string, b *topo.Body) {
	t.Helper()
	for _, f := range b.Faces() {
		if _, elliptic := f.Geometry().(geom.EllipticalCylinder); !elliptic {
			continue
		}
		for _, e := range boundaryEdgesOf(f) {
			if _, straight := e.Geometry().(geom.LineSegment); !straight {
				continue
			}
			if d, ok := curveOffSurface(e.Geometry(), f.Geometry()); ok && d/boundingDiag(b) > ellipticRimOnFaceTol {
				t.Errorf("%s: elliptic wall (face %d) is still bounded by a straight CHORD (edge %d), %.6g off itself — the elliptic rim was not carried",
					name, f.ID(), e.ID(), d)
			}
		}
	}
}
