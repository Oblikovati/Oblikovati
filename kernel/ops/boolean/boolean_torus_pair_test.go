// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The TORUS × TORUS corpus at the boolean (ADR-0066, Oblikovati#3514) — a fillet meeting a fillet, two
// chain links, a torus boss on a ring.
//
// ADR-0061 stage 5 recorded the pair as a standing refusal, and boolean_torus_section_test.go's
// TestATorusPairIsRefusedByName was its guard. The premise was that the torus reduction needs an
// implicit QUADRIC on the other side; the reduction actually needs an implicit form whose restriction
// to a CIRCLE is degree two, and a torus's quartic is one. These are that guard's positive form.
//
// The certification has three layers, and none of them is a whole-body number on its own:
//
//   - PER FACE — every face's surface is one of the two OPERANDS (not merely "a torus"), and every
//     face's loop count is asserted. A body that measures right can still be the wrong shape.
//   - BOUNDARY — each result's analytic surface area against a chart integral of the two tori's own
//     membership. The three operations give three independent equations in the four regions the
//     section cuts the two boundaries into, so matching all three pins every region: a face that kept
//     the wrong SIDE moves this number by the whole difference, which a volume cannot see.
//   - WHOLE BODY — Requicha's identity at BOTH facetings, and the volumes against a stratified
//     membership integral. Smoke tests, in that order, and labelled as such.

// torusPairRow is one corpus pair: two solid tori and the face census each operation must leave.
type torusPairRow struct {
	name string
	a, b torusSpec
	// loops is the number of boundary loops on each face of each result, in the body's own face order.
	// Its length is the face count, so one field asserts both.
	loops map[ops.PartFeatureOperation][]int
}

// torusSpec is a torus written as its constructor's arguments, so the oracle reads the fixture rather
// than the body the kernel built from it.
type torusSpec struct {
	centre math.Point3
	axis   math.Vector3
	major  float64
	minor  float64
}

// body builds the solid this spec describes.
func (s torusSpec) body(t *testing.T, name string) *topo.Body {
	t.Helper()
	b, err := brep.SolidTorus(s.centre, s.axis, s.major, s.minor, name)
	if err != nil {
		t.Fatalf("SolidTorus(%v, %v, %g, %g): %v", s.centre, s.axis, s.major, s.minor, err)
	}
	return b
}

// volume is the exact 2π²Rr².
func (s torusSpec) volume() float64 { return 2 * stdmath.Pi * stdmath.Pi * s.major * s.minor * s.minor }

// area is the exact 4π²Rr — the whole of this torus's boundary.
func (s torusSpec) area() float64 { return 4 * stdmath.Pi * stdmath.Pi * s.major * s.minor }

// contains is the torus's own analytic membership, written from the spec: the point's distance from the
// tube's centre circle is at most the tube radius.
func (s torusSpec) contains(p math.Point3) bool {
	w := s.centre.VectorTo(p)
	axis := unitOf(s.axis)
	z := float64(w.Dot(axis))
	d := float64(w.Sub(axis.Scale(math.Scalar(z))).Length()) - s.major
	return d*d+z*z <= s.minor*s.minor
}

// pointAt is the spec's own chart, used by the area oracle so the integral runs on the fixture's
// geometry rather than on the surface the kernel put on a face.
func (s torusSpec) pointAt(u, v float64) math.Point3 {
	axis := unitOf(s.axis)
	e1 := unitOf(math.AnyPerpendicular(axis.AsUnit()).AsVector())
	e2 := axis.Cross(e1)
	rho := s.major + s.minor*stdmath.Cos(v)
	radial := e1.Scale(math.Scalar(stdmath.Cos(u))).Add(e2.Scale(math.Scalar(stdmath.Sin(u))))
	return s.centre.TranslateBy(radial.Scale(math.Scalar(rho))).TranslateBy(axis.Scale(math.Scalar(s.minor * stdmath.Sin(v))))
}

// unitOf normalises a vector for the oracle's own frame.
func unitOf(v math.Vector3) math.Vector3 {
	return v.Scale(math.Scalar(1 / float64(v.Length())))
}

// torusPairCorpusRows are the shapes. Each is a pair the reduction solves and the general pipeline
// builds; the pairs it refuses are rows of their own in kernel/geom's torus_torus_test.go and in
// boolean_named_decline_test.go.
func torusPairCorpusRows() []torusPairRow {
	ring := torusSpec{math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5}
	return []torusPairRow{{
		// Two chain links, and the exact fixture TestATorusPairIsRefusedByName drove.
		name: "linked rings",
		a:    ring,
		b:    torusSpec{math.P3(5, 0, 0), math.V3(1, 0, 0), 5, 1.5},
		// Join: each ring keeps its own surface, holed where the other entered. Cut: the ring holed the
		// same way, plus the link's two patches lining the cavity. Intersect: those two patches from
		// each surface, four simple lenses.
		loops: map[ops.PartFeatureOperation][]int{
			ops.Join: {2, 2}, ops.Cut: {2, 1, 1}, ops.Intersect: {1, 1, 1, 1},
		},
	}, {
		// A small ring threaded through the big one's hole and out through its tube.
		name: "small ring through the hole",
		a:    ring,
		b:    torusSpec{math.P3(3.5, 0, 0), math.V3(1, 0, 0), 2, 0.7},
		loops: map[ops.PartFeatureOperation][]int{
			ops.Join: {2, 2}, ops.Cut: {2, 1, 1}, ops.Intersect: {1, 1, 1, 1},
		},
	}, {
		// COAXIAL: two rings on one axis whose meridian circles cross. Their section is whole tube
		// CIRCLES, so the reduction never resolves an azimuth at all — a different family of the same
		// classification, and the one a pair of concentric fillets lands in. Every face is a band
		// between two of those circles, so every face carries two loops in every operation.
		name: "coaxial rings",
		a:    ring,
		b:    torusSpec{math.P3(0, 0, 0), math.V3(0, 0, 1), 6, 1.5},
		loops: map[ops.PartFeatureOperation][]int{
			ops.Join: {2, 2}, ops.Cut: {2, 2}, ops.Intersect: {2, 2},
		},
	}}
}

// TestATorusPairBuildsAndItsFacesAreTheRightPatches is the per-face layer. Every operation must build a
// valid closed manifold solid claiming an EXACT boundary, every face must sit on one of the two operand
// surfaces, and the area each operand contributes must be the area the analytic oracle says that
// operation keeps of it.
func TestATorusPairBuildsAndItsFacesAreTheRightPatches(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier: `make test-corpus`")
	}
	t.Parallel()
	for _, row := range torusPairCorpusRows() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			a, b := row.a.body(t, "a"), row.b.body(t, "b")
			for _, op := range []ops.PartFeatureOperation{ops.Join, ops.Cut, ops.Intersect} {
				res := builtTorusPair(t, op, a, b)
				assertEveryFaceIsAnOperandPatch(t, op, res, row)
				assertBoundaryAreaMatchesTheOracle(t, op, res, row)
			}
		})
	}
}

// builtTorusPair runs one boolean and asserts the post-conditions every kernel operation owes.
func builtTorusPair(t *testing.T, op ops.PartFeatureOperation, a, b *topo.Body) *topo.Body {
	t.Helper()
	res, err := ops.Boolean(op, a, b)
	if err != nil {
		t.Fatalf("%v: %v", op, err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("%v: not a valid closed manifold solid: %+v", op, r)
	}
	if tol := res.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("%v reports AchievedBoundaryTolerance %g, want 0 — the section is a closed form", op, tol)
	}
	return res
}

// assertEveryFaceIsAnOperandPatch checks each face's surface against the two operand SPECS, not merely
// against the kind geom.Torus: a face carrying some third torus would pass a kind census and be wrong.
// The loop count is asserted beside it, because a patch with the right area and the wrong number of
// boundaries is the other way a result looks right and is not.
func assertEveryFaceIsAnOperandPatch(t *testing.T, op ops.PartFeatureOperation, res *topo.Body, row torusPairRow) {
	t.Helper()
	want := row.loops[op]
	if len(res.Faces()) != len(want) {
		t.Errorf("%v: %d faces, want %d", op, len(res.Faces()), len(want))
		return
	}
	for i, f := range res.Faces() {
		s, ok := f.Geometry().(geom.Torus)
		if !ok {
			t.Errorf("%v: face %d carries %T, want one of the operands' torus surfaces", op, i, f.Geometry())
			continue
		}
		if whichOperand(s, row) < 0 {
			t.Errorf("%v: face %d sits on a torus (R=%g r=%g at %v) that is neither operand", op, i, s.MajorRadius, s.MinorRadius, s.Center)
		}
		if n := len(f.Loops()); n != want[i] {
			t.Errorf("%v: face %d has %d loops, want %d", op, i, n, want[i])
		}
	}
}

// whichOperand returns 0 or 1 for the operand a face's surface belongs to, or −1 for neither. It
// compares the radii and the centre, which is what distinguishes the two operands here.
func whichOperand(s geom.Torus, row torusPairRow) int {
	for i, spec := range []torusSpec{row.a, row.b} {
		if sameTorus(s, spec) {
			return i
		}
	}
	return -1
}

// sameTorus reports the surface being the spec's own torus, to the weld at the fixture's own size.
func sameTorus(s geom.Torus, spec torusSpec) bool {
	weld := geom.ResolutionForSize(spec.major + spec.minor).Weld()
	return stdmath.Abs(s.MajorRadius-spec.major) <= weld && stdmath.Abs(s.MinorRadius-spec.minor) <= weld &&
		float64(s.Center.DistanceTo(spec.centre)) <= weld
}

// assertBoundaryAreaMatchesTheOracle is the BOUNDARY gate: the result's analytic surface area against
// the area the oracle says that operation keeps of the two operands' boundaries, integrated on each
// operand's own chart from the two specs' analytic membership.
//
// It is read from the ANALYTIC integrator, never from the mesh. Mass properties integrate the B-rep,
// and "an oracle that gates a result must be more exact than the result it gates" cuts the other way
// too: the faceted area of these faces is not the quantity under test, and one of them is a known
// tessellator gap (TestTheHoledTorusFaceDeclinesItsMeshByName).
//
// A whole-body area is a weak gate for ONE operation and a strong one for three. The section cuts the
// two boundaries into four regions — each operand's boundary inside and outside the other — and the
// three operations keep three different pairs of them. Matching all three therefore pins all four, so a
// face that kept the complement of the region it should have cannot hide.
func assertBoundaryAreaMatchesTheOracle(t *testing.T, op ops.PartFeatureOperation, res *topo.Body, row torusPairRow) {
	t.Helper()
	inA := keptBoundaryArea(row.a, row.b)
	inB := keptBoundaryArea(row.b, row.a)
	want := map[ops.PartFeatureOperation]float64{
		ops.Join:      row.a.area() - inA + row.b.area() - inB,
		ops.Cut:       row.a.area() - inA + inB,
		ops.Intersect: inA + inB,
	}[op]
	got := query.BodyGeometryProperties(res, ops.PropertyQuality()).Area
	if rel := stdmath.Abs(got-want) / want; rel > torusAreaOracleTol {
		t.Errorf("%v: boundary area %.5f, the analytic oracle says %.5f (rel %.4g > %.4g)", op, got, want, rel, torusAreaOracleTol)
	}
}

// torusAreaOracleTol is the CHART INTEGRAL's own error, not the kernel's. At torusAreaGrid the midpoint
// rule's boundary cells dominate and carry O(1/N) of the region's perimeter. Measured over the nine
// rows (three pairs × three operations) the worst relative disagreement is 9.3e-4 and the best 2.0e-7,
// and this allows three times the worst — the kernel's areas agree far more closely than the oracle can
// resolve, so tightening it would be measuring the oracle.
const torusAreaOracleTol = 3e-3

// keptBoundaryArea is the area of the part of self's boundary that lies INSIDE other, integrated on
// self's own chart with the element r(R + r·cos v) du dv. Its complement is self.area() less this, and
// the caller combines the two per operation.
func keptBoundaryArea(self, other torusSpec) float64 {
	total, step := 0.0, 2*stdmath.Pi/torusAreaGrid
	for i := range torusAreaGrid {
		v := (float64(i) + 0.5) * step
		element := self.minor * (self.major + self.minor*stdmath.Cos(v)) * step * step
		for j := range torusAreaGrid {
			if other.contains(self.pointAt((float64(j)+0.5)*step, v)) {
				total += element
			}
		}
	}
	return total
}

// torusAreaGrid is the chart integral's side. 1200² midpoint cells put the oracle's own error at a few
// parts in ten thousand on these regions and cost about a second per pair.
const torusAreaGrid = 1200

// TestTheHoledTorusFaceDeclinesItsMeshByName pins a KNOWN GAP one layer downstream, so that it cannot
// become a silent one.
//
// The B-rep is exact — assertBoundaryAreaMatchesTheOracle reads it against an independent oracle and it
// agrees to a part in ten thousand. Its display MESH is not: the torus face that keeps its whole surface
// less one window, bounded by a torus×torus section loop, is meshed over the surface's whole domain,
// covering material the face does not carry. The tessellator does not do that quietly — it records
// tessellate.trim-ignored-full-domain and tessellate.chart-mesher-declined as Defects — and this row
// asserts that it keeps saying so, which is the contract the ground rules actually impose ("never
// degrade silently"). The row inverts when the chart mesher takes the shape (ADR-0066's follow-up).
//
// The gap is specific to this face: the same ring cut by an AXIAL DRILL leaves a two-loop torus face
// whose mesh IS bounded by its rim (measured: 291.88 against a whole-torus 296.09, no diagnostic).
func TestTheHoledTorusFaceDeclinesItsMeshByName(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier: `make test-corpus`")
	}
	t.Parallel()
	row := torusPairCorpusRows()[1] // the small ring through the hole
	res := builtTorusPair(t, ops.Cut, row.a.body(t, "a"), row.b.body(t, "b"))
	found := false
	for _, d := range query.BodyMeshDiagnostics(res, ops.PropertyQuality()) {
		if d.Code == tessellate.CodeTrimIgnoredFullDomain {
			found = true
			if d.Severity != diag.Defect {
				t.Errorf("the dropped trim is recorded as %v, want a Defect", d.Severity)
			}
		}
	}
	if !found {
		t.Errorf("the holed torus face meshed without recording %q; either the mesher took the shape — "+
			"invert this row — or the degradation went silent", tessellate.CodeTrimIgnoredFullDomain)
	}
}

// TestATorusPairSatisfiesRequichaAtBothFacetings is the whole-body layer, and it is a SMOKE TEST beside
// the per-face gate above rather than a proof on its own.
//
// Requicha's identity ties three bodies built by three separate trims of one section to each other and
// to the operands' own analytic volumes. It is asserted at BOTH facetings — the display default's ~36
// facets per circle, which under-reports a curved solid's volume by ~0.64%, and the property faceting —
// because an identity that held only at one of them would be an artifact of that faceting rather than a
// statement about the bodies. Each faceting's own bias cancels out of the identity, so the tolerance is
// the kernel's arithmetic and not the mesh's.
//
// The volumes are then read against a stratified Monte Carlo of the two tori's ANALYTIC membership,
// which is independent of the kernel's classifier — a body can satisfy the identity and still be the
// wrong shape if all three are wrong together.
func TestATorusPairSatisfiesRequichaAtBothFacetings(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier: `make test-corpus`")
	}
	t.Parallel()
	for _, row := range torusPairCorpusRows() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			a, b := row.a.body(t, "a"), row.b.body(t, "b")
			built := map[ops.PartFeatureOperation]*topo.Body{}
			for _, op := range []ops.PartFeatureOperation{ops.Join, ops.Cut, ops.Intersect} {
				built[op] = builtTorusPair(t, op, a, b)
			}
			for _, q := range []struct {
				name string
				q    ops.Quality
			}{{"display faceting", ops.DefaultQuality()}, {"property faceting", ops.PropertyQuality()}} {
				vol := func(op ops.PartFeatureOperation) float64 {
					return query.BodyGeometryProperties(built[op], q.q).Volume
				}
				join, cut, lens := vol(ops.Join), vol(ops.Cut), vol(ops.Intersect)
				assertRelative(t, q.name+": V(∪) + V(∩)", join+lens, row.a.volume()+row.b.volume(), requichaTol)
				assertRelative(t, q.name+": V(−) + V(∩)", cut+lens, row.a.volume(), requichaTol)
			}
			assertTorusVolumesAgainstMembership(t, row, built)
		})
	}
}

// assertTorusVolumesAgainstMembership reads each result's volume against the stratified integral of the
// two specs' own membership.
func assertTorusVolumesAgainstMembership(t *testing.T, row torusPairRow, built map[ops.PartFeatureOperation]*topo.Body) {
	t.Helper()
	whole := built[ops.Join].RangeBox()
	for _, op := range []ops.PartFeatureOperation{ops.Join, ops.Cut, ops.Intersect} {
		box := whole
		if op == ops.Intersect {
			box = built[ops.Intersect].RangeBox()
		}
		got := query.BodyGeometryProperties(built[op], ops.PropertyQuality()).Volume
		assertRelative(t, op.String()+" against the membership integral", got, torusMembershipVolume(op, row, box), membershipTol)
	}
}

// torusMembershipVolume integrates "inside a op inside b" over the box from the two SPECS' analytic
// membership — deliberately not the kernel's classifier. The lattice is stratified and the seed fixed,
// so the number is the same on every run and platform (see skewMembershipVolume for why plain triples
// from one stream carry a seed-dependent bias that does not shrink with the sample count).
func torusMembershipVolume(op ops.PartFeatureOperation, row torusPairRow, box math.Box) float64 {
	rng := rand.New(rand.NewSource(29))
	span := box.Max.AsVector().Sub(box.Min.AsVector())
	hits := 0
	for i := range skewStrata {
		for j := range skewStrata {
			for k := range skewStrata {
				p := math.P3(
					math.Scalar(float64(box.Min.X)+jitteredStratum(rng, i)*float64(span.X)),
					math.Scalar(float64(box.Min.Y)+jitteredStratum(rng, j)*float64(span.Y)),
					math.Scalar(float64(box.Min.Z)+jitteredStratum(rng, k)*float64(span.Z)))
				if keepsUnderOp(op, row.a.contains(p), row.b.contains(p)) {
					hits++
				}
			}
		}
	}
	return float64(span.X*span.Y*span.Z) * float64(hits) / float64(skewStrata*skewStrata*skewStrata)
}
