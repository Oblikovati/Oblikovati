// SPDX-License-Identifier: GPL-2.0-only

package transform_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// boredBlock builds a 3×3×1.5 block with a through bore (radius 0.4) and returns the body
// plus the bore wall's reference key and analytic cylinder. The bore face is REVERSED — its
// surface +radial normal points into the solid, so the face must carry the opposite sense.
func boredBlock(t *testing.T) (*topo.Body, []byte, geom.Cylinder) {
	t.Helper()
	block := subd.ToBody(subd.Box(3, 3, 1.5), "blk")
	bored, err := brep.CutCylindricalHole(block, math.P3(1.5, 1.5, -0.1), math.V3(0, 0, 1), 0.4)
	if err != nil {
		t.Fatalf("CutCylindricalHole: %v", err)
	}
	for _, f := range bored.Faces() {
		if cyl, ok := f.Geometry().(geom.Cylinder); ok {
			return bored, f.ReferenceKey(), cyl
		}
	}
	t.Fatal("boredBlock: no cylindrical bore face")
	return nil, nil, geom.Cylinder{}
}

// TestTransformedReflectedToolCuts pins the mirror bug the fixing-block part surfaced: a
// reflected tool must REMOVE material when cut, not add it. geom.TransformSurface used to
// rebuild a reflected plane's normal as (m·U)×(m·V) = −m·N (inward), and the planar B-rep
// boolean trusts a face's surface normal as outward — so cutting a mirrored tool inverted
// the classification and the volume went UP (a mirrored hole added material). The fix keeps
// the reflected normal outward; here a tool reflected onto the block must cut a clean slot.
func TestTransformedReflectedToolCuts(t *testing.T) {
	t.Parallel()
	block := subd.ToBody(subd.Box(4, 2, 2), "blk") // x∈[0,4], volume 16
	tool := subd.ToBody(subd.Box(1, 2, 2), "tool") // volume 4
	left, err := transform.TransformBody(tool, math.Translation4(math.V3(0.5, 0, 0)), keepLineage)
	if err != nil {
		t.Fatalf("translate tool: %v", err)
	}
	nrm, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	mirrored, err := transform.TransformBody(left, math.Reflection4(math.P3(2, 0, 0), nrm), keepLineage)
	if err != nil {
		t.Fatalf("reflect tool: %v", err)
	}
	res, err := ops.Boolean(ops.Cut, block, mirrored)
	if err != nil {
		t.Fatalf("cut mirrored tool: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("cut result not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-12) > 1e-6 {
		t.Fatalf("reflected-tool cut volume = %.4f, want 12 (16−4); >16 means the mirror added material", got)
	}
}

// TestReplaceFaceSurfacePreservesReversedSense pins the bug where ReplaceFaceSurface dropped
// a face's reversed sense: retyping a bored (reversed) wall to an internal threaded cylinder
// that cuts OUTWARD must REMOVE material (volume drops). Before the fix the new face came out
// un-reversed, flipping its divergence-theorem contribution and ADDING ~0.9 cm³ instead.
//
// The sense is asserted DIRECTLY as well as through the volume. It used to be checked only through
// that proxy, and the proxy broke for a reason with nothing to do with the sense: the analytic
// integrator read the retyped surface's finite v-domain as CLOSEDNESS, took the wall's whole trim
// for the complement of itself, and dropped its contribution — 0.5 cm³ and 3.77 cm² gone with the
// sense entirely intact (Oblikovati/Oblikovati#3453). A proxy that can move on its own must be
// backed by the thing it stands for.
func TestReplaceFaceSurfacePreservesReversedSense(t *testing.T) {
	t.Parallel()
	bored, key, cyl := boredBlock(t)
	plain := ops.BodyGeometryProperties(bored, ops.DefaultQuality())
	plainVol := plain.Volume

	// Depth 0 → identical geometry to the plain bore: volume AND area must be unchanged (a pure
	// sense/integration-path check, isolated from any thread geometry).
	flat := geom.ThreadedCylinder{Cylinder: cyl, Pitch: 0.125, Depth: 0, Internal: true, RightHanded: true, VMin: 0, VMax: 1.5}
	flatBody, err := transform.ReplaceFaceSurface(bored, key, flat)
	if err != nil {
		t.Fatalf("ReplaceFaceSurface(depth0): %v", err)
	}
	if !threadedWall(t, flatBody).Reversed() {
		t.Error("the retyped wall lost its reversed sense: a bore's +radial normal points INTO the solid")
	}
	got := ops.BodyGeometryProperties(flatBody, ops.DefaultQuality())
	if stdmath.Abs(got.Volume-plainVol) > 1e-3 {
		t.Fatalf("depth-0 retype changed volume: got %.4f want %.4f", got.Volume, plainVol)
	}
	if stdmath.Abs(got.Area-plain.Area) > 1e-3 {
		t.Fatalf("depth-0 retype changed area: got %.4f want %.4f (the wall's own contribution went missing)",
			got.Area, plain.Area)
	}

	// A real internal thread cuts outward → removes material → volume strictly below the bore.
	cut := geom.ThreadedCylinder{Cylinder: cyl, Pitch: 0.125, Depth: 0.068, Internal: true, RightHanded: true, VMin: 0, VMax: 1.5}
	cutBody, err := transform.ReplaceFaceSurface(bored, key, cut)
	if err != nil {
		t.Fatalf("ReplaceFaceSurface(cut): %v", err)
	}
	if r := ops.Validate(cutBody); !r.Valid || !r.Closed || !r.Manifold {
		t.Fatalf("threaded body not a valid closed solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(cutBody, ops.DefaultQuality()).Volume; got >= plainVol {
		t.Fatalf("internal thread volume = %.4f, must be < bore %.4f (it cuts outward)", got, plainVol)
	}
}

// threadedWall returns the body's threaded-cylinder face — the bore wall after a retype.
func threadedWall(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.ThreadedCylinder); ok {
			return f
		}
	}
	t.Fatal("threadedWall: the body carries no threaded-cylinder face")
	return nil
}

// TestAngularDeflectionRoundsSmallHoles pins the facet floor: a small bore must not render as
// a coarse polygon. The angular-deflection bound in DefaultQuality forces a full circle to at
// least 24 facets regardless of radius (a chord tolerance alone left a 4 mm bore an octagon).
func TestAngularDeflectionRoundsSmallHoles(t *testing.T) {
	t.Parallel()
	bored, _, _ := boredBlock(t)
	for _, e := range bored.Edges() {
		if _, ok := e.Geometry().(geom.Circle); !ok {
			continue
		}
		if facets := len(tessellate.TessellateEdge(e, ops.DefaultQuality())) - 1; facets < 24 {
			t.Fatalf("bore circle tessellated to %d facets, want ≥24 (looks polygonal)", facets)
		}
	}
}

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
	t.Parallel()
	src := boxBody(t)
	srcVol := ops.BodyGeometryProperties(src, ops.DefaultQuality()).Volume

	dst, err := transform.TransformBody(src, math.Translation4(math.V3(10, 0, 0)), keepLineage)
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
	t.Parallel()
	src := boxBody(t)
	srcVol := ops.BodyGeometryProperties(src, ops.DefaultQuality()).Volume

	reflect := math.Scale4(-1, 1, 1) // determinant -1
	dst, err := transform.TransformBody(src, reflect, keepLineage)
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
	t.Parallel()
	src := boxBody(t)
	srcFaceKey := src.Faces()[0].ReferenceKey()

	dst, err := transform.TransformBody(src, math.Translation4(math.V3(1, 2, 3)), keepLineage)
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	if _, ok := dst.FindFaceByKey(srcFaceKey); !ok {
		t.Fatal("identity-lineage move should preserve the source face reference key")
	}
}

func TestTransformBodyCopyLineageGivesDistinctKeys(t *testing.T) {
	t.Parallel()
	src := boxBody(t)
	srcFaceKey := src.Faces()[0].ReferenceKey()
	copyN := func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append(l.Tokens(), topo.Tok("pattern", "copy", 1))...)
	}

	dst, err := transform.TransformBody(src, math.Translation4(math.V3(5, 0, 0)), copyN)
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
	t.Parallel()
	src := boxBody(t)
	if _, err := transform.TransformBody(src, math.Scale4(2, 1, 1), keepLineage); err == nil {
		t.Fatal("non-uniform scale should be rejected (analytic types cannot represent it)")
	}
}
