// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Sheet-metal Punch Tool feature (M13-F03). A punch stamps every closed profile of a sketch
// through the sheet in one shot — the die-pattern counterpart of the single-profile Cut, for a
// row of louvers, a grid of vents, or a perforation array placed from one sketch. By default it
// punches through all the material; a depth limits it to a coined/embossed cutout. The geometry
// reuses the shared profile-prism builder + through-all span, so a punch is the boolean
// complement of the sketch's thickened profiles.
//
// A punch is a die tool, so it also carries the tool's settings (Inventor's PunchToolFeature,
// #1968): an Angle that turns the die about its centroid, whether it may punch across a bend, a
// flat-pattern representation, and a tool id — the metadata the flat punch results and drawing
// punch notes read.

// SheetMetalPunchDefinition is the punch recipe: the sketch whose closed profiles are stamped, an
// optional depth (nil ⇒ through all), and the die-tool settings (#1968).
type SheetMetalPunchDefinition struct {
	Sketch    *sketch.Sketch
	Direction ExtentDirection
	Depth     func() float64 // nil ⇒ punch through all the material
	// Angle turns the punched profiles about their centroid (radians); nil/0 ⇒ no rotation.
	Angle func() float64
	// AcrossBends lets the punch span a bend; UnfoldInFlat controls whether it develops into the
	// flat; ToolID names the die; Representation is its flat/drawing appearance.
	AcrossBends    bool
	UnfoldInFlat   bool
	ToolID         string
	Representation types.PunchRepresentationType
}

// SheetMetalPunchFeature stamps the sketch's profiles through the running sheet each recompute.
type SheetMetalPunchFeature struct {
	def      *SheetMetalPunchDefinition
	featName string
}

// Definition returns the punch recipe.
func (f *SheetMetalPunchFeature) Definition() *SheetMetalPunchDefinition { return f.def }

// Kind identifies the feature for serialization and the model tree.
func (f *SheetMetalPunchFeature) Kind() string { return "sheet-metal-punch" }

// Recompute resolves every closed profile of the sketch, builds the punch tool, and subtracts
// it from the sheet.
func (f *SheetMetalPunchFeature) Recompute(in Input) (Output, error) {
	n := f.def.Sketch.Profiles().Count()
	if n == 0 {
		return Output{}, fmt.Errorf("sheet-metal punch: the sketch has no closed profile to punch")
	}
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	profiles, err := resolveClosedProfiles(f.def.Sketch, indices, "sheet-metal punch")
	if err != nil {
		return Output{}, err
	}
	plane := f.punchPlane(profiles)
	sp, err := f.punchSpan(in.Bodies, plane)
	if err != nil {
		return Output{}, err
	}
	tool := buildProfilePrisms(profiles, plane, sp, 0, f.featName, in.Diag)
	bodies, err := combine(in, tool, ops.Cut)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// punchPlane is the sketch plane the punch prisms are built on, turned by the die's Angle about the
// profiles' centroid. Rotating the plane's IN-PLANE frame (not the sketch geometry) turns every
// profile about that centroid while the same 2D coordinates map through it, so the shared prism
// builder needs no change. The normal is unchanged (an in-plane rotation), so the through-all span
// is unaffected.
func (f *SheetMetalPunchFeature) punchPlane(profiles []*sketch.Profile) sketch.Plane {
	plane := f.def.Sketch.Plane()
	angle := evalFloat(f.def.Angle)
	if angle == 0 {
		return plane
	}
	return rotatedPlaneAboutCentroid(plane, profilesCentroid2(profiles), angle)
}

// rotatedPlaneAboutCentroid returns plane with its X/Y axes turned by angle and its origin shifted
// so a 2D point q maps to plane.ToModel(c + R·(q−c)) — an in-plane rotation about the 2D centroid c.
// Derivation: X' = cosθ·X + sinθ·Y, Y' = −sinθ·X + cosθ·Y, O' = M(c) − (Rc)ᵤ·X − (Rc)ᵥ·Y.
func rotatedPlaneAboutCentroid(plane sketch.Plane, c math.Point2, angle float64) sketch.Plane {
	x, y := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	cos, sin := stdmath.Cos(angle), stdmath.Sin(angle)
	xp := x.Scale(math.Scalar(cos)).Add(y.Scale(math.Scalar(sin)))
	yp := x.Scale(math.Scalar(-sin)).Add(y.Scale(math.Scalar(cos)))
	rc := math.P2(math.Scalar(cos)*c.X-math.Scalar(sin)*c.Y, math.Scalar(sin)*c.X+math.Scalar(cos)*c.Y)
	origin := plane.ToModel(c).TranslateBy(x.Scale(rc.X).Negate()).TranslateBy(y.Scale(rc.Y).Negate())
	rotated, err := sketch.NewPlane(origin, xp.AsUnit(), yp.AsUnit())
	if err != nil {
		return plane // a degenerate frame keeps the punch un-rotated rather than failing the feature
	}
	return rotated
}

// profilesCentroid2 averages every profile's outer-loop vertices — the punch group's centre, the
// point the die turns about.
func profilesCentroid2(profiles []*sketch.Profile) math.Point2 {
	var sx, sy math.Scalar
	var n int
	for _, p := range profiles {
		for _, v := range p.OuterLoop().Polygon() {
			sx, sy, n = sx+v.X, sy+v.Y, n+1
		}
	}
	if n == 0 {
		return math.P2(0, 0)
	}
	return math.P2(sx/math.Scalar(n), sy/math.Scalar(n))
}

// punchSpan returns the punch's span: a fixed depth when Depth is set, else through all the
// running material on the chosen side of the sketch plane.
func (f *SheetMetalPunchFeature) punchSpan(bodies []*topo.Body, plane sketch.Plane) (span, error) {
	if f.def.Depth != nil {
		return distanceSpan(Extent{Type: DistanceExtent, Direction: f.def.Direction, Distance: f.def.Depth})
	}
	return throughAllSpan(Extent{Type: ThroughAllExtent, Direction: f.def.Direction}, bodies, plane)
}

// SheetMetalPunchFeatures adds punch features into the engine.
type SheetMetalPunchFeatures struct{ engine *PartFeatures }

// NewSheetMetalPunchFeatures binds the collection to a feature engine.
func NewSheetMetalPunchFeatures(engine *PartFeatures) *SheetMetalPunchFeatures {
	return &SheetMetalPunchFeatures{engine}
}

// Add appends a punch feature, naming it Punch1, Punch2, … .
func (c *SheetMetalPunchFeatures) Add(def *SheetMetalPunchDefinition) *PartFeature {
	f := &SheetMetalPunchFeature{def: def}
	pf := c.engine.Add(f)
	pf.SetName(c.engine.UniqueName("Punch"))
	f.featName = pf.Name()
	return pf
}

var _ Feature = (*SheetMetalPunchFeature)(nil)
