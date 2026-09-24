// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The FACETING-MONOTONICITY row (Oblikovati/Oblikovati#3547, and #3515 review round 4).
//
// These four pairs were review round 3's NEW-6: bodies whose volume broke Requicha's identity by 1 % to
// 130 %, valid, closed, manifold, with nothing recorded. Two reviews and one implementer each diagnosed
// them in a different wrong place — the boolean's classification, then the section — and all three were
// wrong. The bodies were right the whole time. What was wrong was the MESH the volume was read off at
// `PropertyQuality`, and #3518's chart mesher fixed it on this wave's base.
//
// The criterion that separates the two was already written in boolean_torus_skew_test.go, where
// skewFacetings says it: *a residual that MOVES with the faceting is a tessellation artefact wearing the
// identity's clothes rather than a certificate that the section was right*. Nobody read the second
// faceting until round 4's review. This row is that criterion made into a test, so the next instance is
// caught by a run instead of by three rounds of argument.
//
// What it asserts is an ORDERING rather than a number, which is what makes it a regression row rather
// than a golden: **a FINER faceting may not read the identity worse than a coarser one.** Measured on
// this branch, the four rows read
//
//	row      DefaultQuality   PropertyQuality
//	  2         3.66e-02          2.42e-04
//	621         2.13e-02          2.13e-04
//	680         4.01e-02          2.09e-04
//	816         1.55e-02          2.04e-04
//
// — property an order or two better, everywhere. At review 3's commit `f46c0849` the same rows read
// property 1.32, 1.21e-01, 1.16e-02 and 1.30 against the same DefaultQuality figures, so the finer read
// was 3× to 36× WORSE and this row fails. A defect of that shape cannot hide behind a tolerance, because
// the row does not carry one for the comparison; it only carries a cap on the finer reading.
//
// The generator is review round 3's, recovered in round 4's review from its own pinned table (#3547's
// first acceptance criterion):
//
//	major  := 2 + 6*rng.Float64()                    // rand.NewSource(20260909), 4000 rows
//	minor  := 0.2 + 0.45*major*rng.Float64()         // ring always about +z at the origin
//	px, py, pz := -4 + 8*rng.Float64() each
//	dir    := V3(NormFloat64, NormFloat64, NormFloat64)
//	radius := 0.1 + 4*rng.Float64()                  // rod trimmed to length 24 about its own point
func TestAFinerFacetingNeverReadsTheIdentityWorse(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier: `make test-corpus`")
	}
	t.Parallel()
	for _, row := range massPropsCorpus() {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			ring, rod := massPropsPair(t, row)
			rec := &diag.Recorder{}
			cut, err := ops.BooleanWithDiagnostics(ops.Cut, ring, rod, rec)
			if err != nil {
				t.Fatalf("cut: %v", err)
			}
			lens, err := ops.BooleanWithDiagnostics(ops.Intersect, ring, rod, rec)
			if err != nil {
				t.Fatalf("intersect: %v", err)
			}
			assertBodyIsASolid(t, "cut", cut)
			assertBodyIsASolid(t, "intersect", lens)
			coarse := requichaResidual(cut, lens, row.ringVolume(), ops.DefaultQuality())
			fine := requichaResidual(cut, lens, row.ringVolume(), ops.PropertyQuality())
			t.Logf("Requicha residual: default %.4e, property %.4e", coarse, fine)
			if fine > coarse {
				t.Errorf("the FINER faceting reads the identity worse: property %.4e against default %.4e; "+
					"a residual that grows with the faceting is a mesh defect, not a body", fine, coarse)
			}
			if fine > massPropsResidualCap {
				t.Errorf("property-faceting Requicha residual %.4e is over the %.0e cap", fine, massPropsResidualCap)
			}
			assertTheResidualIsDeclared(t, rec)
		})
	}
}

// massPropsResidualCap bounds the FINER reading on its own, with an order of margin over the 2.42e-04
// these rows measure. It is deliberately loose: the row's subject is the ordering above, and a cap tight
// enough to be interesting here would be a golden on a tessellation, which is the thing this row exists
// to stop anyone trusting.
const massPropsResidualCap = 2e-3 // tol:approximation — relative Requicha residual of a faceted read

// massPropsRow is one recovered corpus pair: the ring's radii, and the rod's axis point, unit direction
// and radius.
type massPropsRow struct {
	name         string
	major, minor float64
	px, py, pz   float64
	ax, ay, az   float64
	radius       float64
}

// ringVolume is the ring's exact analytic volume, 2π²Rr² — the right-hand side of the identity, and an
// oracle the kernel has no part in.
func (r massPropsRow) ringVolume() float64 {
	return 2 * stdmath.Pi * stdmath.Pi * r.major * r.minor * r.minor
}

// massPropsCorpus is rows 2, 621, 680 and 816 of the recovered generator — review round 3's four worked
// examples, the ones whose figures it published.
func massPropsCorpus() []massPropsRow {
	return []massPropsRow{
		{"row 2", 3.5517294550162517, 0.27389614338774859,
			-2.2128909550214262, -0.21793146070323788, -2.7620628051347742,
			0.28495417298057557, 0.92693213193036428, -0.24412689752664737, 3.4962132444635894},
		{"row 621", 4.3284662985685269, 0.85790826505966944,
			-1.4859580019296779, 3.3638731338737831, 3.1734943778469207,
			0.93924116484177067, 0.30706155304007371, 0.15342502048652773, 3.0953676493942104},
		{"row 680", 4.3640948275364746, 0.79431470434010087,
			-3.7950462971871848, 2.7221378109594196, 2.7741019871803738,
			0.74884613733386129, 0.65754275564848252, -0.082867286032280926, 2.4175022209570405},
		{"row 816", 3.5094315011325756, 1.5217266277740265,
			1.0961993119255764, -1.9773266243501477, -1.3355055298699248,
			0.0153116512480557, -0.74269059232006407, 0.669459660782732, 3.7227638108215984},
	}
}

// massPropsPair builds the row's ring and its rod, the rod trimmed to length 24 centred on its own point
// so both caps stand clear of the ring.
func massPropsPair(t *testing.T, r massPropsRow) (ring, rod *topo.Body) {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), r.major, r.minor, "ring")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	axis := math.V3(math.Scalar(r.ax), math.Scalar(r.ay), math.Scalar(r.az))
	base := math.P3(math.Scalar(r.px), math.Scalar(r.py), math.Scalar(r.pz))
	rod, err = brep.SolidCylinder(base.TranslateBy(axis.Scale(-massPropsRodHalfLength)), axis, r.radius,
		2*massPropsRodHalfLength)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return ring, rod
}

// massPropsRodHalfLength is half the rod's 24-unit length, which is review round 3's own trim.
const massPropsRodHalfLength = 12.0

// requichaResidual is |V(cut) + V(∩) − V(ring)| / V(ring), read at one faceting.
func requichaResidual(cut, lens *topo.Body, ringVolume float64, quality ops.Quality) float64 {
	moved := query.BodyGeometryProperties(cut, quality).Volume + query.BodyGeometryProperties(lens, quality).Volume
	return stdmath.Abs(moved-ringVolume) / ringVolume
}

// assertBodyIsASolid is the part of the row that says the BODIES were never the problem.
func assertBodyIsASolid(t *testing.T, what string, b *topo.Body) {
	t.Helper()
	if r := ops.Validate(b); !r.Valid || !r.Closed || !r.Manifold || !b.IsSolid() {
		t.Fatalf("%s is not a valid closed manifold solid: %+v", what, r)
	}
}

// assertTheResidualIsDeclared requires the faceted residual to be DECLARED rather than implied: the
// volume gate #3516 added reports when it could not bracket a moved volume, and a row whose identity
// only holds to 2e-4 is exactly a row a user should be told the kernel did not verify.
func assertTheResidualIsDeclared(t *testing.T, rec *diag.Recorder) {
	t.Helper()
	for _, d := range rec.Records() {
		if d.Code == ops.CodeBooleanVolumeNotBracketed {
			return
		}
	}
	t.Errorf("no %s diagnostic: the residual is undeclared, which is the half of #3547 that is about "+
		"never degrading silently; the recorder holds %v", ops.CodeBooleanVolumeNotBracketed, rec.Records())
}
