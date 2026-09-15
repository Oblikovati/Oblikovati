// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The torus against a quadric that is NOT invariant about its axis (ADR-0061 stage 5, third slice).
//
// boolean_torus_section_test.go covers the family whose azimuth dependence collapses to ONE harmonic —
// a sphere anywhere, a cylinder or cone parallel to the ring's axis. Everything else keeps the second
// harmonic, so a tube circle meets the tool at up to FOUR azimuths rather than two and the branch
// pairing is a real question. These rows drive the three shapes that answer takes:
//
//   - a rod ACROSS the ring, whose quadric is an infinite cylinder and so pierces the tube on BOTH
//     flanks: two lanes per station, four section loops, of which only the ones inside the rod's own
//     finite trim survive into the body;
//   - the same rod FATTER than the tube it crosses, which swallows the tube's flank whole. Its branch
//     pair never folds, so the section is four independent FULL-PERIOD branches rather than folded
//     loops — the topology the reduction used to decline by name (Oblikovati/Oblikovati#3515);
//   - a TILTED drill, one lane, entering the tube's top and leaving its bottom;
//   - an off-centre COUNTERSINK, whose axis is parallel to the ring's. That one is axis-invariant —
//     being off-centre moves the quadric's linear term, never its tensor — so it is the reproduction
//     row: it must still take the one-harmonic path and come out exactly as it did before.
//
// Every row is certified twice over. Requicha's identity V(∪) + V(∩) = V(A) + V(B) and
// V(−) + V(∩) = V(A) ties the three operations to each other and to the operands' own analytic
// volumes — three bodies built by three separate trims of the same section have no reason to satisfy
// it unless the section is right. And each volume is checked against a Monte Carlo integral of the two
// primitives' ANALYTIC membership, which is independent of everything the kernel did. A body that
// measures right could still be the wrong shape, so the face census is asserted beside both.

// skewRingTool is one tool driven against the corpus ring: the body, its own analytic volume, the
// membership predicate the independent oracle integrates, and the face census each operation leaves.
type skewRingTool struct {
	name   string
	body   *topo.Body
	volume float64
	inside func(x, y, z float64) bool
	census map[ops.PartFeatureOperation]skewFaceCensus
}

// skewFaceCensus is a result's faces tallied by analytic surface kind.
type skewFaceCensus struct{ tori, cylinders, cones, planes, faces int }

// skewCorpusRing is the fixture ring shared with boolean_torus_section_test.go: major radius 5, minor
// 1.5, about z.
func skewCorpusRing(t *testing.T) *topo.Body {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5, "ring")
	if err != nil {
		t.Fatalf("ring: %v", err)
	}
	return ring
}

// skewCorpusRod is the rod driven ACROSS the ring: radius 1 along +x, from the ring's own centre out
// past its far side, so it pierces the tube once and its two caps stand clear of the ring's surface.
func skewCorpusRod(t *testing.T) skewRingTool {
	t.Helper()
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1, 9)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	return skewRingTool{
		name:   "rod across the ring",
		body:   rod,
		volume: stdmath.Pi * 9,
		inside: func(x, y, z float64) bool { return y*y+z*z <= 1 && x >= 0 && x <= 9 },
		census: map[ops.PartFeatureOperation]skewFaceCensus{
			// The ring's surface with the bore's two seams, the rod's wall in the two pieces the ring
			// leaves of it (inside the hole, and beyond the far flank), and the rod's two caps.
			ops.Join: {tori: 1, cylinders: 2, planes: 2, faces: 5},
			// The ring's surface holed twice, and the tunnel wall bounded by both seams.
			ops.Cut: {tori: 1, cylinders: 1, faces: 2},
			// The plug: the rod's wall between the seams, capped by the two torus patches it cut out.
			ops.Intersect: {tori: 2, cylinders: 1, faces: 3},
		},
	}
}

// skewCorpusFatRod is the same rod at a radius that EXCEEDS the tube's own. Its axis runs down the
// tube's centre circle at (5, 0, 0), so a radius over 1.5 contains that whole tube circle: the rod
// swallows the ring's flank instead of piercing it, and severs the ring into a C.
//
// That is what makes the section wrap. The tube circle meets the rod's wall at four azimuths at EVERY
// tube angle and no two of them ever merge, so there is no fold to bound a window with and the section
// is four full-period branches. Nothing in the reduction is told there are four: the station's quartic
// certifies its own real roots and the extremum tracks label them (Oblikovati/Oblikovati#3515).
//
// It is an ordinary part — a pin through a ring's cross-section — and it used to be a named decline.
func skewCorpusFatRod(t *testing.T) skewRingTool {
	t.Helper()
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 2, 9)
	if err != nil {
		t.Fatalf("fat rod: %v", err)
	}
	return skewRingTool{
		name:   "rod fatter than the tube",
		body:   rod,
		volume: stdmath.Pi * 4 * 9,
		inside: func(x, y, z float64) bool { return y*y+z*z <= 4 && x >= 0 && x <= 9 },
		census: map[ops.PartFeatureOperation]skewFaceCensus{
			// The ring's surface with the swallowed collar cut out of it — one face bounded by the two
			// full-period branches — plus the rod's wall holed where the ring's two cut ends stand in it,
			// and the rod's two caps, both clear of the ring.
			ops.Join: {tori: 1, cylinders: 1, planes: 2, faces: 4},
			// The severed C: the ring's surface, and the rod's wall in the two patches that close the cut.
			ops.Cut: {tori: 1, cylinders: 2, faces: 3},
			// The plug: the swallowed collar of the ring's surface, closed by the same two wall patches.
			ops.Intersect: {tori: 1, cylinders: 2, faces: 3},
		},
	}
}

// skewCorpusTiltedDrill is a drill through the tube at (5, 0, 0) whose axis leans out of the ring's
// own by atan(0.3) — the commonest non-invariant tool in a real part, and the one whose station
// carries a single lane rather than two.
func skewCorpusTiltedDrill(t *testing.T) skewRingTool {
	t.Helper()
	axis, err := math.UnitVector3FromVector(math.V3(0.3, 0, 1))
	if err != nil {
		t.Fatalf("drill axis: %v", err)
	}
	// Long enough that both caps stand clear of the ring's own reach along this axis (5 either side of
	// the tube's centre leaves them at |z| ≈ 4.8, against the ring's 1.5), so the only contact in the
	// row is the one being tested: the drill's wall against the ring's surface.
	const length = 10.0
	dir := axis.AsVector()
	base := math.P3(5, 0, 0).TranslateBy(dir.Scale(-length / 2))
	drill, err := brep.SolidCylinder(base, dir, 0.8, length)
	if err != nil {
		t.Fatalf("tilted drill: %v", err)
	}
	return skewRingTool{
		name:   "tilted drill",
		body:   drill,
		volume: stdmath.Pi * 0.8 * 0.8 * length,
		inside: insideFiniteCylinder(base, dir, 0.8, length),
		census: map[ops.PartFeatureOperation]skewFaceCensus{
			ops.Join:      {tori: 1, cylinders: 2, planes: 2, faces: 5},
			ops.Cut:       {tori: 1, cylinders: 1, faces: 2},
			ops.Intersect: {tori: 2, cylinders: 1, faces: 3},
		},
	}
}

// skewCorpusCountersink is a conical seat sunk into the ring's tube from above, on an axis PARALLEL to
// the ring's but 5 out from it. Its tensor is invariant about the ring's axis however far off-centre it
// sits, so it must take the one-harmonic path unchanged — the reproduction row.
func skewCorpusCountersink(t *testing.T) skewRingTool {
	t.Helper()
	sink, err := brep.SolidCylinderCone(math.P3(5, 0, -1), math.P3(5, 0, 4), 0.4, 2.2, "sink")
	if err != nil {
		t.Fatalf("countersink: %v", err)
	}
	const h, r0, r1 = 5.0, 0.4, 2.2
	return skewRingTool{
		name:   "off-centre countersink",
		body:   sink,
		volume: stdmath.Pi * h / 3 * (r0*r0 + r0*r1 + r1*r1),
		inside: func(x, y, z float64) bool {
			return z >= -1 && z <= 4 && stdmath.Hypot(x-5, y) <= r0+(z+1)*(r1-r0)/h
		},
		census: map[ops.PartFeatureOperation]skewFaceCensus{
			// The sink's lower cap sits inside the ring's material, so the union keeps only its top.
			ops.Join:      {tori: 1, cones: 1, planes: 1, faces: 3},
			ops.Cut:       {tori: 1, cones: 1, planes: 1, faces: 3},
			ops.Intersect: {tori: 1, cones: 1, planes: 1, faces: 3},
		},
	}
}

// insideFiniteCylinder is the analytic membership of a capped cylinder — the oracle's own geometry,
// written from the constructor's arguments rather than read back off the body.
func insideFiniteCylinder(base math.Point3, dir math.Vector3, radius, height float64) func(x, y, z float64) bool {
	return func(x, y, z float64) bool {
		w := base.VectorTo(math.P3(math.Scalar(x), math.Scalar(y), math.Scalar(z)))
		along := float64(w.Dot(dir))
		if along < 0 || along > height {
			return false
		}
		return float64(w.Sub(dir.Scale(math.Scalar(along))).LengthSquared()) <= radius*radius
	}
}

// TestASkewToolThroughARingIsExact drives all three operations over each tool and certifies every
// result against Requicha's identity, against the independent membership integral, and against the
// face census the contact implies.
func TestASkewToolThroughARingIsExact(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~15s): `make test-corpus`")
	}
	t.Parallel()
	ring := skewCorpusRing(t)
	ringVol := 2 * stdmath.Pi * stdmath.Pi * 5 * 1.5 * 1.5
	inRing := func(x, y, z float64) bool { d := stdmath.Hypot(x, y) - 5; return d*d+z*z <= 1.5*1.5 }
	tools := []skewRingTool{skewCorpusRod(t), skewCorpusFatRod(t), skewCorpusTiltedDrill(t), skewCorpusCountersink(t)}
	for _, tool := range tools {
		t.Run(tool.name, func(t *testing.T) {
			t.Parallel()
			join := skewOpVolumes(t, ops.Join, ring, tool)
			cut := skewOpVolumes(t, ops.Cut, ring, tool)
			lens := skewOpVolumes(t, ops.Intersect, ring, tool)
			assertRequichaAtEveryFaceting(t, ringVol, tool.volume, join, cut, lens)
			whole := ring.RangeBox().Union(tool.body.RangeBox())
			assertRelative(t, "V(∪) against the membership integral",
				join[0], skewMembershipVolume(ops.Join, inRing, tool.inside, whole), membershipTol)
			assertRelative(t, "V(−) against the membership integral",
				cut[0], skewMembershipVolume(ops.Cut, inRing, tool.inside, whole), membershipTol)
			assertRelative(t, "V(∩) against the membership integral",
				lens[0], skewMembershipVolume(ops.Intersect, inRing, tool.inside, overlapBox(ring.RangeBox(), tool.body.RangeBox())),
				membershipTol)
		})
	}
}

// skewFacetings are the tessellation qualities every row's volumes are read at. Requicha's identity
// ties three ANALYTIC bodies to each other, so a residual that MOVES with the faceting is a tessellation
// artefact wearing the identity's clothes rather than a certificate that the section was right. Reading
// it at two qualities is what tells the two apart.
var skewFacetings = [...]struct {
	name    string
	quality ops.Quality
}{
	{"default faceting", ops.DefaultQuality()},
	{"property faceting", ops.PropertyQuality()},
}

// assertRequichaAtEveryFaceting ties the three operations to each other and to the operands' own
// analytic volumes, at each faceting the volumes were read at.
func assertRequichaAtEveryFaceting(t *testing.T, ringVol, toolVol float64, join, cut, lens [len(skewFacetings)]float64) {
	t.Helper()
	for i, f := range skewFacetings {
		assertRelative(t, f.name+": V(−) + V(∩)", cut[i]+lens[i], ringVol, requichaTol)
		assertRelative(t, f.name+": V(∪) + V(∩)", join[i]+lens[i], ringVol+toolVol, requichaTol)
	}
}

// requichaTol is how far the three operations may drift from the identity that ties them. It is the
// kernel's own arithmetic, not an oracle's sampling error, so it is tight.
const requichaTol = 1e-5

// membershipTol is the MONTE CARLO's error, not the kernel's: at skewMembershipSamples the integral's
// own relative standard error is under a part in a thousand on every row here, and this allows three
// of those. The kernel's answer is exact.
const membershipTol = 2e-3

// skewOpVolumes runs one boolean, asserts it is a valid closed manifold solid with the face census the
// contact implies and an exact boundary, and returns its volume read at each of [skewFacetings].
func skewOpVolumes(t *testing.T, op ops.PartFeatureOperation, ring *topo.Body, tool skewRingTool) [len(skewFacetings)]float64 {
	t.Helper()
	res, err := ops.Boolean(op, ring, tool.body)
	if err != nil {
		t.Fatalf("%s %v: %v", tool.name, op, err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("%s %v: not a valid closed manifold solid: %+v", tool.name, op, r)
	}
	if tol := res.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("%s %v reports AchievedBoundaryTolerance %g, want 0 — the section is a closed form", tool.name, op, tol)
	}
	if got, want := skewCensusOf(res), tool.census[op]; got != want {
		t.Errorf("%s %v: census %+v, want %+v", tool.name, op, got, want)
	}
	var out [len(skewFacetings)]float64
	for i, f := range skewFacetings {
		out[i] = query.BodyGeometryProperties(res, f.quality).Volume
	}
	return out
}

// skewCensusOf tallies a body's faces by analytic surface kind.
func skewCensusOf(b *topo.Body) skewFaceCensus {
	c := skewFaceCensus{faces: len(b.Faces())}
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus:
			c.tori++
		case geom.Cylinder:
			c.cylinders++
		case geom.Cone:
			c.cones++
		case geom.Plane:
			c.planes++
		}
	}
	return c
}

// assertRelative fails when got and want differ by more than tol relative to want.
func assertRelative(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / want; rel > tol {
		t.Errorf("%s = %.6f, want %.6f — rel %.4g > %.4g", what, got, want, rel, tol)
	}
}

// overlapBox is the box both boxes cover, which bounds their solids' intersection — so the lens is
// integrated over a box derived from the geometry rather than one chosen by hand.
func overlapBox(a, b math.Box) math.Box {
	lo := math.P3(max(a.Min.X, b.Min.X), max(a.Min.Y, b.Min.Y), max(a.Min.Z, b.Min.Z))
	hi := math.P3(min(a.Max.X, b.Max.X), min(a.Max.Y, b.Max.Y), min(a.Max.Z, b.Max.Z))
	return math.NewBox(lo, hi)
}

// skewMembershipVolume integrates "inside the ring op inside the tool" over the box, from the two
// primitives' own analytic membership — deliberately not the kernel's classifier, so the oracle is
// independent of what it gates. The seed is fixed, so the number is the same on every run and platform.
//
// The samples are STRATIFIED: one jittered sample per cell of a skewStrata³ lattice, rather than
// skewStrata³ independent points. Only the cells the boundary crosses carry any variance at all, which
// takes the oracle's own error from a part in a thousand to a few parts in a hundred thousand on these
// rows — and it removes the sampler's 3D lattice structure, which plain triples from one stream showed
// as a seed-dependent bias of ~1e-3 that did NOT shrink with the sample count (measured: the
// countersink lens read 5.5305, 5.5535 and 5.5536 from three seeds against an exact 5.5502832).
func skewMembershipVolume(op ops.PartFeatureOperation, inRing, inTool func(x, y, z float64) bool, box math.Box) float64 {
	rng := rand.New(rand.NewSource(11))
	span := box.Max.AsVector().Sub(box.Min.AsVector())
	hits := 0
	for i := range skewStrata {
		for j := range skewStrata {
			for k := range skewStrata {
				x := float64(box.Min.X) + jitteredStratum(rng, i)*float64(span.X)
				y := float64(box.Min.Y) + jitteredStratum(rng, j)*float64(span.Y)
				z := float64(box.Min.Z) + jitteredStratum(rng, k)*float64(span.Z)
				if keepsUnderOp(op, inRing(x, y, z), inTool(x, y, z)) {
					hits++
				}
			}
		}
	}
	return float64(span.X*span.Y*span.Z) * float64(hits) / float64(skewStrata*skewStrata*skewStrata)
}

// jitteredStratum is a uniform sample of the i-th of skewStrata equal slices of [0, 1).
func jitteredStratum(rng *rand.Rand, i int) float64 {
	return (float64(i) + rng.Float64()) / skewStrata
}

// skewStrata is the lattice's side, so the integral takes skewStrata³ ≈ 4.1 million samples. Measured
// against the countersink lens's exact 5.5502832 (a one-dimensional quadrature of the disc∩annulus area
// per height), a lattice this fine lands within 2e-5 relative — a hundredth of membershipTol.
const skewStrata = 160
