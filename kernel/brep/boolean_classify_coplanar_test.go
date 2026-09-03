// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// wedgeStepBlock is the corpus shape for the shared-plane classification defect (#3459), reduced from
// the multipoint disk's 523-face target to eighteen faces. Three properties reproduce it:
//
//   - a face in the plane z=0.3 — the step over y ≥ 0.5 — so a point in that plane makes every
//     candidate ray direction pierce it at t≈0 and the parity classifier finds no clean direction;
//   - a razor WEDGE standing on that plane, whose two walls converge, so a point near its tip lies
//     within the on-plane band of BOTH walls and the winding fallback zeroes exactly the faces that
//     would have decided the point (faceSolidAngle's on-plane rule);
//   - a bore, so the body carries a curved face and membership takes the ray-parity classifier. The
//     all-planar winding probe answers this shape correctly; the defect is the mixed one's.
//
// A point near the wedge tip in the plane z=0.3 is INTERIOR — material runs through the plane there —
// and both evaluators call it outside. That is the disk's three unpaired half-millimetre edges.
func wedgeStepBlock(t *testing.T, tip float64) *topo.Body {
	t.Helper()
	lower, err := SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 0.3), "lower")
	if err != nil {
		t.Fatalf("lower block: %v", err)
	}
	upper, err := SolidBlock(math.P3(0, 0, 0.3), math.P3(1, 0.5, 1), "upper")
	if err != nil {
		t.Fatalf("upper block: %v", err)
	}
	body, err := Boolean(Union, lower, upper)
	if err != nil {
		t.Fatalf("step block: %v", err)
	}
	if body, err = Boolean(Union, body, wedgeFin(tip)); err != nil {
		t.Fatalf("wedge fin: %v", err)
	}
	bore, err := SolidCylinder(math.P3(0.8, 0.8, -0.1), math.V3(0, 0, 1), 0.1, 0.5)
	if err != nil {
		t.Fatalf("bore: %v", err)
	}
	if body, err = Boolean(Difference, body, bore); err != nil {
		t.Fatalf("bore the step block: %v", err)
	}
	return body
}

// wedgeFin is the razor wedge standing on z=0.3: a triangular prism whose apex at (0.3, 0.5) opens to
// `tip` wide at x=0.4, so its two walls are within any given band of each other near the apex.
func wedgeFin(tip float64) *topo.Body {
	tri := []math.Point3{
		math.P3(0.3, 0.5, 0), math.P3(0.4, math.Scalar(0.5+tip), 0), math.P3(0.4, 0.5, 0),
	}
	verts := make([]math.Point3, 0, len(tri)*2)
	for _, p := range tri {
		verts = append(verts, math.P3(p.X, p.Y, 0.3))
	}
	for _, p := range tri {
		verts = append(verts, math.P3(p.X, p.Y, 1))
	}
	faces := [][]int{{2, 1, 0}, {3, 4, 5}}
	for i := range tri {
		next := (i + 1) % len(tri)
		faces = append(faces, []int{i, next, next + len(tri), i + len(tri)})
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, "wedge")
}

// wedgeTipPoint returns a point in the shared plane z=0.3, inside the wedge at half its local width,
// `along` of the way from the apex. The wedge is `tipWidth` wide at x=0.4 over a 0.1 run.
func wedgeTipPoint(along, tipWidth float64) math.Point3 {
	return math.P3(math.Scalar(0.3+along), math.Scalar(0.5+tipWidth/0.1*along/2), 0.3)
}

// TestUncoveredPointOnASharedPlaneIsInside pins the classification defect the multipoint disk exposed
// (#3459). Near the wedge tip the point is in the solid's INTERIOR — material runs through the plane
// z=0.3 there — yet both evaluators of the mixed membership oracle call it outside: the parity
// classifier because every direction grazes the step face at t≈0, the winding fallback because it
// zeroes the wedge walls the point sits between. Probing to each side of the shared plane answers it,
// and the two probes agreeing is the certificate that the point was never on the boundary.
func TestUncoveredPointOnASharedPlaneIsInside(t *testing.T) {
	t.Parallel()
	const tip = 3.4e-5 // the disk's own wedge depth
	body := wedgeStepBlock(t, tip)
	oracle := newInsideOracle(body, partitionFaces(body).allFaces())
	if _, planar := oracle.(*solidProbe); planar {
		t.Fatal("the bore left no curved face: this shape must take the ray-parity classifier")
	}
	step, ok := oracle.onPlaneStep()
	if !ok {
		t.Fatal("no probe step for the mixed oracle")
	}
	band := step / offPlaneProbeSteps
	degenerate := 0
	for _, along := range []float64{3e-4, 5.1e-4} { // wedge half-width below the on-plane band here
		p := wedgeTipPoint(along, tip)
		above := math.P3(p.X, p.Y, math.Scalar(0.3+100*band))
		below := math.P3(p.X, p.Y, math.Scalar(0.3-100*band))
		if !oracle.inside(above) || !oracle.inside(below) {
			t.Fatalf("premise at along=%g: the material must run THROUGH the plane at the wedge tip", along)
		}
		if !oracle.inside(p) {
			degenerate++ // the defect: the plain query calls an interior point outside
		}
		in, agreed := insideOffPlane(oracle, p, math.V3(0, 0, 1), step)
		if !agreed {
			t.Errorf("along=%g: the two sides disagreed, but the point is not on the boundary", along)
		}
		if !in {
			t.Errorf("along=%g: a point in the solid's interior classified as outside", along)
		}
		if !insidePlaneSafe(oracle, p, math.V3(0, 0, 1), step, true) {
			t.Errorf("along=%g: insidePlaneSafe classified an interior point as outside", along)
		}
	}
	if degenerate == 0 {
		t.Skip("the plain query answers every sample correctly: this shape no longer reproduces the " +
			"degeneracy, so the assertions above pass vacuously — re-derive the sample points")
	}
}

// TestCoplanarCoverReportsThePlaneWithoutTheCover is the signal insidePlaneSafe is gated on: a
// fragment point in a plane the other solid shares but on no face of it must report onPlane WITHOUT
// covered, so the classifier knows its volumetric query at that point is degenerate.
func TestCoplanarCoverReportsThePlaneWithoutTheCover(t *testing.T) {
	t.Parallel()
	body := wedgeStepBlock(t, 3.4e-5)
	var stepFace curvedFace
	for _, f := range partitionFaces(body).planar {
		pl, isPlane := f.surface.(geom.Plane)
		if !isPlane || float64(pl.Origin.Z) != 0.3 {
			continue
		}
		stepFace = f
	}
	if len(stepFace.loops) == 0 {
		t.Fatal("no face found in the shared plane z=0.3")
	}
	ux, _ := math.NewUnitVector3(1, 0, 0)
	uy, _ := math.NewUnitVector3(0, 1, 0)
	cap0 := curvedFace{surface: geom.Plane{Origin: math.P3(0, 0, 0.3), UAxis: ux, VAxis: uy}}
	covered, _, _ := coplanarCover(cap0, math.P3(0.25, 0.25, 0.3), []curvedFace{stepFace}, 1e-9)
	if covered {
		t.Error("a point at y=0.25 is not under the step face (y ≥ 0.5); coplanarCover must not cover it")
	}
	// The step face runs from y=0.5; a point 0.4 of a band short of it is inside that band, so the
	// query there IS degenerate, while one well clear of the edge is not.
	if _, _, degenerate := coplanarCover(cap0, math.P3(0.25, math.Scalar(0.5-0.4*1e-7), 0.3), []curvedFace{stepFace}, 1e-7); !degenerate {
		t.Error("a point inside the band of a coplanar face's boundary must report degenerate")
	}
	if _, _, degenerate := coplanarCover(cap0, math.P3(0.25, 0.25, 0.3), []curvedFace{stepFace}, 1e-7); degenerate {
		t.Error("a point 0.25 clear of the coplanar face's boundary must not report degenerate")
	}
	if _, _, degenerate := coplanarCover(cap0, math.P3(0.25, 0.25, 0.3), nil, 1e-7); degenerate {
		t.Error("no coplanar face at all must not report degenerate")
	}
}

// TestCoplanarIsFalseForANonPlanarFace pins the slotted screw's cross-hole panic (#3459). The mixed
// boolean's coplanar cover screens EVERY face of the other operand, cylinders and cones included, and
// coplanar took the plane before checking the kind. A cylinder's NormalAt(0,0) is a valid unit vector,
// so a cylinder whose normal happens to align with the plane's passed the parallel test and the type
// assertion panicked — recovered as a sick feature, which left the cut silently doing nothing.
func TestCoplanarIsFalseForANonPlanarFace(t *testing.T) {
	t.Parallel()
	ux, _ := math.NewUnitVector3(1, 0, 0)
	uy, _ := math.NewUnitVector3(0, 1, 0)
	uz, _ := math.NewUnitVector3(0, 0, 1)
	plane := curvedFace{surface: geom.Plane{Origin: math.P3(0, 0, 0), UAxis: ux, VAxis: uy}}
	// A cylinder about +Z: NormalAt(0,0) is radial (+X here), so it is never parallel to +Z — take the
	// pairing that DOES align, a cylinder about +X against a plane whose normal is +X.
	cyl := curvedFace{surface: geom.Cylinder{Origin: math.P3(0, 0, 0), AxisDir: uz, Ref: ux, Radius: 1}}
	yzPlane := curvedFace{surface: geom.Plane{Origin: math.P3(0, 0, 0), UAxis: uy, VAxis: uz}}
	for _, c := range []struct {
		name string
		a, b curvedFace
	}{
		{"plane vs cylinder", plane, cyl},
		{"cylinder vs plane", cyl, plane},
		{"aligned-normal plane vs cylinder", yzPlane, cyl},
		{"cylinder vs itself", cyl, cyl},
	} {
		if coplanar(c.a, c.b) {
			t.Errorf("%s: a non-planar face is coplanar with nothing", c.name)
		}
	}
	if !coplanar(plane, plane) {
		t.Error("a plane must still be coplanar with itself")
	}
}

// TestUVSegIndexFindsEverySegmentTheScanWould pins the equivalence the grid rests on (#3459): a query
// must see every segment a linear scan could have matched, so recoverEdge's answer is unchanged. It is
// checked on a chart-spanning segment (which the index files as wide) beside short ones, and at points
// on, beside and far from each.
func TestUVSegIndexFindsEverySegmentTheScanWould(t *testing.T) {
	t.Parallel()
	segs := []uvSeg{
		{a: math.P2(0, 0), b: math.P2(1, 0)},
		{a: math.P2(1, 0), b: math.P2(1, 1)},
		{a: math.P2(0, 0), b: math.P2(1000, 1000)}, // spans the chart: filed wide
		{a: math.P2(0.5, 0.5), b: math.P2(0.5, 0.6)},
	}
	ix := newUVSegIndex(segs)
	for _, p := range []math.Point2{
		math.P2(0.5, 0), math.P2(1, 0.5), math.P2(0.5, 0.55), math.P2(500, 500),
		math.P2(0.5, math.Scalar(tjTol/2)), math.P2(-5, -5), math.P2(0, 0),
	} {
		want := -1
		best := tjTol
		for i, s := range segs {
			if d := perpDistToSeg(p, s.a, s.b); d < best {
				want, best = i, d
			}
		}
		got := -1
		best = tjTol
		for _, i := range ix.near(p) {
			if d := perpDistToSeg(p, segs[i].a, segs[i].b); d < best {
				got, best = i, d
			}
		}
		if got != want {
			t.Errorf("at %v the index matched segment %d, the scan matched %d", p, got, want)
		}
	}
}

// TestUVSegIndexHandlesDegenerateInputs: an empty list and a list with no extent must still answer,
// because a chart can produce either and the caller has no special case for them.
func TestUVSegIndexHandlesDegenerateInputs(t *testing.T) {
	t.Parallel()
	if got := newUVSegIndex(nil).near(math.P2(0, 0)); len(got) != 0 {
		t.Errorf("an empty index returned %d candidates, want none", len(got))
	}
	point := []uvSeg{{a: math.P2(2, 3), b: math.P2(2, 3)}, {a: math.P2(2, 3), b: math.P2(2, 3)}}
	if got := newUVSegIndex(point).near(math.P2(2, 3)); len(got) != len(point) {
		t.Errorf("a zero-extent index returned %d candidates, want all %d", len(got), len(point))
	}
}
