// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdrand "math/rand"
	"reflect"
	"testing"

	"oblikovati.org/math"
)

// bruteImprintPairs is the retired O(Fa·Fb) scan, kept here as the oracle: the culled
// imprintCandidates must reproduce its impA/impB/prov element for element (#1607).
func bruteImprintPairs(fa, fb []planarFace) (impA, impB [][][2]math.Point3, prov []imprintSeg) {
	impA = make([][][2]math.Point3, len(fa))
	impB = make([][][2]math.Point3, len(fb))
	for i := range fa {
		for j := range fb {
			onA, onB := pairImprints(fa[i], fb[j])
			impA[i] = append(impA[i], onA...)
			impB[j] = append(impB[j], onB...)
			prov = appendTagged(prov, onA, fa[i], fb[j])
			prov = appendTagged(prov, onB, fb[j], fa[i])
		}
	}
	return impA, impB, prov
}

// randomPairFixture builds a base box and a randomly placed/sized tool box: overlapping,
// tangent (snapped-to-face), and disjoint configurations all occur across seeds.
func randomPairFixture(rng *stdrand.Rand) (fa, fb []planarFace) {
	fa = provFaces("base", 0, 0, 0, 8, 8, 4)
	px := rng.Float64()*16 - 4
	py := rng.Float64()*16 - 4
	pz := rng.Float64()*10 - 3
	if rng.Intn(3) == 0 { // every third tool lands exactly on a base plane: coplanar contact
		pz = 4
	}
	fb = provFaces("tool", px, py, pz, 1+rng.Float64()*6, 1+rng.Float64()*6, 1+rng.Float64()*4)
	return fa, fb
}

// TestCrossingFaceCandidatesSupersetOfImprintingPairs is the culling-soundness gate (#1607):
// on randomized fixtures, every face pair the brute narrow phase produces ANY segment for must
// appear in the AABB candidate set (both orientations). A miss would silently drop an imprint
// and change the boolean's output.
func TestCrossingFaceCandidatesSupersetOfImprintingPairs(t *testing.T) {
	rng := stdrand.New(stdrand.NewSource(1607))
	for trial := 0; trial < 60; trial++ {
		fa, fb := randomPairFixture(rng)
		pairs := crossingFaceCandidates(fa, fb)
		for i := range fa {
			for j := range fb {
				onA, onB := pairImprints(fa[i], fb[j])
				if len(onA) == 0 && len(onB) == 0 {
					continue
				}
				if !containsIndex(pairs.bForA[i], j) || !containsIndex(pairs.aForB[j], i) {
					t.Fatalf("trial %d: imprinting pair (a=%d, b=%d) missing from candidates %v / %v",
						trial, i, j, pairs.bForA[i], pairs.aForB[j])
				}
			}
		}
	}
}

func containsIndex(idx []int, want int) bool {
	for _, v := range idx {
		if v == want {
			return true
		}
	}
	return false
}

// TestImprintCandidatesMatchesBruteScan pins bit-identity, not just superset-ness: the culled
// single-pass imprint+provenance must emit exactly the brute scan's slices — same segments,
// same order (the 2D arrangement is order-sensitive at tolerance) — on randomized fixtures.
func TestImprintCandidatesMatchesBruteScan(t *testing.T) {
	rng := stdrand.New(stdrand.NewSource(1411))
	for trial := 0; trial < 60; trial++ {
		fa, fb := randomPairFixture(rng)
		wantA, wantB, wantProv := bruteImprintPairs(fa, fb)
		gotA, gotB, gotProv := imprintCandidates(fa, fb, crossingFaceCandidates(fa, fb))
		if !reflect.DeepEqual(gotA, wantA) || !reflect.DeepEqual(gotB, wantB) {
			t.Fatalf("trial %d: culled imprint slices differ from the brute scan", trial)
		}
		if !reflect.DeepEqual(gotProv, wantProv) {
			t.Fatalf("trial %d: culled provenance differs from the brute scan", trial)
		}
	}
}

// TestFacesAtPreservesOrder pins the coplanar-cover candidate contract: facesAt keeps the
// ascending-index order the full scan iterated in, so first-match semantics are unchanged.
func TestFacesAtPreservesOrder(t *testing.T) {
	faces := provFaces("base", 0, 0, 0, 2, 2, 2)
	got := facesAt(faces, []int{4, 1, 3})
	if len(got) != 3 {
		t.Fatalf("facesAt returned %d faces, want 3", len(got))
	}
	for k, j := range []int{4, 1, 3} {
		if got[k].lineage.KeyString() != faces[j].lineage.KeyString() {
			t.Fatalf("facesAt[%d] is not source face %d", k, j)
		}
	}
}
