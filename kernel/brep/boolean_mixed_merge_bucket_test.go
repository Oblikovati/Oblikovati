// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The merge scan skips every pair whose faces sit in different mergeBuckets (#3523). That is sound
// only if such a pair could never have answered anything but declineUnshared — the exit that merges
// nothing, records nothing and is never reported. These rows gate exactly that, and nothing wider:
// they say nothing about what the scan does INSIDE one bucket, where every pair is still tested.

// bucketProbeSurfaces is one surface per geom.SurfaceKind. Zero values are enough: the property under
// test is decided by SurfacesCoincide's type switch, which never reads a field when the kinds differ.
// A new geom.SurfaceKind fails the count check below rather than quietly leaving a kind untested.
var bucketProbeSurfaces = []geom.Surface{
	geom.Plane{}, geom.Cylinder{}, geom.Sphere{}, geom.Cone{}, geom.Torus{},
	geom.BSplineSurface{}, geom.EllipticalCylinder{}, geom.EllipticalCone{},
	geom.OffsetSurface{}, geom.ThreadedCylinder{},
}

// TestCrossBucketPairsAreUnshared proves the skip is not a behaviour change: over every ordered pair
// of faces whose bucket keys differ — by surface kind, by keep-sense, or by both — the pair function
// returns declineUnshared, and that exit is not reportable.
func TestCrossBucketPairsAreUnshared(t *testing.T) {
	t.Parallel()
	if len(bucketProbeSurfaces) != len(geom.SurfaceKinds()) {
		t.Fatalf("bucketProbeSurfaces has %d entries, want one per geom.SurfaceKind (%d) — a new kind needs a probe",
			len(bucketProbeSurfaces), len(geom.SurfaceKinds()))
	}
	for _, a := range bucketProbeFaces() {
		for _, b := range bucketProbeFaces() {
			if faceMergeBucket(a) == faceMergeBucket(b) {
				continue
			}
			assertUnshared(t, a, b)
		}
	}
}

// bucketProbeFaces is every probe surface in both keep-senses.
func bucketProbeFaces() []curvedFace {
	faces := make([]curvedFace, 0, 2*len(bucketProbeSurfaces))
	for _, s := range bucketProbeSurfaces {
		faces = append(faces, curvedFace{surface: s}, curvedFace{surface: s, reversed: true})
	}
	return faces
}

// assertUnshared fails when a cross-bucket pair answers anything the scan would have had to act on.
func assertUnshared(t *testing.T, a, b curvedFace) {
	t.Helper()
	rec := &diag.Recorder{}
	_, why := mergePairOnOneSurface(a, b, math.EmptyBox(), math.EmptyBox(), rec)
	if why != declineUnshared {
		t.Errorf("%T(reversed=%v) with %T(reversed=%v) gave %q, want the unshared exit — the scan skips "+
			"this pair, so any other answer is output the skip would lose",
			a.surface, a.reversed, b.surface, b.reversed, why)
	}
	if why.reportable() {
		t.Errorf("%T with %T gives a reportable exit; skipping the pair would drop a diagnostic", a.surface, b.surface)
	}
}

// TestSameKindSameSenseSharesABucket keeps the row above from passing vacuously: the faces the merge
// actually joins — one kind, one sense — must land in ONE bucket, so the scan still offers them.
func TestSameKindSameSenseSharesABucket(t *testing.T) {
	t.Parallel()
	for _, s := range bucketProbeSurfaces {
		a, b := curvedFace{surface: s}, curvedFace{surface: s}
		if faceMergeBucket(a) != faceMergeBucket(b) {
			t.Errorf("two %T faces of one sense are in different buckets; the scan would never pair them", s)
		}
		if faceMergeBucket(a) == faceMergeBucket(curvedFace{surface: s, reversed: true}) {
			t.Errorf("two %T faces of OPPOSITE sense share a bucket; the key does not carry the sense", s)
		}
	}
}

// TestFaceMergeFactsAreOnePerFaceInIndexOrder: the scan indexes the table by face index, so a table
// that lost or reordered an entry would test the wrong pair.
func TestFaceMergeFactsAreOnePerFaceInIndexOrder(t *testing.T) {
	t.Parallel()
	faces := bucketProbeFaces()
	facts := faceMergeFacts(faces)
	if len(facts) != len(faces) {
		t.Fatalf("faceMergeFacts gave %d facts for %d faces, want one each", len(facts), len(faces))
	}
	for i, f := range faces {
		if facts[i].bucket != faceMergeBucket(f) {
			t.Errorf("fact %d carries bucket %+v, want face %d's %+v", i, facts[i].bucket, i, faceMergeBucket(f))
		}
		if facts[i].box != faceLoopBox(f) {
			t.Errorf("fact %d carries box %v, want face %d's %v — the tabulated box is the resolution "+
				"onOneSurface reads, so a different value is a different answer", i, facts[i].box, i, faceLoopBox(f))
		}
	}
}

// TestTabulatedBoxMatchesTheWallsOwnBox is the same equality on a face that HAS loops — the probe
// faces above are loopless, so on their own they would let a box table that ignored the loops pass.
func TestTabulatedBoxMatchesTheWallsOwnBox(t *testing.T) {
	t.Parallel()
	wall := wallFaceOf(t, math.P3(0, 0, 0), 2, 4)
	facts := faceMergeFacts([]curvedFace{wall})
	if facts[0].box != faceLoopBox(wall) {
		t.Fatalf("tabulated box %v, want the wall's own %v", facts[0].box, faceLoopBox(wall))
	}
	if facts[0].box == math.EmptyBox() {
		t.Fatal("the wall's loop box is empty; the equality above is vacuous")
	}
}
