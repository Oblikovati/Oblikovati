// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The far-end trim's ±BRANCH guard — OCCT blend/complex/D8 (`CFI_id150018.rle`, `blend result a 30 a_20`:
// a 210.916×510.843 rounded-rectangle plate, r=24 corners, 100 tall, filleted r=30 along the top edge of
// one long wall).
//
// THE DEFECT. fillet_farend_trim.go slides each station of a terminal section cap along the filleted edge's
// axis onto the stop wall and takes the NEAREST crossing. On a wall SYMMETRIC about the section plane the
// two crossings are EXACTLY equidistant, so "nearest" was resolved by geom.IntersectCurveSurface's output
// order — independently at each of the 33 stations. That symmetry is a configuration class, not a fluke: the
// stop wall here is a corner cylinder TANGENT to the filleted wall at the terminal vertex, so its axis lies
// IN the section plane. The station list alternated between the two branches (dy 0, −1.32, +2.63, +3.93,
// … +23.37, −23.97, +24.00 …) and the cubic fitted through it was on neither: 18.8877 off the host cylinder
// it bounds (rel 0.0336 — the corpus's second-largest off-surface residual) and 2.009 off the fillet band.
//
// WHY THIS CASE IS NOT ALSO AN AREA GATE. D8's whole-body area cannot be compared with OCCT's: OCCT
// propagates the fillet along the whole tangent-continuous closed top-edge loop (18 faces, 8 edges), we
// fillet the one picked edge (11 faces). So the gates here are the case's OWN exact geometry — the trim
// curve on both of its faces, on a single branch, and the fillet band tiling its closed-form area — none of
// which depend on which solid OCCT chose to build. See stopface-reversed-report.md.

// d8TrimLineagePrefix is the lineage prefix of the two far-end trim edges: the fillet cylinder crossed with
// an imported host face. Found by PROVENANCE, not by geometry, so the guard cannot silently start measuring
// a different edge.
const d8TrimLineagePrefix = "import:step#16:edge#1/fillet:cyl#0/fillet:x#0/import:step#16:face#"

// TestFarEndTrimCurveStaysOnOneWallBranch asserts each of complex/D8's two far-end trim curves lies on both
// faces it bounds and does not cross back over its own section plane (the direct "one branch" statement),
// and that the fillet band therefore tiles its closed-form area.
func TestFarEndTrimCurveStaysOnOneWallBranch(t *testing.T) {
	rec := corpusRecord(t, "complex", "D8")
	body := gridCaseBody(t, rec)
	band := faceByLineage(t, body, "import:step#16:edge#1/fillet:cyl#0")
	bounds := boundaryEdgesWithLineagePrefix(band, d8TrimLineagePrefix)
	if len(bounds) != 4 {
		t.Fatalf("complex/D8: found %d band boundary edges under %q, want 4 (2 axial rulings + 2 far-end trims)",
			len(bounds), d8TrimLineagePrefix)
	}
	axis := band.Geometry().(geom.Cylinder).AxisDir.AsVector()
	trims := 0
	for _, e := range bounds {
		assertTrimOnEveryFaceItBounds(t, e, loopSegOnFaceTol*boundingDiag(body))
		// A band ruling is a straight LineSegment by construction; a band∩curved-wall trim is not. That is
		// how the two far-end trims are told from the two rulings without hard-coding a host face index.
		wall, curved := curvedStopWallOf(e, band)
		if !curved {
			continue
		}
		trims++
		assertTrimStaysOnOneBranch(t, e, wall.Origin, axis)
	}
	if trims != 2 {
		t.Fatalf("complex/D8: %d of the band's 4 boundary edges are curved far-end trims, want 2", trims)
	}
	assertBandTilesItsClosedFormArea(t, body, band)
	assertLoopSegmentsOnFaces(t, rec, body, 0) // default budget: complex/D8 carries NO off-surface debt entry
}

// assertTrimOnEveryFaceItBounds fails when the trim curve leaves any face it bounds — the fillet band on one
// side, the imported host wall on the other. Both must hold: the zigzag was off BOTH (18.8877 / 2.009).
func assertTrimOnEveryFaceItBounds(t *testing.T, e *topo.Edge, budget float64) {
	t.Helper()
	for _, f := range e.Faces() {
		d, ok := curveOffSurface(e.Geometry(), f.Geometry())
		if ok && d > budget {
			t.Errorf("far-end trim edge %d (%T) leaves face %d (%T) by %.6g (budget %.4g)",
				e.ID(), e.Geometry(), f.ID(), f.Geometry(), d, budget)
		}
	}
}

// assertTrimStaysOnOneBranch fails when the curve's axial offset from the terminal SECTION PLANE changes
// SIGN: the stop wall's two branches lie on opposite sides of that plane, so a sign change is a station list
// that jumped between them. The plane is located by the stop cylinder's own origin, because the wall is
// tangent to the filleted face AT the terminal vertex — that tangency is precisely why its axis lies in the
// plane and why the two crossings tie. Stated on the SHIPPED curve rather than on the station list, so it
// holds whatever the fit.
func assertTrimStaysOnOneBranch(t *testing.T, e *topo.Edge, onPlane math.Point3, axis math.Vector3) {
	t.Helper()
	c := e.Geometry()
	lo, hi := c.Domain()
	const stations = 64
	tol := 1e-9 * c.PointAt(lo).DistanceTo(c.PointAt(hi))
	seen := 0
	for i := 0; i <= stations; i++ {
		off := onPlane.VectorTo(c.PointAt(lo + (hi-lo)*float64(i)/stations)).Dot(axis)
		if stdmath.Abs(off) <= tol {
			continue
		}
		if sign := sideSign(off); seen != 0 && sign != seen {
			t.Fatalf("far-end trim edge %d crosses its own section plane at station %d (axial offset %+.6g): "+
				"the slide jumped between the stop wall's two branches", e.ID(), i, off)
		} else {
			seen = sign
		}
	}
}

// curvedStopWallOf returns the CYLINDRICAL stop wall a curved far-end trim bounds, and false for a straight
// band ruling (whose other face is the filleted edge's own A/B face, not a stop wall).
func curvedStopWallOf(e *topo.Edge, band *topo.Face) (geom.Cylinder, bool) {
	if _, straight := geom.InnerCurve(e.Geometry()).(geom.LineSegment); straight {
		return geom.Cylinder{}, false
	}
	for _, f := range e.Faces() {
		if cyl, ok := f.Geometry().(geom.Cylinder); ok && f != band {
			return cyl, true
		}
	}
	return geom.Cylinder{}, false
}

// sideSign is +1 for a positive axial offset and −1 otherwise.
func sideSign(off float64) int {
	if off > 0 {
		return 1
	}
	return -1
}

// assertBandTilesItsClosedFormArea is the SIDE oracle: being on ONE branch does not say WHICH one, because
// the wall's two branches are mirror images of each other. The band's area does — reaching PAST the terminal
// vertex (the side the stop face's own quarter-cylinder lies on) versus retreating from it differ by 2∫h,
// which is 3057 of D8's 23340. Closed form, taken entirely from the shipped body's own surfaces: the band is
// a radius-rf cylinder whose ruling at angle φ runs the picked edge's length L plus the two overhangs
// h(φ) = √(R²−δ(φ)²), δ being the ruling's distance from the stop cylinder's axis along the two axes' common
// perpendicular. The angular interval is read off the band's own two straight rulings, so no parameterization
// convention is assumed.
func assertBandTilesItsClosedFormArea(t *testing.T, body *topo.Body, band *topo.Face) {
	t.Helper()
	bc := band.Geometry().(geom.Cylinder)
	w1 := faceByLineage(t, body, "import:step#16:face#1").Geometry().(geom.Cylinder)
	w2 := faceByLineage(t, body, "import:step#16:face#2").Geometry().(geom.Cylinder)
	lo, hi := bandRulingAngles(t, band, bc)
	length := stdmath.Abs(w1.Origin.VectorTo(w2.Origin).Dot(bc.AxisDir.AsVector()))
	want := bc.Radius * simpson(func(phi float64) float64 {
		p := bandPoint(bc, phi)
		return length + overhang(bc, w1, p) + overhang(bc, w2, p)
	}, lo, hi)
	// Measured on the mesh the body actually SHIPS, not on the solo per-face tessellation. The two used
	// to differ: the cross-face conformance repair adopted a boundary-only re-mesh of this band on a fold
	// count alone, and that re-mesh — with no interior node between the band's two straight axial rulings
	// — realised the full 90° arc as its chord, shipping 21339.8 for a face whose solo mesh (and closed
	// form) is 23340. The solo assertion could not see it. See kernel/ops/conformance_adopt.go.
	got := ops.MeshArea(shippedFaceMesh(t, body, band))
	if rel := stdmath.Abs(got-want) / want; rel > 1e-4 {
		t.Errorf("complex/D8 fillet band tiles %.6g, closed form %.6g (rel %+.4f%%) — the far-end trim took "+
			"the wrong side of the stop wall, or the conformance repair under-tiled the band", got, want, (got-want)/want*100)
	}
}

// shippedFaceMesh returns face f's mesh as the BODY tessellation pipeline produces it — the per-face
// meshing plus the cross-face conformance repair's adoption decision — which is what every downstream
// consumer sees, and is not the same thing as TessellateFace(f) on its own.
func shippedFaceMesh(t *testing.T, body *topo.Body, f *topo.Face) *ops.Mesh {
	t.Helper()
	facets := ops.CalculateBodyFacets(body, ops.PropertyQuality())
	for i, g := range facets.Faces {
		if g == f {
			return facets.FaceMeshes[i]
		}
	}
	t.Fatalf("face %d is not in its own body's facet set", f.ID())
	return nil
}

// bandRulingAngles returns the band's angular extent about its own axis, read off the midpoints of its two
// STRAIGHT boundary rulings (the contacts with the two filleted faces). It fails unless they span the
// quarter turn a 90° convex fillet must subtend — the sanity check that the interval was read correctly.
func bandRulingAngles(t *testing.T, band *topo.Face, bc geom.Cylinder) (lo, hi float64) {
	t.Helper()
	var angles []float64
	for _, e := range boundaryEdgesOf(band) {
		if seg, straight := geom.InnerCurve(e.Geometry()).(geom.LineSegment); straight {
			a, b := seg.Domain()
			angles = append(angles, bandAngle(bc, seg.PointAt((a+b)/2)))
		}
	}
	if len(angles) != 2 {
		t.Fatalf("band has %d straight rulings, want 2", len(angles))
	}
	lo, hi = stdmath.Min(angles[0], angles[1]), stdmath.Max(angles[0], angles[1])
	if hi-lo > stdmath.Pi {
		lo, hi = hi, lo+2*stdmath.Pi // the quarter is the interval that WRAPS the atan2 branch cut
	}
	if stdmath.Abs((hi-lo)-stdmath.Pi/2) > 1e-9 {
		t.Fatalf("band rulings subtend %.9g rad, want a quarter turn", hi-lo)
	}
	return lo, hi
}

// bandAngle is p's angle about the band cylinder's axis in the cylinder's own (Ref, Axis×Ref) frame.
func bandAngle(bc geom.Cylinder, p math.Point3) float64 {
	v := bc.Origin.VectorTo(p)
	u := bc.Ref.AsVector()
	return stdmath.Atan2(v.Dot(bc.AxisDir.AsVector().Cross(u)), v.Dot(u))
}

// bandPoint is the point on the band cylinder at angle phi, in the section plane through its origin.
func bandPoint(bc geom.Cylinder, phi float64) math.Point3 {
	u := bc.Ref.AsVector()
	w := bc.AxisDir.AsVector().Cross(u)
	return bc.Origin.TranslateBy(u.Scale(math.Scalar(bc.Radius * stdmath.Cos(phi))).
		Add(w.Scale(math.Scalar(bc.Radius * stdmath.Sin(phi)))))
}

// overhang is how far past the terminal section plane the band's ruling through p reaches before it meets the
// stop cylinder: √(R²−δ²) with δ the ruling's distance from that cylinder's axis, measured along the common
// perpendicular of the two axes. 0 when the ruling misses the cylinder entirely.
func overhang(band, wall geom.Cylinder, p math.Point3) float64 {
	perp, err := math.UnitVector3FromVector(band.AxisDir.AsVector().Cross(wall.AxisDir.AsVector()))
	if err != nil {
		return 0
	}
	delta := wall.Origin.VectorTo(p).Dot(perp.AsVector())
	return stdmath.Sqrt(stdmath.Max(wall.Radius*wall.Radius-delta*delta, 0))
}

// simpson integrates f over [a,b] by composite Simpson on 4096 spans — the band's integrand is smooth
// except for a √ endpoint, where the residual is decades under the 1e-4 gate.
func simpson(f func(float64) float64, a, b float64) float64 {
	const spans = 4096
	h := (b - a) / spans
	sum := f(a) + f(b)
	for i := 1; i < spans; i++ {
		w := 2.0
		if i%2 == 1 {
			w = 4.0
		}
		sum += w * f(a+float64(i)*h)
	}
	return sum * h / 3
}

// faceByLineage returns the body's single face with the given lineage key, failing when there is not
// exactly one.
func faceByLineage(t *testing.T, b *topo.Body, key string) *topo.Face {
	t.Helper()
	var out []*topo.Face
	for _, f := range b.Faces() {
		if f.Lineage().KeyString() == key {
			out = append(out, f)
		}
	}
	if len(out) != 1 {
		t.Fatalf("found %d faces with lineage %q, want exactly 1", len(out), key)
	}
	return out[0]
}

// boundaryEdgesWithLineagePrefix returns f's distinct boundary edges whose lineage key starts with prefix.
func boundaryEdgesWithLineagePrefix(f *topo.Face, prefix string) []*topo.Edge {
	var out []*topo.Edge
	for _, e := range boundaryEdgesOf(f) {
		if strings.HasPrefix(e.Lineage().KeyString(), prefix) {
			out = append(out, e)
		}
	}
	return out
}

// corpusRecord returns the corpus record for a grid+case, failing when it is absent.
func corpusRecord(t *testing.T, grid, name string) Record {
	t.Helper()
	for _, r := range Corpus() {
		if r.Grid == grid && r.Case == name {
			return r
		}
	}
	t.Fatalf("no corpus record for %s/%s", grid, name)
	return Record{}
}

// gridCaseBody runs one record's real fillet and returns its single result body (the any-grid counterpart of
// caseResultBody, which is hard-wired to the simple grid).
func gridCaseBody(t *testing.T, r Record) *topo.Body {
	t.Helper()
	body, err := importInput(filepath.Join(CorpusFixtureDir(), r.InputStep))
	if err != nil {
		t.Skipf("%s/%s import-divergence (not a fillet defect): %v", r.Grid, r.Case, err)
	}
	sets, ok := scoreLocate(r, body)
	if !ok {
		t.Skipf("%s/%s picks could not be located on the imported body", r.Grid, r.Case)
	}
	res, okFillet, reason := runFillet(body, sets)
	if !okFillet || len(res) != 1 || res[0] == nil {
		t.Fatalf("%s/%s fillet unhealthy: ok=%v reason=%q results=%d", r.Grid, r.Case, okFillet, reason, len(res))
	}
	return res[0]
}
