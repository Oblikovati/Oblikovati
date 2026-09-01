// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdrand "math/rand"
	"reflect"
	"testing"

	dset "oblikovati.org/kernel/ops/internal/disjoint"
	"oblikovati.org/math"
)

// bruteEndpointClusterSnap is the retired O(m²) clustering pre-pass, kept as the oracle: the
// grid-hash version must reproduce its output bit for bit (#1607).
func bruteEndpointClusterSnap(pts []math.Point3, tol float64) *boundaryClusters {
	cluster := make([]int, len(pts))
	for i := range cluster {
		cluster[i] = i
	}
	for i := range pts {
		for j := i + 1; j < len(pts); j++ {
			if float64(pts[i].DistanceTo(pts[j])) <= tol {
				dset.Union(cluster, i, j)
			}
		}
	}
	return clusterCentroids(pts, cluster)
}

// sewClusterFixture builds a reproducible endpoint soup: cluster nuclei with satellites
// inside tol (must meld), strays far away (must stay singleton), and a chain of points spaced
// just under tol (must meld transitively across grid cells).
func sewClusterFixture(rng *stdrand.Rand, nuclei int, tol float64) []math.Point3 {
	var pts []math.Point3
	for range nuclei {
		c := math.P3(rng.Float64()*50, rng.Float64()*50, rng.Float64()*50)
		pts = append(pts, c)
		for range 3 {
			pts = append(pts, c.TranslateBy(math.V3(rng.Float64()*tol/3, rng.Float64()*tol/3, rng.Float64()*tol/3)))
		}
		pts = append(pts, c.TranslateBy(math.V3(10*tol, 0, 0))) // stray
	}
	for s := range 8 { // the transitive chain
		pts = append(pts, math.P3(60+float64(s)*0.9*tol, 60, 60))
	}
	return pts
}

// TestEndpointClusterSnapMatchesBrute pins bit-identity of the grid-hash clustering against
// the retired O(m²) oracle on randomized fixtures: same components, same centroids, same
// cell keys.
func TestEndpointClusterSnapMatchesBrute(t *testing.T) {
	t.Parallel()
	rng := stdrand.New(stdrand.NewSource(84)) // PBI-084, the original sew work item
	for trial := range 20 {
		tol := 0.05 + rng.Float64()*0.2
		pts := sewClusterFixture(rng, 20, tol)
		got := endpointClusterSnap(pts, tol)
		want := bruteEndpointClusterSnap(pts, tol)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("trial %d (tol=%g): grid clustering differs from the brute oracle", trial, tol)
		}
	}
}

// TestEndpointClusterSnapChainMelds pins the transitive case the grid must not break: a chain
// spaced under tol melds into ONE cluster even though its ends are many cells apart, so every
// chain member snaps to the same centroid.
func TestEndpointClusterSnapChainMelds(t *testing.T) {
	t.Parallel()
	tol := 0.1
	var pts []math.Point3
	for s := range 10 {
		pts = append(pts, math.P3(float64(s)*0.9*tol, 0, 0))
	}
	bc := endpointClusterSnap(pts, tol)
	first := bc.apply(pts[0])
	for s, p := range pts {
		if bc.apply(p) != first {
			t.Fatalf("chain member %d snapped to %v, want the shared centroid %v", s, bc.apply(p), first)
		}
	}
}
