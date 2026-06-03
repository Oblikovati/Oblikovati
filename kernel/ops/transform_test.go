// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/subd"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// boxBody builds a validated 2×2×2 solid box at the origin via the sub-D primitive.
func boxBody(t *testing.T) *topo.Body {
	t.Helper()
	b := subd.ToBody(subd.Box(2, 2, 2), "src")
	if !ops.Validate(b).Valid {
		t.Fatal("boxBody: source box is not valid")
	}
	return b
}

func keepLineage(l topo.Lineage) topo.Lineage { return l }

func TestTransformBodyTranslatePreservesVolume(t *testing.T) {
	src := boxBody(t)
	srcVol := ops.BodyGeometryProperties(src, ops.DefaultQuality()).Volume

	dst, err := ops.TransformBody(src, math.Translation4(math.V3(10, 0, 0)), keepLineage)
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	if r := ops.Validate(dst); !r.Valid {
		t.Fatalf("translated body invalid: %v", r.Issues)
	}
	if !dst.IsSolid() {
		t.Fatal("translated body should stay solid")
	}
	box := dst.RangeBox()
	if stdmath.Abs(box.Min.X-10) > 1e-9 || stdmath.Abs(box.Max.X-12) > 1e-9 {
		t.Fatalf("range box X = [%g,%g], want [10,12]", box.Min.X, box.Max.X)
	}
	if got := ops.BodyGeometryProperties(dst, ops.DefaultQuality()).Volume; stdmath.Abs(got-srcVol) > 1e-6 {
		t.Fatalf("volume changed under translation: got %g want %g", got, srcVol)
	}
}

func TestTransformBodyReflectStaysManifoldAndOutward(t *testing.T) {
	src := boxBody(t)
	srcVol := ops.BodyGeometryProperties(src, ops.DefaultQuality()).Volume

	reflect := math.Scale4(-1, 1, 1) // determinant -1
	dst, err := ops.TransformBody(src, reflect, keepLineage)
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	r := ops.Validate(dst)
	if !r.Valid || !r.Manifold || !r.OrientationOK || !r.Closed {
		t.Fatalf("reflected body not a valid manifold solid: %+v", r)
	}
	box := dst.RangeBox()
	if stdmath.Abs(box.Min.X-(-2)) > 1e-9 || stdmath.Abs(box.Max.X-0) > 1e-9 {
		t.Fatalf("reflected range box X = [%g,%g], want [-2,0]", box.Min.X, box.Max.X)
	}
	// Volume magnitude is preserved by a reflection (the divergence-theorem sum
	// stays positive because the winding flip keeps normals outward).
	if got := ops.BodyGeometryProperties(dst, ops.DefaultQuality()).Volume; got <= 0 || stdmath.Abs(got-srcVol) > 1e-6 {
		t.Fatalf("reflected volume = %g, want +%g (outward normals)", got, srcVol)
	}
}

func TestTransformBodyIdentityLineageRebindsKeys(t *testing.T) {
	src := boxBody(t)
	srcFaceKey := src.Faces()[0].ReferenceKey()

	dst, err := ops.TransformBody(src, math.Translation4(math.V3(1, 2, 3)), keepLineage)
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	if _, ok := dst.FindFaceByKey(srcFaceKey); !ok {
		t.Fatal("identity-lineage move should preserve the source face reference key")
	}
}

func TestTransformBodyCopyLineageGivesDistinctKeys(t *testing.T) {
	src := boxBody(t)
	srcFaceKey := src.Faces()[0].ReferenceKey()
	copyN := func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append(l.Tokens(), topo.Tok("pattern", "copy", 1))...)
	}

	dst, err := ops.TransformBody(src, math.Translation4(math.V3(5, 0, 0)), copyN)
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	if !ops.Validate(dst).Valid {
		t.Fatal("copy body should be valid")
	}
	if _, ok := dst.FindFaceByKey(srcFaceKey); ok {
		t.Fatal("a distinct-lineage copy must not collide with the source's reference keys")
	}
}

func TestTransformBodyRejectsNonUniformScale(t *testing.T) {
	src := boxBody(t)
	if _, err := ops.TransformBody(src, math.Scale4(2, 1, 1), keepLineage); err == nil {
		t.Fatal("non-uniform scale should be rejected (analytic types cannot represent it)")
	}
}
