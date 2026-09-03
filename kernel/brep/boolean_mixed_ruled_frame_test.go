// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"fmt"
	stdmath "math"
	"sort"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The multipoint-disk pin (#3459): a 2 mm pin whose lower end was cut by a plane tilted 7.5° off
// its axis, so its cylinder face is framed by ONE circle rim and ONE elliptical rim. A profile
// extruded across it (along Z) then bites it three ways at once — a flat that wraps the whole
// azimuth, a six-facet chain that wraps it near the oblique end, and a tooth whose imprint closes
// into an island 6 µm under the circle rim. Every number is the translated part's own.
const (
	pinRadius  = 0.1
	pinAxisX   = 0.0
	pinAxisZ   = 1.15
	pinTopY    = -0.4
	pinCutY    = -1.5 // the tilted cut plane crosses the axis here
	pinTiltDeg = 7.5
)

// obliqueEndedPin builds the pin: a full cylinder along +Y, its lower end cut by the tilted plane.
func obliqueEndedPin(t *testing.T) *topo.Body {
	t.Helper()
	cyl, err := brep.SolidCylinder(math.P3(pinAxisX, pinCutY-0.2, pinAxisZ), math.V3(0, 1, 0), pinRadius, pinTopY-(pinCutY-0.2))
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	res, err := brep.BooleanDiag(brep.Difference, cyl, tiltedSlab(), nil)
	if err != nil {
		t.Fatalf("oblique end cut declined: %v", err)
	}
	return res
}

// tiltedSlab is a box rotated pinTiltDeg about Z whose TOP face is the cut plane through
// (pinAxisX, pinCutY, pinAxisZ); everything on the -Y side of that plane is removed by it.
func tiltedSlab() *topo.Body {
	a := pinTiltDeg * stdmath.Pi / 180
	ex, ey := math.V3(stdmath.Cos(a), stdmath.Sin(a), 0), math.V3(-stdmath.Sin(a), stdmath.Cos(a), 0)
	c := math.P3(pinAxisX, pinCutY, pinAxisZ)
	corner := func(sx, sy, z float64) math.Point3 {
		return c.TranslateBy(ex.Scale(math.Scalar(sx))).TranslateBy(ey.Scale(math.Scalar(sy))).TranslateBy(math.V3(0, 0, z))
	}
	verts := []math.Point3{
		corner(-1, -1, -1), corner(1, -1, -1), corner(1, 0, -1), corner(-1, 0, -1),
		corner(-1, -1, 1), corner(1, -1, 1), corner(1, 0, 1), corner(-1, 0, 1),
	}
	faces := [][]int{{3, 2, 1, 0}, {4, 5, 6, 7}, {0, 1, 5, 4}, {1, 2, 6, 5}, {2, 3, 7, 6}, {3, 0, 4, 7}}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, "slab")
}

// multipointProfile is the disk's cutting profile (in the XY plane), restricted to the piece that
// meets this pin; multipointProfileTool extrudes it along Z through the whole pin.
func multipointProfile() []math.Point3 {
	return []math.Point3{
		math.P3(-0.8, -1.4, 0),
		math.P3(-0.1488, -1.3921, 0), math.P3(-0.0993, -1.3965, 0), math.P3(-0.0497, -1.3991, 0), math.P3(0, -1.4, 0),
		math.P3(0.0497, -1.3991, 0), math.P3(0.0993, -1.3965, 0), math.P3(0.1488, -1.3921, 0),
		math.P3(0.44, -1.4, 0), math.P3(0.44, -0.6, 0), math.P3(0.1046, -0.5, 0), math.P3(-0.1046, -0.5, 0),
		math.P3(-0.1076, -0.4879, 0), math.P3(-0.0454, -0.4402, 0), math.P3(0, -0.4214, 0),
		math.P3(-0.0269, -0.4102, 0), math.P3(-0.1046, -0.4, 0), math.P3(-0.8, -0.4, 0),
	}
}

func multipointProfileTool() *topo.Body { return prismBody(multipointProfile(), 0, 1.4, "tool") }

// pinCutFloorY is the tilted end plane's height at x: the pin keeps y ≥ this.
func pinCutFloorY(x float64) float64 { return pinCutY + x*stdmath.Tan(pinTiltDeg*stdmath.Pi/180) }

// obliquePinVolume is the pin's closed-form volume: a cylinder of mean height (pinTopY − pinCutY)
// — the tilt's first moment over the disk vanishes.
func obliquePinVolume() float64 { return stdmath.Pi * pinRadius * pinRadius * (pinTopY - pinCutY) }

// profileYIntervalsAt returns the y-intervals in which the vertical line at x lies inside the profile
// (even-odd over its edge crossings, each edge half-open in x so a vertex counts once), each clipped
// to the pin's own y-range at that x.
func profileYIntervalsAt(x float64, profile []math.Point3) [][2]float64 {
	var ys []float64
	for i := range profile {
		p, q := profile[i], profile[(i+1)%len(profile)]
		px, qx := float64(p.X), float64(q.X)
		if (px <= x) == (qx <= x) {
			continue
		}
		ys = append(ys, float64(p.Y)+(float64(q.Y)-float64(p.Y))*(x-px)/(qx-px))
	}
	sort.Float64s(ys)
	var out [][2]float64
	for i := 0; i+1 < len(ys); i += 2 {
		lo, hi := stdmath.Max(ys[i], pinCutFloorY(x)), stdmath.Min(ys[i+1], pinTopY)
		if hi > lo {
			out = append(out, [2]float64{lo, hi})
		}
	}
	return out
}

// removedFromPin integrates the volume the Z-extruded profile takes out of the pin: at each x the
// z-chord of the pin (2√(r²−x²)) times the profile's y-length inside the pin. Substituting x = r·sinθ
// makes the integrand smooth between the profile vertices' abscissae, where Simpson converges to
// machine precision.
func removedFromPin(profile []math.Point3) float64 {
	breaks := []float64{-stdmath.Pi / 2, stdmath.Pi / 2}
	for _, p := range profile {
		if stdmath.Abs(float64(p.X)) < pinRadius {
			breaks = append(breaks, stdmath.Asin(float64(p.X)/pinRadius))
		}
	}
	sort.Float64s(breaks)
	f := func(th float64) float64 {
		x := pinRadius * stdmath.Sin(th)
		length := 0.0
		for _, iv := range profileYIntervalsAt(x, profile) {
			length += iv[1] - iv[0]
		}
		c := stdmath.Cos(th)
		return 2 * pinRadius * pinRadius * c * c * length
	}
	total := 0.0
	for i := 1; i < len(breaks); i++ {
		total += simpson(f, breaks[i-1], breaks[i], 2000)
	}
	return total
}

func simpson(f func(float64) float64, a, b float64, n int) float64 {
	h := (b - a) / float64(n)
	sum := f(a) + f(b)
	for i := 1; i < n; i++ {
		w := 2.0
		if i%2 == 1 {
			w = 4
		}
		sum += w * f(a+float64(i)*h)
	}
	return sum * h / 3
}

func describeCylinderFaces(b *topo.Body) string {
	s := ""
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); !ok {
			continue
		}
		s += fmt.Sprintf("cyl face loops=%d:", len(f.Loops()))
		for _, l := range f.Loops() {
			s += " ["
			for _, u := range l.EdgeUses() {
				s += fmt.Sprintf(" %T", u.Edge().Geometry())
			}
			s += " ]"
		}
		s += "\n"
	}
	return s
}

func TestObliqueEndedPinFixture(t *testing.T) {
	pin := obliqueEndedPin(t)
	if !pin.IsSolid() {
		t.Fatalf("pin is not a solid")
	}
	t.Logf("pin faces=%d\n%s", len(pin.Faces()), describeCylinderFaces(pin))
}

func TestObliqueEndedPinCutByProfileToolStaysAnalytic(t *testing.T) {
	pin := obliqueEndedPin(t)
	tool := multipointProfileTool()
	if !tool.IsSolid() {
		t.Fatalf("tool is not a solid")
	}
	res, err := brep.BooleanDiag(brep.Difference, pin, tool, nil)
	if err != nil {
		t.Fatalf("pin cut declined: %v", err)
	}
	if rep := ops.Validate(res); !rep.ValidSolid() || cylinderFaceCount(res) != 2 {
		t.Fatalf("pin cut invalid: %+v cyls=%d faces=%d\n%s", rep, cylinderFaceCount(res), len(res.Faces()), describeCylinderFaces(res))
	}
	want := obliquePinVolume() - removedFromPin(multipointProfile())
	if got := vol(res); stdmath.Abs(got-want) > 1e-9*want {
		t.Errorf("pin cut volume = %.15g, want %.15g (closed-form pin minus the integrated bite)", got, want)
	}
	if got := vol(pin); stdmath.Abs(got-obliquePinVolume()) > 1e-12 {
		t.Errorf("oblique pin volume = %.15g, want %.15g", got, obliquePinVolume())
	}
	assertPinFacesExact(t, res)
}

// assertPinFacesExact pins the result's exactness face by face: every loop closes, every edge of a
// cylinder face lies on the pin's cylinder, the top stub carries the tooth's island as a hole and
// the oblique stub keeps its elliptical rim.
func assertPinFacesExact(t *testing.T, res *topo.Body) {
	t.Helper()
	holes, ellipticalRims := 0, 0
	for i, f := range res.Faces() {
		for li, l := range f.Loops() {
			assertChainCloses(t, loopEndpointsInOrder(l), i)
			if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
				continue
			}
			if li > 0 {
				holes++
			}
			for _, u := range l.EdgeUses() {
				if _, isEll := u.Edge().Geometry().(geom.EllipticalArc); isEll && li == 0 {
					ellipticalRims++
				}
				for _, tp := range []float64{0, 0.37, 1} {
					p := u.Edge().Geometry().PointAt(tp)
					if d := stdmath.Hypot(float64(p.X)-pinAxisX, float64(p.Z)-pinAxisZ) - pinRadius; stdmath.Abs(d) > 1e-12 {
						t.Errorf("face %d edge point %v is %g off the pin cylinder", i, p, d)
					}
				}
			}
		}
	}
	if holes == 0 || ellipticalRims == 0 {
		t.Errorf("cylinder faces carry %d hole loops and %d elliptical outer edges; want the tooth island and the oblique rim", holes, ellipticalRims)
	}
}

// TestObliqueEndedPinUnderAPlateCutByProfileTool is the disk's own configuration: the pin is a boss
// under a plate, so the face at its top rim is the plate's bottom with the rim as a HOLE, and the
// tool's top edge rests in that plate's plane (a flush contact). The cut must leave one valid solid
// whose volume is the plate plus the pin less the integrated bite.
func TestObliqueEndedPinUnderAPlateCutByProfileTool(t *testing.T) {
	pin := obliqueEndedPin(t)
	plate, _ := brep.SolidBlock(math.P3(-0.5, pinTopY, pinAxisZ-0.5), math.P3(0.5, 0, pinAxisZ+0.5), "plate")
	body, err := brep.BooleanDiag(brep.Union, plate, pin, nil)
	if err != nil {
		t.Fatalf("pin under plate declined: %v", err)
	}
	if rep := ops.Validate(body); !rep.ValidSolid() {
		t.Fatalf("pin under plate invalid: %+v", rep)
	}
	res, err := brep.BooleanDiag(brep.Difference, body, multipointProfileTool(), nil)
	if err != nil {
		t.Fatalf("pin-under-plate cut declined: %v", err)
	}
	if rep := ops.Validate(res); !rep.ValidSolid() {
		t.Fatalf("pin-under-plate cut invalid: %+v", rep)
	}
	want := 1*0.4*1 + obliquePinVolume() - removedFromPin(multipointProfile())
	if got := vol(res); stdmath.Abs(got-want) > 1e-9*want {
		t.Errorf("pin-under-plate cut volume = %.15g, want %.15g", got, want)
	}
}

// TestObliqueEndedPinInAThickPlateCutTwice is the disk's next feature: the first profile leaves a
// planar floor at y=−0.5 in the plate with the stub attached through a circular HOLE, and a second
// tool then imprints that floor away from the stub. The floor is a polygonal face whose hole is
// re-attached EXACTLY after the split; its winding must follow the face, or the rim it shares with the
// stub's wall is traversed the same way by both. The second box's bite has a closed form.
func TestObliqueEndedPinInAThickPlateCutTwice(t *testing.T) {
	pin := obliqueEndedPin(t)
	plate, _ := brep.SolidBlock(math.P3(-0.5, -0.6, pinAxisZ-0.5), math.P3(0.5, 0, pinAxisZ+0.5), "plate")
	body, err := brep.BooleanDiag(brep.Union, plate, pin, nil)
	if err != nil {
		t.Fatalf("pin in plate declined: %v", err)
	}
	once, err := brep.BooleanDiag(brep.Difference, body, multipointProfileTool(), nil)
	if err != nil {
		t.Fatalf("first cut declined: %v", err)
	}
	if rep := ops.Validate(once); !rep.ValidSolid() {
		t.Fatalf("first cut invalid: %+v", rep)
	}
	box, _ := brep.SolidBlock(math.P3(0.2, -0.55, 1.0), math.P3(0.4, -0.45, 1.3), "box")
	twice, err := brep.BooleanDiag(brep.Difference, once, box, nil)
	if err != nil {
		t.Fatalf("second cut declined: %v", err)
	}
	if rep := ops.Validate(twice); !rep.ValidSolid() {
		t.Fatalf("second cut invalid: %+v", rep)
	}
	// Over the box's abscissae the first profile's upper boundary is its diagonal (0.1046,−0.5)→(0.44,−0.6),
	// so the box takes the plate from that diagonal (or its own floor at −0.55, whichever is higher) up to
	// −0.45, across z∈[1.0,1.3].
	slope := 0.1 / (0.44 - 0.1046)
	xStar := 0.1046 + 0.05/slope // where the diagonal passes y = −0.55
	sloped := 0.05*(xStar-0.2) + slope/2*((xStar-0.1046)*(xStar-0.1046)-(0.2-0.1046)*(0.2-0.1046))
	want := 0.3 * (0.1*(0.4-xStar) + sloped)
	if removed := vol(once) - vol(twice); stdmath.Abs(removed-want) > 1e-9 {
		t.Errorf("second cut removed %.12g, want %.12g", removed, want)
	}
}
