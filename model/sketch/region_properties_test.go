// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// rectangleProfile builds a closed axis-aligned rectangle profile [0,w]×[0,h]
// offset so its centroid is at (cx, cy).
func rectangleProfile(t *testing.T, w, h, cx, cy float64) *Profile {
	t.Helper()
	s := NewSketches().Add(XYPlane())
	x0, y0 := cx-w/2, cy-h/2
	corners := []gmath.Point2{
		gmath.P2(gmath.Scalar(x0), gmath.Scalar(y0)),
		gmath.P2(gmath.Scalar(x0+w), gmath.Scalar(y0)),
		gmath.P2(gmath.Scalar(x0+w), gmath.Scalar(y0+h)),
		gmath.P2(gmath.Scalar(x0), gmath.Scalar(y0+h)),
	}
	for i := range corners {
		s.Lines().AddByTwoPoints(corners[i], corners[(i+1)%4])
	}
	profiles := s.Profiles().All()
	if len(profiles) != 1 {
		t.Fatalf("rectangle yielded %d profiles, want 1", len(profiles))
	}
	return profiles[0]
}

// relErr is the relative error |got−want|/|want|.
func relErr(got, want float64) float64 { return math.Abs(got-want) / math.Abs(want) }

// TestRegionPropertiesRectangleExact: polygon integrals are exact for a
// rectangle — area b·h, centroid at the center, Ixx = b·h³/12, Iyy = h·b³/12,
// Ixy = 0, perimeter 2(b+h) — even off the origin (parallel-axis shift).
func TestRegionPropertiesRectangleExact(t *testing.T) {
	const b, h, cx, cy = 4.0, 2.0, 3.0, 5.0
	props, err := rectangleProfile(t, b, h, cx, cy).RegionProperties(types.AccuracyLow)
	if err != nil {
		t.Fatalf("RegionProperties: %v", err)
	}
	if math.Abs(props.Area()-b*h) > 1e-12 || math.Abs(props.Perimeter()-2*(b+h)) > 1e-12 {
		t.Errorf("area/perimeter = %v/%v, want %v/%v", props.Area(), props.Perimeter(), b*h, 2*(b+h))
	}
	gotX, gotY := props.Centroid()
	if math.Abs(gotX-cx) > 1e-12 || math.Abs(gotY-cy) > 1e-12 {
		t.Errorf("centroid = (%v, %v), want (%v, %v)", gotX, gotY, cx, cy)
	}
	ixx, iyy, ixy := props.MomentsOfInertia()
	if math.Abs(ixx-b*h*h*h/12) > 1e-9 || math.Abs(iyy-h*b*b*b/12) > 1e-9 || math.Abs(ixy) > 1e-9 {
		t.Errorf("moments = (%v, %v, %v), want (%v, %v, 0)", ixx, iyy, ixy, b*h*h*h/12, h*b*b*b/12)
	}
	// Wide rectangle: Iyy > Ixx, so the first principal axis is the y axis.
	if a := props.RotationAngle(); math.Abs(math.Abs(a)-math.Pi/2) > 1e-9 {
		t.Errorf("rotation angle = %v, want ±π/2 (I1 on the y axis)", a)
	}
}

// TestRegionPropertiesAnnulus: a circle with a concentric hole — area
// π(R²−r²), centroid at the center, Ixx = Iyy = π(R⁴−r⁴)/4, both rims in the
// perimeter. Curved boundaries converge with accuracy.
func TestRegionPropertiesAnnulus(t *testing.T) {
	const R, r = 2.0, 1.0
	s := NewSketches().Add(XYPlane())
	s.Circles().AddByCenterRadius(gmath.P2(0, 0), R)
	s.Circles().AddByCenterRadius(gmath.P2(0, 0), r)
	profiles := s.Profiles().All()
	annulus := profiles[0]
	for _, p := range profiles { // pick the region that carries the hole
		if len(p.InnerLoops()) == 1 {
			annulus = p
		}
	}
	props, err := annulus.RegionProperties(types.AccuracyVeryHigh)
	if err != nil {
		t.Fatalf("RegionProperties: %v", err)
	}
	if got, want := props.Area(), math.Pi*(R*R-r*r); relErr(got, want) > 1e-3 {
		t.Errorf("area = %v, want %v within 0.1%%", got, want)
	}
	if got, want := props.Perimeter(), 2*math.Pi*(R+r); relErr(got, want) > 1e-3 {
		t.Errorf("perimeter = %v, want both rims %v within 0.1%%", got, want)
	}
	ixx, iyy, _ := props.MomentsOfInertia()
	want := math.Pi * (R*R*R*R - r*r*r*r) / 4
	if relErr(ixx, want) > 5e-3 || relErr(iyy, want) > 5e-3 {
		t.Errorf("Ixx/Iyy = %v/%v, want %v within 0.5%%", ixx, iyy, want)
	}
	if a := props.RotationAngle(); a != 0 {
		t.Errorf("rotation angle = %v, want 0 for the isotropic annulus", a)
	}
}

// TestRegionPropertiesAccuracyConverges: higher accuracy must reduce the
// circle-area sampling error monotonically — the setting has to do something.
func TestRegionPropertiesAccuracyConverges(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Circles().AddByCenterRadius(gmath.P2(0, 0), 2)
	profile := s.Profiles().All()[0]
	exact := math.Pi * 4
	prev := math.Inf(1)
	for _, acc := range []types.Accuracy{
		types.AccuracyLow, types.AccuracyMedium, types.AccuracyHigh, types.AccuracyVeryHigh,
	} {
		props, err := profile.RegionProperties(acc)
		if err != nil {
			t.Fatalf("RegionProperties(%v): %v", acc, err)
		}
		gap := math.Abs(props.Area() - exact)
		if gap >= prev {
			t.Errorf("accuracy %v did not reduce the area error (%g → %g)", acc, prev, gap)
		}
		prev = gap
	}
}

// TestRegionPropertiesRotatedPrincipalAxes: a rectangle rotated 30° must
// report its principal axes rotated by the same angle and the same principal
// moments as the axis-aligned one.
func TestRegionPropertiesRotatedPrincipalAxes(t *testing.T) {
	const b, h, theta = 4.0, 2.0, math.Pi / 6
	s := NewSketches().Add(XYPlane())
	rot := func(x, y float64) gmath.Point2 {
		c, sn := math.Cos(theta), math.Sin(theta)
		return gmath.P2(gmath.Scalar(x*c-y*sn), gmath.Scalar(x*sn+y*c))
	}
	corners := []gmath.Point2{rot(-b/2, -h/2), rot(b/2, -h/2), rot(b/2, h/2), rot(-b/2, h/2)}
	for i := range corners {
		s.Lines().AddByTwoPoints(corners[i], corners[(i+1)%4])
	}
	props, err := s.Profiles().All()[0].RegionProperties(types.AccuracyHigh)
	if err != nil {
		t.Fatalf("RegionProperties: %v", err)
	}
	i1, i2 := props.PrincipalMoments()
	if math.Abs(i1-h*b*b*b/12) > 1e-9 || math.Abs(i2-b*h*h*h/12) > 1e-9 {
		t.Errorf("principal moments = (%v, %v), want (%v, %v)", i1, i2, h*b*b*b/12, b*h*h*h/12)
	}
	// The wide rectangle's first principal axis is its short (y) symmetry
	// axis, rotated by theta; angles are defined modulo π.
	want := math.Mod(math.Pi/2+theta, math.Pi)
	got := math.Mod(props.RotationAngle()+math.Pi, math.Pi)
	if math.Abs(got-want) > 1e-9 && math.Abs(math.Abs(got-want)-math.Pi) > 1e-9 {
		t.Errorf("rotation angle = %v (mod π %v), want %v", props.RotationAngle(), got, want)
	}
	first, second := props.PrincipalAxes()
	if math.Abs(float64(first.Dot(second))) > 1e-12 {
		t.Errorf("principal axes %v/%v are not orthogonal", first, second)
	}
}

// TestRegionPropertiesRejectsOpenProfile: an open chain encloses nothing.
func TestRegionPropertiesRejectsOpenProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	for _, p := range s.Profiles().All() {
		if p.IsClosed() {
			continue
		}
		if _, err := p.RegionProperties(types.AccuracyHigh); err == nil {
			t.Error("an open profile must be rejected")
		}
	}
}

// TestRegionProperties3DProfileInPlane: a planar 3D rectangle loop reports
// its section properties in its own plane frame.
func TestRegionProperties3DProfileInPlane(t *testing.T) {
	s3 := NewSketches3D().Add()
	// A 4×2 rectangle in the plane z = y (tilted 45° about x).
	k := math.Sqrt2 / 2
	corners := []gmath.Point3{
		gmath.P3(0, 0, 0), gmath.P3(4, 0, 0),
		gmath.P3(4, gmath.Scalar(2*k), gmath.Scalar(2*k)), gmath.P3(0, gmath.Scalar(2*k), gmath.Scalar(2*k)),
	}
	for i := range corners {
		s3.AddLine3D(corners[i], corners[(i+1)%len(corners)])
	}
	profiles := s3.Profiles3D()
	if len(profiles) != 1 {
		t.Fatalf("profiles = %d, want the tilted rectangle", len(profiles))
	}
	props, err := profiles[0].RegionProperties(types.AccuracyHigh)
	if err != nil {
		t.Fatalf("RegionProperties: %v", err)
	}
	if math.Abs(props.Area()-8) > 1e-9 || math.Abs(props.Perimeter()-12) > 1e-9 {
		t.Errorf("area/perimeter = %v/%v, want 8/12 in the loop plane", props.Area(), props.Perimeter())
	}
	ixx, iyy, ixy := props.MomentsOfInertia()
	if math.Abs(ixx-4*8/12.0) > 1e-9 || math.Abs(iyy-2*64/12.0) > 1e-9 || math.Abs(ixy) > 1e-9 {
		t.Errorf("moments = (%v, %v, %v), want the plane-frame rectangle values", ixx, iyy, ixy)
	}
}
