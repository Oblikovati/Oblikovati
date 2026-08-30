// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/model/sketch"
)

// Revolve-feature serialization (M48 #2238 split of serialize_work.go). The YAML shape and serialize/
// restore of a revolve feature — profile sketch, axis (a sketch line or a work axis / the profile's own
// axis), extent, direction and second angle. Shares the work-geometry reference helpers in serialize_work.go.

// RevolveData is a revolve's recipe: the sketch profile, the revolution axis (a
// WorkRef, typically an origin axis or a user work axis), the swept angle, and the
// boolean operation. Generation is still deferred, but the definition round-trips.
type RevolveData struct {
	Sketch     int     `yaml:"sketch"`
	Profile    int     `yaml:"profile"`
	Axis       string  `yaml:"axis,omitempty"`       // a work-axis WorkRef; empty ⇒ centerline mode
	AxisSketch int     `yaml:"axisSketch,omitempty"` // 1-based sketch index of a specific centerline (0 = none)
	AxisLine   int     `yaml:"axisLine,omitempty"`   // that centerline's line index
	Angle      float64 `yaml:"angle,omitempty"`      // 0 ⇒ full revolution
	Angle2     float64 `yaml:"angle2,omitempty"`     // second-direction sweep (#313)
	Direction  string  `yaml:"direction,omitempty"`  // which way Angle sweeps (#2019); empty ⇒ positive
	Operation  string  `yaml:"operation"`
	// Extent is how the revolve terminates (#1860): empty ⇒ the angle extent (the default, so an
	// ordinary revolve's recipe is unchanged), else "to-face", "from-to" or "to-next". ToPlane and
	// FromPlane are the terminators' WorkRefs, named exactly as an extrude's are.
	Extent    string `yaml:"extent,omitempty"`
	ToPlane   string `yaml:"toPlane,omitempty"`
	FromPlane string `yaml:"fromPlane,omitempty"`
	// ProfilePoint selects the revolved region by an interior seed point (sketch 2-D cm) rather
	// than by index, since an external author cannot predict the reader's region ordering. When
	// present it wins over Profile; absent, the index is used unchanged.
	ProfilePoint []float64 `yaml:"profilePoint,omitempty"`
}

func serializeRevolve(def *RevolveDefinition, sk SketchIndexer) (*RevolveData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("revolve references a sketch that is not in the part")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	d := &RevolveData{
		Sketch: idx, Profile: def.ProfileIndex, Angle: evalFloat(def.Angle), Angle2: evalFloat(def.Angle2),
		Direction: directionNames[def.Direction], Operation: op,
		Extent: revolveExtentName(def.Extent), ToPlane: planeRefOf(def.ToPlane), FromPlane: planeRefOf(def.FromPlane),
	}
	switch {
	case def.Axis != nil:
		d.Axis = string(def.Axis.Key())
	case def.AxisCenterline != nil: // a specific centerline (1-based so 0 means "none")
		asi, ok := sk.IndexOf(def.AxisCenterlineSketch)
		if !ok {
			return nil, fmt.Errorf("revolve axis centerline references a sketch not in the part")
		}
		d.AxisSketch, d.AxisLine = asi+1, lineIndexOf(def.AxisCenterlineSketch, def.AxisCenterline)
	}
	return d, nil // both empty ⇒ revolve about the profile sketch's own centerline
}

// lineIndexOf returns the position of line within the sketch's line collection (-1 if absent).
func lineIndexOf(sk *sketch.Sketch, line *sketch.Line) int {
	for i := 0; i < sk.Lines().Count(); i++ {
		if sk.Lines().Item(i) == line {
			return i
		}
	}
	return -1
}

func restoreRevolve(fs *PartFeatures, d *RevolveData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("revolve feature is missing its payload")
	}
	pf, err := restoreRevolveAboutItsAxis(fs, d, sk, work)
	if err != nil {
		return nil, err
	}
	// The sweep (second angle, direction, extent) and the region seed are axis-independent, so they
	// are applied once here rather than repeated down every axis branch.
	if err := restoreRevolveExtent(pf, d, work); err != nil {
		return nil, err
	}
	return withProfileSeed(restoreRevolveDirection(restoreSecondAngle(pf, d.Angle2), d.Direction), d.ProfilePoint), nil
}

// restoreRevolveExtent puts a persisted geometric termination back on a restored revolve (#1860).
// An empty recipe extent is the angle extent, which needs no terminator.
func restoreRevolveExtent(pf *PartFeature, d *RevolveData, work *WorkGeometry) error {
	rf, ok := pf.feature.(*RevolveFeature)
	if !ok || d.Extent == "" {
		return nil
	}
	rf.def.Extent = parseExtentName(d.Extent)
	var err error
	if rf.def.ToPlane, err = restorePlaneRef(work, d.ToPlane); err != nil {
		return err
	}
	rf.def.FromPlane, err = restorePlaneRef(work, d.FromPlane)
	return err
}

// restorePlaneRef resolves an optional terminator reference; an empty one is simply absent.
func restorePlaneRef(work *WorkGeometry, ref string) (*WorkPlane, error) {
	if ref == "" {
		return nil, nil
	}
	return resolvePlaneRef(work, ref)
}

// planeRefOf is the recipe spelling of an optional terminator plane.
func planeRefOf(wp *WorkPlane) string {
	if wp == nil {
		return ""
	}
	return string(wp.Key())
}

// revolveExtentName is the recipe spelling of a revolve's termination. The angle extent is the
// default and writes nothing, so an ordinary revolve's recipe is unchanged by #1860.
func revolveExtentName(t ExtentType) string {
	if t == DistanceExtent {
		return ""
	}
	return extentNames[t]
}

// restoreRevolveAboutItsAxis rebuilds the revolve on whichever axis the recipe names: a specific
// centerline, the profile sketch's own centerline, or a work axis.
func restoreRevolveAboutItsAxis(fs *PartFeatures, d *RevolveData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("revolve references sketch index %d, which does not exist", d.Sketch)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	angle := d.Angle
	// resolveSeed gives the fallback index; the seed is ALSO kept on the definition
	// (withProfileSeed) so it re-resolves against the current regions every recompute — the
	// sketch re-solves between load and recompute and reorders its regions (#region-seed).
	profile := resolveSeed(skt, d.ProfilePoint, d.Profile)
	if d.AxisSketch > 0 { // a specific centerline (1-based index)
		line, err := restoredAxisCenterline(d, sk)
		if err != nil {
			return nil, err
		}
		return NewRevolveFeatures(fs).AddAboutCenterlineLine(skt, profile, line.sketch, line.line, func() float64 { return angle }, op), nil
	}
	if d.Axis == "" { // revolve about the profile sketch's own (single) centerline
		return NewRevolveFeatures(fs).AddAboutCenterline(skt, profile, func() float64 { return angle }, op), nil
	}
	if work == nil {
		return nil, fmt.Errorf("revolve needs the part's work geometry to resolve its axis")
	}
	axis, err := work.axis(WorkRef(d.Axis))
	if err != nil {
		return nil, fmt.Errorf("revolve axis: %w", err)
	}
	return NewRevolveFeatures(fs).Add(skt, profile, axis, func() float64 { return angle }, op), nil
}

// restoredCenterline is a revolve axis recovered from its 1-based sketch index and line index.
type restoredCenterline struct {
	sketch *sketch.Sketch
	line   *sketch.Line
}

// restoredAxisCenterline resolves the recipe's axis centerline, reporting the offending index and
// the range it had to fall in when it does not.
func restoredAxisCenterline(d *RevolveData, sk SketchIndexer) (restoredCenterline, error) {
	axisSk, ok := sk.At(d.AxisSketch - 1)
	if !ok {
		return restoredCenterline{}, fmt.Errorf("revolve axis centerline references sketch index %d, which does not exist", d.AxisSketch-1)
	}
	if d.AxisLine < 0 || d.AxisLine >= axisSk.Lines().Count() {
		return restoredCenterline{}, fmt.Errorf("revolve axis centerline references line %d, outside the sketch's %d lines", d.AxisLine, axisSk.Lines().Count())
	}
	return restoredCenterline{sketch: axisSk, line: axisSk.Lines().Item(d.AxisLine)}, nil
}

// withProfileSeed records the interior seed point on a restored revolve so its region is
// re-resolved by containment each recompute (survives the sketch re-solving — #region-seed).
func withProfileSeed(pf *PartFeature, seed []float64) *PartFeature {
	if len(seed) > 0 {
		if rf, ok := pf.feature.(*RevolveFeature); ok {
			rf.def.ProfileSeed = append([]float64(nil), seed...)
		}
	}
	return pf
}

// restoreSecondAngle re-applies a persisted second-direction sweep onto a
// freshly restored revolve (the centerline Add paths take one angle).
func restoreSecondAngle(pf *PartFeature, angle2 float64) *PartFeature {
	if angle2 <= 0 {
		return pf
	}
	if rf, ok := pf.feature.(*RevolveFeature); ok {
		rf.def.Angle2 = func() float64 { return angle2 }
	}
	return pf
}

// restoreRevolveDirection puts the saved sweep direction back on a restored revolve (#2019). It is
// applied on EVERY axis path — a flipped revolve about a centerline is as real as one about a work
// axis — so it sits outside the axis branches rather than inside one of them.
func restoreRevolveDirection(pf *PartFeature, name string) *PartFeature {
	if rf, ok := pf.feature.(*RevolveFeature); ok {
		rf.def.Direction = parseDirectionName(name)
	}
	return pf
}
