// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
)

// TestJacobiEigenSym3 diagonalises a non-diagonal symmetric matrix (exercising the Jacobi
// rotations): [[2,1,0],[1,2,0],[0,0,5]] has eigenvalues 1, 3, 5. The 2×2 block (2,1;1,2) has
// eigenvectors (1,∓1)/√2.
func TestJacobiEigenSym3(t *testing.T) {
	vals, vecs := jacobiEigenSym3(sym3{xx: 2, yy: 2, zz: 5, xy: 1, yz: 0, zx: 0})
	want := [3]float64{1, 3, 5} // ascending
	for i := 0; i < 3; i++ {
		if math.Abs(vals[i]-want[i]) > 1e-9 {
			t.Fatalf("eigenvalue[%d] = %g, want %g (all: %v)", i, vals[i], want[i], vals)
		}
		if math.Abs(vecLen(vecs[i])-1) > 1e-9 {
			t.Errorf("eigenvector[%d] = %v, want unit length", i, vecs[i])
		}
	}
	// The smallest eigenvalue (1) pairs with (1,-1,0)/√2; check |components| up to sign.
	const inv2 = 0.70710678
	if math.Abs(math.Abs(vecs[0][0])-inv2) > 1e-6 || math.Abs(math.Abs(vecs[0][1])-inv2) > 1e-6 {
		t.Errorf("eigenvector for λ=1 = %v, want (±1,∓1,0)/√2", vecs[0])
	}
}

// TestQualityForAccuracyLevels: every accuracy level computes the box's (tessellation-exact)
// volume, exercising the low/medium/high quality mapping.
func TestQualityForAccuracyLevels(t *testing.T) {
	for _, acc := range []types.MassPropertiesAccuracy{types.MassPropertiesLow, types.MassPropertiesMedium, types.MassPropertiesHigh} {
		mp := MassPropertiesOf([]*topo.Body{block53x2(t)}, 1, acc)
		approx(t, "volume@"+acc.String(), mp.VolumeMm3, 30000)
	}
}

func vecLen(v [3]float64) float64 { return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]) }
