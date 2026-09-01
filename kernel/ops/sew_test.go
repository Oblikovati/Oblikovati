// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// gappedCubeQuilt is the open unit-cube quilt whose lid floats `gap` above the
// walls — the canonical "imported shell with a real gap" PBI-084 targets. The
// five lower faces stitch exactly; the lid's rim sits at z = 1+gap.
func gappedCubeQuilt(t *testing.T, gap float64) *topo.Body {
	t.Helper()
	p := math.P3
	z := math.Scalar(1 + gap)
	faces := cubeFaces()[:1] // bottom
	faces = append(faces,
		quadBody("front", p(0, 0, 0), p(1, 0, 0), p(1, 0, 1), p(0, 0, 1)),
		quadBody("back", p(0, 1, 0), p(0, 1, 1), p(1, 1, 1), p(1, 1, 0)),
		quadBody("left", p(0, 0, 0), p(0, 0, 1), p(0, 1, 1), p(0, 1, 0)),
		quadBody("right", p(1, 0, 0), p(1, 1, 0), p(1, 1, 1), p(1, 0, 1)),
		quadBody("lid", p(0, 0, z), p(1, 0, z), p(1, 1, z), p(0, 1, z)),
	)
	quilt, err := Stitch(faces, 0, true, "import")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	return quilt
}

// TestSewClosesGappedLid: a lid floating 5e-5 above the walls (within the
// default sew tolerance) sews into a valid closed solid of ~unit volume.
func TestSewClosesGappedLid(t *testing.T) {
	t.Parallel()
	quilt := gappedCubeQuilt(t, 5e-5)
	if len(BoundaryEdges(quilt)) == 0 {
		t.Fatal("premise: the gapped quilt must be open before sewing")
	}
	solid, err := Sew(quilt, 0)
	if err != nil {
		t.Fatalf("Sew: %v", err)
	}
	if !solid.IsSolid() {
		t.Error("sewn body should be a solid")
	}
	if r := Validate(solid); !r.Valid || !r.Closed || !r.Manifold {
		t.Errorf("sewn body validation = %+v, want fully valid", r)
	}
	if v := BodyGeometryProperties(solid, DefaultQuality()).Volume; stdmath.Abs(v-1) > 1e-3 {
		t.Errorf("sewn cube volume = %g, want ~1", v)
	}
}

// TestSewRecordsSeamResidual: the lid's rim edges record the absorbed gap as
// their healing tolerance, the way M25 import healing does.
func TestSewRecordsSeamResidual(t *testing.T) {
	t.Parallel()
	const gap = 5e-5
	solid, err := Sew(gappedCubeQuilt(t, gap), 0)
	if err != nil {
		t.Fatalf("Sew: %v", err)
	}
	maxResidual := 0.0
	for _, e := range solid.Edges() {
		if e.Tolerance() > maxResidual {
			maxResidual = e.Tolerance()
		}
	}
	if maxResidual < gap/4 || maxResidual > gap*2 {
		t.Errorf("max seam residual = %g, want about the %g gap", maxResidual, gap)
	}
}

// TestSewReportsOversizedGapPrecisely: a gap beyond the tolerance fails with
// every offending boundary edge and its measured gap in the error.
func TestSewReportsOversizedGapPrecisely(t *testing.T) {
	t.Parallel()
	quilt := gappedCubeQuilt(t, 0.1)
	_, err := Sew(quilt, 1e-3)
	if err == nil {
		t.Fatal("a 0.1 gap must not sew at tolerance 1e-3")
	}
	msg := err.Error()
	if !strings.Contains(msg, "boundary edges exceed tolerance") || !strings.Contains(msg, "gap") {
		t.Errorf("error %q should report the boundary edges and their gaps", msg)
	}
}

// TestSewAlreadyClosedPromotesToSolid: sewing a closed quilt is the
// stitch-to-solid promotion, not an error.
func TestSewAlreadyClosedPromotesToSolid(t *testing.T) {
	t.Parallel()
	quilt, err := Stitch(cubeFaces(), 0, true, "import")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	solid, err := Sew(quilt, 0)
	if err != nil {
		t.Fatalf("Sew: %v", err)
	}
	if !solid.IsSolid() {
		t.Error("sewing a closed quilt should promote it to a solid")
	}
}
