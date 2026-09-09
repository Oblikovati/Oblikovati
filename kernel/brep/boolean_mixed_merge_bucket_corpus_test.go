// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	"sort"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The skip's gate, driven by the bodies the corpus actually builds (#3523 review, finding I-2).
//
// TestCrossBucketPairsAreUnshared states the same property over one synthetic face per
// geom.SurfaceKind with an EMPTY loop box, so it holds the property at one resolution on faces that
// carry no geometry. The property the skip needs is wider than that: a cross-bucket pair answers
// declineUnshared for ANY field values at ANY resolution. That is true because SurfacesCoincide's
// every arm asserts b to a's concrete type and defaults to false — but reading a type switch is an
// argument, and an argument is not a gate.
//
// This row is the gate. It harvests every face of the five cocylindrical-merge corpus bodies, plus
// one built cone, sphere and torus for the three kinds that corpus never produces — real surfaces at
// real coordinates, real loops, each at its OWN loop-box resolution — and drives the SAME pair
// function the scan uses over every cross-bucket pair of them. Nothing here is a bare curvedFace
// literal, and there is no second engine and no production hook: the test replays the scan's own
// faceMergeFacts and its own bucket compare rather than restating either.
//
// What it does NOT claim: these are not literally the pairs one scan skipped, because the scan's
// face list changes as it merges. It is the same universally-quantified property over a real and
// much larger population.

// TestNoCrossBucketPairOfACorpusFaceIsAnythingButUnshared drives the property over the corpus.
func TestNoCrossBucketPairOfACorpusFaceIsAnythingButUnshared(t *testing.T) {
	t.Parallel()
	faces := corpusMergeFaces(t)
	rec := &diag.Recorder{}
	skipped := auditCrossBucketPairs(t, faces, rec)
	assertCorpusSkipIsMeaningful(t, len(faces), skipped)
	if got := len(rec.Records()); got != 0 {
		t.Errorf("the skipped pairs recorded %d diagnostic(s); declineUnshared must record nothing, or "+
			"skipping the pair would drop one", got)
	}
}

// corpusMergeFaces is every face of every body in the cocylindrical-merge corpus, as the merge sees
// them — the same chartCorpus the seam-slit and chart rows walk — plus one primitive per remaining
// surface kind. The corpus alone only builds planes and cylinders, and a cross-bucket row that never
// sees a cone, a sphere or a torus would state the property only where it happens to be easiest.
func corpusMergeFaces(t *testing.T) []curvedFace {
	t.Helper()
	var out []curvedFace
	for _, tc := range chartCorpus(t) {
		out = append(out, facesOfAny(tc.body)...)
	}
	out = append(out, primitiveFacesOfEveryKind(t)...)
	if len(out) == 0 {
		t.Fatal("the corpus produced no faces; this row would gate nothing")
	}
	return out
}

// primitiveFacesOfEveryKind adds real cone, sphere and torus faces — built at off-origin centres and
// odd radii, so their fields are populated and their loop boxes are their own.
func primitiveFacesOfEveryKind(t *testing.T) []curvedFace {
	t.Helper()
	var out []curvedFace
	for _, b := range []*topo.Body{
		buildOrFail(t, "cone", func() (*topo.Body, error) {
			return SolidCylinderCone(math.P3(1.5, -2.25, 0.75), math.P3(1.5, -2.25, 4.125), 2.125, 0.875, "cone")
		}),
		buildOrFail(t, "sphere", func() (*topo.Body, error) {
			return SolidSphere(math.P3(-3.125, 0.5, 2.75), 1.875, "sphere")
		}),
		buildOrFail(t, "torus", func() (*topo.Body, error) {
			return SolidTorus(math.P3(4.25, 1.125, -0.5), math.V3(0, 0, 1), 3.375, 0.625, "torus")
		}),
	} {
		out = append(out, facesOfAny(b)...)
	}
	return out
}

// buildOrFail names which primitive refused, since a silent skip would narrow the row.
func buildOrFail(t *testing.T, name string, build func() (*topo.Body, error)) *topo.Body {
	t.Helper()
	b, err := build()
	if err != nil {
		t.Fatalf("building the %s probe body: %v", name, err)
	}
	return b
}

// auditCrossBucketPairs runs the pair function on every pair the scan's bucket test would skip, and
// returns how many there were. Boxes come from each face's own loops, exactly as the scan's table
// builds them.
func auditCrossBucketPairs(t *testing.T, faces []curvedFace, rec *diag.Recorder) int {
	t.Helper()
	facts := faceMergeFacts(faces)
	skipped := 0
	for i := range faces {
		for j := i + 1; j < len(faces); j++ {
			if facts[i].bucket == facts[j].bucket {
				continue
			}
			skipped++
			assertPairIsUnshared(t, faces[i], faces[j], facts[i], facts[j], rec)
		}
	}
	return skipped
}

// assertPairIsUnshared is the statement itself: the scan skips this pair, so any answer but the
// silent one is output the skip would lose.
func assertPairIsUnshared(t *testing.T, a, b curvedFace, fa, fb faceMergeFact, rec *diag.Recorder) {
	t.Helper()
	_, why := mergePairOnOneSurface(a, b, fa.box, fb.box, rec)
	if why != declineUnshared {
		t.Errorf("a %T face (reversed=%v) and a %T face (reversed=%v) are in different buckets, so the "+
			"scan never offers them — but the pair function answers %q, not the silent unshared exit",
			a.surface, a.reversed, b.surface, b.reversed, why)
	}
}

// assertCorpusSkipIsMeaningful stops the row passing vacuously: the corpus must actually contain
// faces the bucket separates, and it must contain more than one bucket's worth.
func assertCorpusSkipIsMeaningful(t *testing.T, faceCount, skipped int) {
	t.Helper()
	if faceCount < 2 {
		t.Fatalf("the corpus gave %d face(s); there are no pairs to skip", faceCount)
	}
	if skipped == 0 {
		t.Fatal("no corpus pair spans two buckets, so this row asserts nothing — the corpus must carry " +
			"faces of at least two surface kinds or two senses")
	}
	t.Logf("audited %d cross-bucket pairs over %d corpus faces", skipped, faceCount)
}

// TestTheCorpusFacesSpanSeveralBuckets names what the row above actually covered, so a corpus that
// quietly narrowed to one surface kind fails here instead of passing there.
func TestTheCorpusFacesSpanSeveralBuckets(t *testing.T) {
	t.Parallel()
	faces := corpusMergeFaces(t)
	buckets := map[mergeBucket]int{}
	for _, f := range faces {
		buckets[faceMergeBucket(f)]++
	}
	if len(buckets) < 2 {
		t.Fatalf("the corpus faces occupy %d bucket(s); the cross-bucket row cannot be meaningful", len(buckets))
	}
	for _, line := range sortedBucketCensus(buckets) {
		t.Log(line)
	}
}

// sortedBucketCensus renders the histogram in a fixed order. A map walk would print it differently
// on every run, and a log two runs cannot be diffed against each other is worth less than one.
func sortedBucketCensus(buckets map[mergeBucket]int) []string {
	lines := make([]string, 0, len(buckets))
	for b, n := range buckets {
		lines = append(lines, fmt.Sprintf("bucket %+v: %d faces", b, n))
	}
	sort.Strings(lines)
	return lines
}
