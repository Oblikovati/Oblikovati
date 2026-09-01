// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestMixTransmissionZeroWeightReproducesOpaqueExactly is PBI-344's explicit regression
// guard: transmission_weight=0 must reproduce PBI-343's output exactly.
func TestMixTransmissionZeroWeightReproducesOpaqueExactly(t *testing.T) {
	t.Parallel()
	opaque := NewColor3(0.31, 0.22, 0.05)
	got := MixTransmission(opaque, Gray(0.9), 0)
	if got != opaque {
		t.Errorf("MixTransmission(weight=0) = %+v, want opaque unchanged = %+v", got, opaque)
	}
}

// --- IORStack: nested-dielectric unit test (glass submerged in water) ---

func TestIORStackGlassInWaterScenario(t *testing.T) {
	t.Parallel()
	const airIOR, waterIOR, glassIOR = 1.0, 1.33, 1.5
	s := NewIORStack()
	if got := s.Top(); got != airIOR {
		t.Fatalf("initial Top() = %v, want ambient %v", got, airIOR)
	}

	// Ray enters the water from air.
	if got := s.RelativeIOR(waterIOR); math.Abs(got-waterIOR/airIOR) > 1e-9 {
		t.Errorf("air→water RelativeIOR = %v, want %v", got, waterIOR/airIOR)
	}
	s.Push(waterIOR)
	if s.Depth() != 2 || s.Top() != waterIOR {
		t.Fatalf("after entering water: depth=%d top=%v, want depth=2 top=%v", s.Depth(), s.Top(), waterIOR)
	}

	// Ray enters the submerged glass bead from water.
	if got := s.RelativeIOR(glassIOR); math.Abs(got-glassIOR/waterIOR) > 1e-9 {
		t.Errorf("water→glass RelativeIOR = %v, want %v", got, glassIOR/waterIOR)
	}
	s.Push(glassIOR)
	if s.Depth() != 3 || s.Top() != glassIOR {
		t.Fatalf("after entering glass: depth=%d top=%v, want depth=3 top=%v", s.Depth(), s.Top(), glassIOR)
	}

	// Ray exits the glass back into water.
	s.Pop()
	if s.Depth() != 2 || s.Top() != waterIOR {
		t.Fatalf("after exiting glass: depth=%d top=%v, want depth=2 top=%v", s.Depth(), s.Top(), waterIOR)
	}

	// Ray exits the water back into air.
	s.Pop()
	if s.Depth() != 1 || s.Top() != airIOR {
		t.Fatalf("after exiting water: depth=%d top=%v, want depth=1 top=%v", s.Depth(), s.Top(), airIOR)
	}
}

func TestIORStackPopBelowAmbientIsNoOp(t *testing.T) {
	t.Parallel()
	s := NewIORStack()
	s.Pop() // popping the ambient medium itself must not underflow
	if s.Depth() != 1 || s.Top() != 1 {
		t.Errorf("Pop() below ambient: depth=%d top=%v, want depth=1 top=1", s.Depth(), s.Top())
	}
}

// --- Abbe-number dispersion ---

// TestDispersiveIORZeroScaleIsWavelengthIndependent is a regression guard: scale=0 must
// reproduce the plain (undispersed) IOR at every wavelength.
func TestDispersiveIORZeroScaleIsWavelengthIndependent(t *testing.T) {
	t.Parallel()
	const nd = 1.5
	for _, lambda := range []float64{fraunhoferLambdaCNM, fraunhoferLambdaDNM, fraunhoferLambdaFNM} {
		if got := DispersiveIOR(nd, 20, 0, lambda); got != nd {
			t.Errorf("DispersiveIOR(scale=0, λ=%v) = %v, want %v unchanged", lambda, got, nd)
		}
	}
}

// TestDispersiveIORMatchesReferenceAtLambdaD checks the hand-derivable identity: the
// Cauchy coefficients are defined so n(λd) = nd exactly, by construction of the spec's
// own formula.
func TestDispersiveIORMatchesReferenceAtLambdaD(t *testing.T) {
	t.Parallel()
	const nd = 1.52
	got := DispersiveIOR(nd, 20, 1, fraunhoferLambdaDNM)
	if math.Abs(got-nd) > 1e-9 {
		t.Errorf("DispersiveIOR(λ=λd) = %v, want nd = %v", got, nd)
	}
}

// TestDispersiveIORBluerIsHigher checks normal dispersion's defining qualitative
// property: shorter wavelengths (blue, λF) refract more than longer ones (red, λC), i.e.
// n(λF) > n(λd) > n(λC), for a typical positive-dispersion dielectric.
func TestDispersiveIORBluerIsHigher(t *testing.T) {
	t.Parallel()
	const nd = 1.52
	nRed := DispersiveIOR(nd, 20, 1, fraunhoferLambdaCNM)
	nYellow := DispersiveIOR(nd, 20, 1, fraunhoferLambdaDNM)
	nBlue := DispersiveIOR(nd, 20, 1, fraunhoferLambdaFNM)
	if !(nBlue > nYellow && nYellow > nRed) {
		t.Errorf("dispersion ordering: red=%v yellow=%v blue=%v, want red < yellow < blue", nRed, nYellow, nBlue)
	}
}

// TestDispersiveIORLowerAbbeIsMoreDispersive checks the spec's stated inverse
// relationship (index.html line 823): a lower Abbe number must produce a LARGER spread
// between the red and blue IORs (more dispersion).
func TestDispersiveIORLowerAbbeIsMoreDispersive(t *testing.T) {
	t.Parallel()
	spread := func(abbe float64) float64 {
		return DispersiveIOR(1.52, abbe, 1, fraunhoferLambdaFNM) - DispersiveIOR(1.52, abbe, 1, fraunhoferLambdaCNM)
	}
	if spread(9.87) <= spread(55) {
		t.Errorf("low-Abbe (rutile-like, 9.87) spread=%v, want > high-Abbe (water-like, 55) spread=%v", spread(9.87), spread(55))
	}
}

// --- Vector Snell refraction ---

func TestRefractNormalIncidenceGoesStraightThrough(t *testing.T) {
	t.Parallel()
	wt, ok := Refract(Vec3{Z: 1}, 1.5)
	if !ok {
		t.Fatal("Refract(normal incidence) reported no refraction")
	}
	want := Vec3{Z: -1}
	if math.Abs(wt.X-want.X) > 1e-9 || math.Abs(wt.Y-want.Y) > 1e-9 || math.Abs(wt.Z-want.Z) > 1e-9 {
		t.Errorf("Refract(normal, iorRatio=1.5) = %+v, want straight through %+v", wt, want)
	}
}

// TestRefractSatisfiesSnellsLaw checks the defining relationship sinθt = sinθi/iorRatio
// at an oblique angle.
func TestRefractSatisfiesSnellsLaw(t *testing.T) {
	t.Parallel()
	const iorRatio = 1.5
	thetaI := 30.0 * math.Pi / 180
	wi := Vec3{X: math.Sin(thetaI), Z: math.Cos(thetaI)}
	wt, ok := Refract(wi, iorRatio)
	if !ok {
		t.Fatal("Refract(30°, ior=1.5) reported no refraction")
	}
	sinThetaT := math.Sqrt(wt.X*wt.X + wt.Y*wt.Y)
	wantSinThetaT := math.Sin(thetaI) / iorRatio
	if math.Abs(sinThetaT-wantSinThetaT) > 1e-9 {
		t.Errorf("Refract: sinθt = %v, want sinθi/iorRatio = %v", sinThetaT, wantSinThetaT)
	}
}

// TestRefractTotalInternalReflection checks the TIR case (a ray in a denser medium
// beyond the critical angle, refracting toward a rarer one) reports no refraction.
func TestRefractTotalInternalReflection(t *testing.T) {
	t.Parallel()
	// Critical angle for ior 1/1.5 is asin(1.5) which doesn't exist in [0,90°] — TIR at
	// any angle steep enough. asin(1/1.5) ≈ 41.8° is the actual critical angle going the
	// OTHER way (dense→rare with iorRatio=1/1.5); pick 60° > that.
	thetaI := 60.0 * math.Pi / 180
	wi := Vec3{X: math.Sin(thetaI), Z: math.Cos(thetaI)}
	if _, ok := Refract(wi, 1/1.5); ok {
		t.Error("Refract beyond the critical angle should report TIR (ok=false)")
	}
}

// --- Beer's-law transmission extinction ---

func TestTransmissionExtinctionBeersLaw(t *testing.T) {
	t.Parallel()
	got := TransmissionExtinction(Color3{R: math.Exp(-1), G: 1, B: 1}, 1)
	want := Color3{R: 1, G: 0, B: 0}
	if math.Abs(got.R-want.R) > 1e-9 || math.Abs(got.G-want.G) > 1e-9 || math.Abs(got.B-want.B) > 1e-9 {
		t.Errorf("TransmissionExtinction((e⁻¹,1,1), depth=1) = %+v, want %+v", got, want)
	}
}

func TestTransmissionExtinctionZeroDepthIsAbsent(t *testing.T) {
	t.Parallel()
	if got := TransmissionExtinction(Gray(0.5), 0); got != (Color3{}) {
		t.Errorf("TransmissionExtinction(depth=0) = %+v, want zero (medium absent)", got)
	}
}

// --- Thin-walled mode ---

// TestThinWallReflectanceAndTransmittanceSumToOne checks the closed-form identity
// 2R/(1+R) + (1-R)/(1+R) = 1 — the geometric series' reflectance and transmittance must
// exactly conserve energy for any Fresnel R.
func TestThinWallReflectanceAndTransmittanceSumToOne(t *testing.T) {
	t.Parallel()
	for _, cosTheta := range []float64{0.05, 0.3, 0.6, 1.0} {
		refl := ThinWallFresnel(1.5, cosTheta)
		trans := ThinWallTransmittance(1.5, cosTheta)
		if math.Abs(refl+trans-1) > 1e-9 {
			t.Errorf("cos=%v: ThinWallFresnel+ThinWallTransmittance = %v, want exactly 1", cosTheta, refl+trans)
		}
	}
}

func TestThinWallFresnelExceedsSingleFaceFresnel(t *testing.T) {
	t.Parallel()
	single := DielectricFresnel(1.5, 0.7)
	thin := ThinWallFresnel(1.5, 0.7)
	if thin <= single {
		t.Errorf("ThinWallFresnel = %v, want > single-face DielectricFresnel = %v (both faces contribute)", thin, single)
	}
}
