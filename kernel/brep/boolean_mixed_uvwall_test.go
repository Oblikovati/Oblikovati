// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Unit cover for the uv×wall conic pairing (#2247, #3460). The fixtures are the two operands of the
// headline contact: a Ø10 cylinder standing on the axis, and a 16×16 plate face the wall sections with a
// circle. Every gate is exercised on the exact geometry the boolean feeds it, so a decline here is the
// same decline the boolean makes.

// uvWallFixture builds the partitions of a cylinder (radius 5, z∈[0,4]) and a plate (16×16, z∈[1,3]).
func uvWallFixture(t *testing.T) (cyl, plate facePartition) {
	t.Helper()
	c, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 4)
	if err != nil {
		t.Fatal(err)
	}
	p, err := SolidBlock(math.P3(-8, -8, 1), math.P3(8, 8, 3), "plate")
	if err != nil {
		t.Fatal(err)
	}
	return partitionFaces(c), partitionFaces(p)
}

// planeFaceAtZ returns the plate's horizontal face at height z, from the polygonal bucket.
func planeFaceAtZ(t *testing.T, p facePartition, z float64) curvedFace {
	t.Helper()
	for _, f := range p.planar {
		box := faceLoopBox(f)
		if stdmath.Abs(float64(box.Min.Z)-z) < 1e-9 && stdmath.Abs(float64(box.Max.Z)-z) < 1e-9 {
			return f
		}
	}
	t.Fatalf("no plate face at z=%g", z)
	return curvedFace{}
}

// TestFaceLoopBoxIsUnpadded: the uv bucket's box convention is the exact loop-point extent.
func TestFaceLoopBoxIsUnpadded(t *testing.T) {
	t.Parallel()
	_, plate := uvWallFixture(t)
	box := faceLoopBox(planeFaceAtZ(t, plate, 3))
	if float64(box.Min.X) != -8 || float64(box.Max.X) != 8 || float64(box.Min.Z) != 3 || float64(box.Max.Z) != 3 {
		t.Errorf("faceLoopBox = %v..%v, want (-8,-8,3)..(8,8,3)", box.Min, box.Max)
	}
}

// TestPromoteConicReceiversMovesPlateFaces: the two plate faces the cylinder wall sections with a circle
// move to the uv bucket; the four side faces (which the wall misses) stay polygonal, index-aligned.
func TestPromoteConicReceiversMovesPlateFaces(t *testing.T) {
	t.Parallel()
	cyl, plate := uvWallFixture(t)
	promoteConicReceivers(&plate, &cyl)
	if len(plate.uv) != 2 || len(plate.planar) != 4 {
		t.Fatalf("buckets = {planar:%d uv:%d}, want {planar:4 uv:2}", len(plate.planar), len(plate.uv))
	}
	if len(plate.planarFull) != 4 || len(plate.planarHoles) != 4 || len(plate.uvBox) != 2 {
		t.Errorf("index alignment broken: planarFull=%d planarHoles=%d uvBox=%d",
			len(plate.planarFull), len(plate.planarHoles), len(plate.uvBox))
	}
	for _, f := range plate.planar {
		if box := faceLoopBox(f); box.Min.Z == box.Max.Z {
			t.Errorf("a horizontal plate face at z=%v stayed polygonal", box.Min.Z)
		}
	}
}

// TestPromoteConicReceiversLeavesWalllessOperandAlone: with no wall on the other side nothing moves.
func TestPromoteConicReceiversLeavesWalllessOperandAlone(t *testing.T) {
	t.Parallel()
	_, plate := uvWallFixture(t)
	other := facePartition{}
	promoteConicReceivers(&plate, &other)
	if len(plate.planar) != 6 || len(plate.uv) != 0 {
		t.Errorf("buckets = {planar:%d uv:%d}, want the untouched {planar:6 uv:0}", len(plate.planar), len(plate.uv))
	}
}

// TestWallConicEntersFace: the wall's section with a plate face inside the band is a circle in its trim;
// with the plate's vertical side face (which the wall's box misses) there is none.
func TestWallConicEntersFace(t *testing.T) {
	t.Parallel()
	cyl, plate := uvWallFixture(t)
	if !wallConicEntersFace(planeFaceAtZ(t, plate, 3), &cyl) {
		t.Error("the cylinder wall sections the plate's top face with a circle inside its trim")
	}
	for _, f := range plate.planar {
		if box := faceLoopBox(f); box.Min.Z != box.Max.Z && wallConicEntersFace(f, &cyl) {
			t.Errorf("a vertical plate face at %v..%v takes no conic from a coaxial wall", box.Min, box.Max)
		}
	}
}

// TestConicAxialSpan: a circle has no axial amplitude; a non-conic section is refused.
func TestConicAxialSpan(t *testing.T) {
	t.Parallel()
	c, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	center, amp, isConic := conicAxialSpan(c, math.V3(0, 0, 1))
	if !isConic || amp != 0 || center != math.P3(0, 0, 3) {
		t.Errorf("conicAxialSpan(circle) = (%v,%v,%v), want ((0,0,3),0,true)", center, amp, isConic)
	}
	if _, _, isConic := conicAxialSpan(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), math.V3(0, 0, 1)); isConic {
		t.Error("a line section is not a conic island")
	}
}

// TestConicBandPlacement: strictly inside, strictly clear, and straddling a rim (neither — declined).
func TestConicBandPlacement(t *testing.T) {
	t.Parallel()
	rs := ruledSide{axis: math.V3(0, 0, 1), band: coneSideBand_{bottom: math.P3(0, 0, 0), vMin: 0, vMax: 4}}
	for _, c := range []struct {
		name                  string
		z, amp                float64
		wantInside, wantClear bool
	}{
		{"inside the band", 3, 0, true, false},
		{"clear above the band", 9, 0, false, true},
		{"straddling the top rim", 4, 0, false, false},
		{"an ellipse reaching past the rim", 3.5, 1, false, false},
	} {
		inside, clear := conicBandPlacement(math.P3(0, 0, c.z), c.amp, rs)
		if inside != c.wantInside || clear != c.wantClear {
			t.Errorf("%s: (inside,clear) = (%v,%v), want (%v,%v)", c.name, inside, clear, c.wantInside, c.wantClear)
		}
	}
}

// TestConicIslandInFace: a circle inside the plate polygon is an island; one clear of it is not (but the
// query is exact); one CROSSING the polygon boundary declines.
func TestConicIslandInFace(t *testing.T) {
	t.Parallel()
	_, plate := uvWallFixture(t)
	top := planeFaceAtZ(t, plate, 3)
	for _, c := range []struct {
		name               string
		center             math.Point3
		radius             float64
		wantIsland, wantOK bool
	}{
		{"wholly inside the trim", math.P3(0, 0, 3), 5, true, true},
		{"wholly outside the trim", math.P3(40, 0, 3), 5, false, true},
		{"enclosing the whole trim", math.P3(0, 0, 3), 40, false, true},
		{"crossing the trim boundary", math.P3(0, 0, 3), 10, false, false},
	} {
		circle, err := geom.NewCircle(c.center, math.V3(0, 0, 1), c.radius)
		if err != nil {
			t.Fatal(err)
		}
		island, ok := conicIslandInFace(circle, top)
		if island != c.wantIsland || ok != c.wantOK {
			t.Errorf("%s: (island,ok) = (%v,%v), want (%v,%v)", c.name, island, ok, c.wantIsland, c.wantOK)
		}
	}
}

// TestConicCrossesFaceBoundary: exactly the crossing case reports true.
func TestConicCrossesFaceBoundary(t *testing.T) {
	t.Parallel()
	_, plate := uvWallFixture(t)
	top := planeFaceAtZ(t, plate, 3)
	pl := facePlane(top)
	for _, c := range []struct {
		radius float64
		want   bool
	}{{5, false}, {10, true}, {40, false}} {
		circle, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), c.radius)
		if err != nil {
			t.Fatal(err)
		}
		pc, ok := toPlaneConic(circle, pl)
		if !ok {
			t.Fatal("toPlaneConic failed on a circle in the face plane")
		}
		if got := conicCrossesFaceBoundary(pc, top); got != c.want {
			t.Errorf("radius %g: crosses = %v, want %v", c.radius, got, c.want)
		}
	}
}

// TestUvWallSharedImprintYieldsOneCircle: the promoted plate face and the wall share exactly the section
// circle of the wall at that height — the same curve value, which is what welds the two splits.
func TestUvWallSharedImprintYieldsOneCircle(t *testing.T) {
	t.Parallel()
	cyl, plate := uvWallFixture(t)
	promoteConicReceivers(&plate, &cyl)
	curves, ok := uvWallSharedImprint(plate.uv[0], cyl.wall[0])
	if !ok || len(curves) != 1 {
		t.Fatalf("uvWallSharedImprint = %d curves, ok=%v; want 1 circle", len(curves), ok)
	}
	circle, isCircle := curves[0].(geom.Circle)
	if !isCircle || circle.Radius != 5 {
		t.Errorf("imprint = %#v, want the wall's radius-5 section circle", curves[0])
	}
}

// TestUvWallSharedImprintDeclinesConicFrame: a receiver whose own boundary is curved (a cylinder cap)
// would need conic×conic frame crossings, so the pairing declines by name rather than approximating.
func TestUvWallSharedImprintDeclinesConicFrame(t *testing.T) {
	t.Parallel()
	cyl, _ := uvWallFixture(t)
	if _, ok := uvWallSharedImprint(cyl.uv[0], cyl.wall[0]); ok {
		t.Error("a circle-framed cap must decline the uv×wall pairing")
	}
}

// TestWallSectionIslandStraddlingRimDeclines: a section at the band's own rim is neither an island nor
// clear, so it declines instead of emitting an imprint that runs off the band.
func TestWallSectionIslandStraddlingRimDeclines(t *testing.T) {
	t.Parallel()
	cyl, plate := uvWallFixture(t)
	rs, ok := ruledSideBandOf(cyl.wall[0])
	if !ok {
		t.Fatal("the cylinder side is a ruled band")
	}
	circle, err := geom.NewCircle(math.P3(0, 0, 4), math.V3(0, 0, 1), 5) // exactly the band's top rim
	if err != nil {
		t.Fatal(err)
	}
	if island, ok, _ := wallSectionIsland(circle, planeFaceAtZ(t, plate, 3), rs); island || ok {
		t.Errorf("a section on the band rim = (%v,%v), want (false,false) — a named decline", island, ok)
	}
}

// TestCollectWallIslandsDropsClearSections: a section clear of both trims is no imprint, not a decline.
func TestCollectWallIslandsDropsClearSections(t *testing.T) {
	t.Parallel()
	cyl, plate := uvWallFixture(t)
	rs, _ := ruledSideBandOf(cyl.wall[0])
	far, err := geom.NewCircle(math.P3(60, 0, 3), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	out, ok := collectWallIslands([]geom.Curve3{far}, planeFaceAtZ(t, plate, 3), rs)
	if !ok || len(out) != 0 {
		t.Errorf("collectWallIslands = (%d curves, ok=%v), want (0, true)", len(out), ok)
	}
}

// TestPairUVWallImprintsWritesBothSides: the SAME curve lands on the uv face's list and on the wall's, so
// the two arrangements split on identical coordinates.
func TestPairUVWallImprintsWritesBothSides(t *testing.T) {
	t.Parallel()
	cyl, plate := uvWallFixture(t)
	promoteConicReceivers(&plate, &cyl)
	uvImp := make([][]geom.Curve3, len(plate.uv))
	wallImp := make([][]geom.Curve3, len(cyl.wall))
	if !pairUVWallImprints(&plate, &cyl, uvImp, wallImp) {
		t.Fatal("pairUVWallImprints declined the cylinder-through-plate contact")
	}
	if len(uvImp[0]) != 1 || len(uvImp[1]) != 1 || len(wallImp[0]) != 2 {
		t.Fatalf("imprint counts = uv{%d,%d} wall{%d}, want uv{1,1} wall{2}", len(uvImp[0]), len(uvImp[1]), len(wallImp[0]))
	}
	if uvImp[0][0] != wallImp[0][0] && uvImp[0][0] != wallImp[0][1] {
		t.Error("the uv face's imprint curve is not the wall's own: the two sides would split differently")
	}
}
