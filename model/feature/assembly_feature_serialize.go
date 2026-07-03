// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/model/sketch"
)

// Serialization of the assembly feature program (#785): each machining feature renders its inputs
// to an [AssemblyFeatureData] (a tagged union over Kind) and is reconstructed from one, so an
// assembly's features round-trip through save/load and undo/redo. Sketch-bearing features
// (extrude/revolve/sweep) reference the assembly's sketch by index; the closure-backed scalars
// (distance/radius/angle) capture their current value and restore as a constant — editing a
// restored feature is a separate concern. The box-cut persists its tool's axis-aligned corners
// (every tool is a brep.SolidBlock) and the proxy-cut persists its source occurrence by name,
// rebound through occByName once the occurrences are bound.

// Assembly feature kind discriminators: the canonical identifier each feature reports from Kind()
// and persists as [AssemblyFeatureData.Kind] (#785). Defined once so the Kind() method, the restore
// switch, and MarshalAssembly cannot drift apart.
const (
	kindAssemblyExtrude  = "assemblyExtrude"
	kindAssemblyRevolve  = "assemblyRevolve"
	kindAssemblySweep    = "assemblySweep"
	kindAssemblyHole     = "assemblyHole"
	kindAssemblyChamfer  = "assemblyChamfer"
	kindAssemblyFillet   = "assemblyFillet"
	kindAssemblyMoveFace = "assemblyMoveFace"
	kindAssemblyCut      = "assemblyCut"
	kindAssemblyProxyCut = "assemblyProxyCut"
)

// AssemblyFeatureData is the serializable form of one assembly machining feature: its kind and the
// superset of inputs the kinds use (each populates only its own).
type AssemblyFeatureData struct {
	Kind         string       `yaml:"kind"`
	SketchIndex  int          `yaml:"sketchIndex,omitempty"`  // extrude/revolve/sweep
	ProfileIndex int          `yaml:"profileIndex,omitempty"` // extrude/revolve/sweep
	Operation    int          `yaml:"operation,omitempty"`    // ops.PartFeatureOperation
	Distance     float64      `yaml:"distance,omitempty"`     // extrude / chamfer
	Radius       float64      `yaml:"radius,omitempty"`       // fillet
	Angle        float64      `yaml:"angle,omitempty"`        // revolve
	FlatCorners  bool         `yaml:"flatCorners,omitempty"`  // chamfer
	EdgeSuffixes [][]byte     `yaml:"edgeSuffixes,omitempty"` // chamfer / fillet
	FaceSuffixes [][]byte     `yaml:"faceSuffixes,omitempty"` // moveFace
	Translation  [3]float64   `yaml:"translation,omitempty"`  // moveFace
	Center       [3]float64   `yaml:"center,omitempty"`       // hole
	Axis         [3]float64   `yaml:"axis,omitempty"`         // hole / revolve direction
	AxisOrigin   [3]float64   `yaml:"axisOrigin,omitempty"`   // revolve datum-axis origin
	HasAxis      bool         `yaml:"hasAxis,omitempty"`      // revolve: an explicit axis (else the sketch centerline)
	Diameter     float64      `yaml:"diameter,omitempty"`     // hole
	Depth        float64      `yaml:"depth,omitempty"`        // hole
	Path         [][3]float64 `yaml:"path,omitempty"`         // sweep
	Twist        float64      `yaml:"twist,omitempty"`        // sweep total twist angle (rad)
	ToolMin      [3]float64   `yaml:"toolMin,omitempty"`      // box-cut tool corners
	ToolMax      [3]float64   `yaml:"toolMax,omitempty"`      // box-cut tool corners
	SourceName   string       `yaml:"sourceName,omitempty"`   // proxy-cut source occurrence name
}

// AssemblyFeatureMarshaler is implemented by the assembly features whose state is self-contained.
// sketchIndex maps the feature's sketch to its index in the assembly's sketch collection.
type AssemblyFeatureMarshaler interface {
	MarshalAssembly(sketchIndex func(*sketch.Sketch) int) AssemblyFeatureData
}

// RestoreAssemblyFeature reconstructs a feature from its data, resolving sketch references against
// sketches (by index) and a proxy-cut's source occurrence through occByName (nil for the kinds that
// hold no occurrence reference). It errors on an unknown kind, an out-of-range sketch index, or an
// unresolvable source.
func RestoreAssemblyFeature(d AssemblyFeatureData, sketches []*sketch.Sketch, occByName func(string) (*occurrence.Occurrence, bool)) (Feature, error) {
	switch d.Kind {
	case kindAssemblyExtrude:
		sk, err := sketchAt(sketches, d.SketchIndex, d.Kind)
		if err != nil {
			return nil, err
		}
		return NewAssemblyExtrudeFeature(sk, d.ProfileIndex, ops.PartFeatureOperation(d.Operation), constScalar(d.Distance)), nil
	case kindAssemblyRevolve:
		sk, err := sketchAt(sketches, d.SketchIndex, d.Kind)
		if err != nil {
			return nil, err
		}
		return NewAssemblyRevolveFeature(sk, d.ProfileIndex, restoreAxis(d), ops.PartFeatureOperation(d.Operation), constScalar(d.Angle)), nil
	case kindAssemblySweep:
		sk, err := sketchAt(sketches, d.SketchIndex, d.Kind)
		if err != nil {
			return nil, err
		}
		return NewAssemblySweepFeature(sk, d.ProfileIndex, ops.PartFeatureOperation(d.Operation), pointsFrom(d.Path), constScalar(d.Twist)), nil
	case kindAssemblyHole:
		axis, err := math.NewUnitVector3(d.Axis[0], d.Axis[1], d.Axis[2])
		if err != nil {
			return nil, fmt.Errorf("feature: restore hole: axis %v is not a direction: %w", d.Axis, err)
		}
		return NewAssemblyHoleFeature(math.P3(d.Center[0], d.Center[1], d.Center[2]), axis, d.Diameter, d.Depth)
	case kindAssemblyChamfer:
		ch := NewAssemblyChamferFeature(d.EdgeSuffixes, constScalar(d.Distance))
		ch.flatCorners = d.FlatCorners // #785: a flat-corner chamfer must restore flat, not reset to mitred
		return ch, nil
	case kindAssemblyFillet:
		return NewAssemblyFilletFeature(d.EdgeSuffixes, constScalar(d.Radius)), nil
	case kindAssemblyMoveFace:
		return NewAssemblyMoveFaceFeature(d.FaceSuffixes, math.V3(math.Scalar(d.Translation[0]), math.Scalar(d.Translation[1]), math.Scalar(d.Translation[2]))), nil
	case kindAssemblyCut:
		tool, err := brep.SolidBlock(math.P3(d.ToolMin[0], d.ToolMin[1], d.ToolMin[2]), math.P3(d.ToolMax[0], d.ToolMax[1], d.ToolMax[2]), "asmCut")
		if err != nil {
			return nil, fmt.Errorf("feature: restore box cut: %w", err)
		}
		return NewAssemblyCutFeature(tool, ops.PartFeatureOperation(d.Operation)), nil
	case kindAssemblyProxyCut:
		if occByName == nil {
			return nil, fmt.Errorf("feature: restore proxy cut %q needs an occurrence resolver", d.SourceName)
		}
		src, ok := occByName(d.SourceName)
		if !ok {
			return nil, fmt.Errorf("feature: restore proxy cut: source occurrence %q not found", d.SourceName)
		}
		return NewAssemblyProxyCutFeature(src, ops.PartFeatureOperation(d.Operation)), nil
	default:
		return nil, fmt.Errorf("feature: cannot restore assembly feature kind %q", d.Kind)
	}
}

// sketchAt resolves a sketch index, erroring when it is out of range (a corrupt recipe).
func sketchAt(sketches []*sketch.Sketch, i int, kind string) (*sketch.Sketch, error) {
	if i < 0 || i >= len(sketches) {
		return nil, fmt.Errorf("feature: restore %s: sketch index %d out of range (%d sketches)", kind, i, len(sketches))
	}
	return sketches[i], nil
}

// restoreAxis rebuilds a revolve's explicit datum axis from data, or nil when none was stored (the
// feature falls back to the sketch's single centerline).
func restoreAxis(d AssemblyFeatureData) *WorkAxis {
	if !d.HasAxis {
		return nil
	}
	dir, err := math.NewUnitVector3(d.Axis[0], d.Axis[1], d.Axis[2])
	if err != nil {
		return nil
	}
	return NewDatumAxis(math.P3(d.AxisOrigin[0], d.AxisOrigin[1], d.AxisOrigin[2]), dir)
}

// constScalar returns a closure yielding a fixed value — the restored form of a closure-backed
// scalar (editing a restored feature is out of scope here).
func constScalar(v float64) func() float64 { return func() float64 { return v } }

// pointsFrom / pathTo convert a sweep path between the flat-array recipe form and points.
func pointsFrom(path [][3]float64) []math.Point3 {
	out := make([]math.Point3, len(path))
	for i, p := range path {
		out[i] = math.P3(p[0], p[1], p[2])
	}
	return out
}

func pathTo(pts []math.Point3) [][3]float64 {
	out := make([][3]float64, len(pts))
	for i, p := range pts {
		out[i] = [3]float64{float64(p.X), float64(p.Y), float64(p.Z)}
	}
	return out
}

// --- per-kind MarshalAssembly ---------------------------------------------

func (f *AssemblyExtrudeFeature) MarshalAssembly(sketchIndex func(*sketch.Sketch) int) AssemblyFeatureData {
	return AssemblyFeatureData{Kind: kindAssemblyExtrude, SketchIndex: sketchIndex(f.sketch), ProfileIndex: f.profileIndex, Operation: int(f.op), Distance: f.distance()}
}

func (f *AssemblyRevolveFeature) MarshalAssembly(sketchIndex func(*sketch.Sketch) int) AssemblyFeatureData {
	d := AssemblyFeatureData{Kind: kindAssemblyRevolve, SketchIndex: sketchIndex(f.sketch), ProfileIndex: f.profileIndex, Operation: int(f.op), Angle: f.angle()}
	if f.axis != nil {
		d.HasAxis = true
		o, dir := f.axis.Origin(), f.axis.Direction().AsVector()
		d.AxisOrigin = [3]float64{float64(o.X), float64(o.Y), float64(o.Z)}
		d.Axis = [3]float64{float64(dir.X), float64(dir.Y), float64(dir.Z)}
	}
	return d
}

func (f *AssemblySweepFeature) MarshalAssembly(sketchIndex func(*sketch.Sketch) int) AssemblyFeatureData {
	return AssemblyFeatureData{Kind: kindAssemblySweep, SketchIndex: sketchIndex(f.sketch), ProfileIndex: f.profileIndex, Operation: int(f.op), Path: pathTo(f.path), Twist: callOrZero(f.twist)}
}

func (f *AssemblyHoleFeature) MarshalAssembly(func(*sketch.Sketch) int) AssemblyFeatureData {
	dir := f.axis.AsVector()
	return AssemblyFeatureData{Kind: kindAssemblyHole,
		Center:   [3]float64{float64(f.center.X), float64(f.center.Y), float64(f.center.Z)},
		Axis:     [3]float64{float64(dir.X), float64(dir.Y), float64(dir.Z)},
		Diameter: f.diameter, Depth: f.depth}
}

func (f *AssemblyChamferFeature) MarshalAssembly(func(*sketch.Sketch) int) AssemblyFeatureData {
	return AssemblyFeatureData{Kind: kindAssemblyChamfer, EdgeSuffixes: f.edgeSuffixes, Distance: f.distance(), FlatCorners: f.flatCorners}
}

func (f *AssemblyFilletFeature) MarshalAssembly(func(*sketch.Sketch) int) AssemblyFeatureData {
	return AssemblyFeatureData{Kind: kindAssemblyFillet, EdgeSuffixes: f.edgeSuffixes, Radius: f.radius()}
}

func (f *AssemblyMoveFaceFeature) MarshalAssembly(func(*sketch.Sketch) int) AssemblyFeatureData {
	return AssemblyFeatureData{Kind: kindAssemblyMoveFace, FaceSuffixes: f.faceSuffixes,
		Translation: [3]float64{float64(f.translation.X), float64(f.translation.Y), float64(f.translation.Z)}}
}

// MarshalAssembly captures the box-cut's tool as its axis-aligned corners (every tool is a
// brep.SolidBlock; its range box recovers the corners), rebuilt on restore.
func (f *AssemblyCutFeature) MarshalAssembly(func(*sketch.Sketch) int) AssemblyFeatureData {
	box := f.tool.RangeBox()
	return AssemblyFeatureData{Kind: kindAssemblyCut, Operation: int(f.op),
		ToolMin: [3]float64{float64(box.Min.X), float64(box.Min.Y), float64(box.Min.Z)},
		ToolMax: [3]float64{float64(box.Max.X), float64(box.Max.Y), float64(box.Max.Z)}}
}

// MarshalAssembly captures the proxy-cut's source occurrence by name; restore rebinds it to the
// live occurrence (the assembly resolves it after the occurrences are bound).
func (f *AssemblyProxyCutFeature) MarshalAssembly(func(*sketch.Sketch) int) AssemblyFeatureData {
	return AssemblyFeatureData{Kind: kindAssemblyProxyCut, Operation: int(f.op), SourceName: f.source.Name()}
}
