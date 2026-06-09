// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/model/sketch"

// SketchConsumer is implemented by feature definitions that consume one or more sketches
// as profile/path/centerline input. The model browser uses it to nest a consumed sketch
// UNDER the feature that uses it — Inventor's chronological model tree — instead of listing
// every sketch in a separate branch (issue #132). Implementations return every distinct
// non-nil sketch the feature depends on; a feature that consumes no sketch (a dress-up, a
// boolean, a work feature) simply does not implement this interface.
//
// Example: an extrude returns its single profile sketch, so the browser draws that sketch as
// a child of "Extrude1" rather than at the top level.
type SketchConsumer interface {
	ConsumedSketches() []*sketch.Sketch
}

// ConsumedSketches returns the sketches this feature consumes, or nil when it consumes none.
// It forwards to the wrapped feature when that feature is a [SketchConsumer], so callers work
// uniformly across every feature kind.
func (f *PartFeature) ConsumedSketches() []*sketch.Sketch {
	if c, ok := f.feature.(SketchConsumer); ok {
		return c.ConsumedSketches()
	}
	return nil
}

// nonNilSketches collects the non-nil sketches from the arguments, de-duplicating by
// pointer so a feature that names the same sketch twice (e.g. a revolve whose profile and
// centerline live in one sketch) reports it once.
func nonNilSketches(sks ...*sketch.Sketch) []*sketch.Sketch {
	var out []*sketch.Sketch
	for _, sk := range sks {
		if sk == nil {
			continue
		}
		if !containsSketch(out, sk) {
			out = append(out, sk)
		}
	}
	return out
}

func containsSketch(in []*sketch.Sketch, want *sketch.Sketch) bool {
	for _, sk := range in {
		if sk == want {
			return true
		}
	}
	return false
}

// Compile-time proof that every sketch-consuming feature reports its sketches, so the
// browser's nesting stays complete as feature kinds evolve.
var (
	_ SketchConsumer = (*ExtrudeFeature)(nil)
	_ SketchConsumer = (*EmbossFeature)(nil)
	_ SketchConsumer = (*RevolveFeature)(nil)
	_ SketchConsumer = (*CoilFeature)(nil)
	_ SketchConsumer = (*RibFeature)(nil)
	_ SketchConsumer = (*SweepFeature)(nil)
	_ SketchConsumer = (*LoftFeature)(nil)
	_ SketchConsumer = (*BoundaryPatchFeature)(nil)
	_ SketchConsumer = (*RuledSurfaceFeature)(nil)
)

// --- per-feature consumed-sketch reporting (the sketches that feed each feature's geometry) ---

func (e *ExtrudeFeature) ConsumedSketches() []*sketch.Sketch { return nonNilSketches(e.def.Sketch) }
func (f *EmbossFeature) ConsumedSketches() []*sketch.Sketch  { return nonNilSketches(f.def.Sketch) }
func (c *CoilFeature) ConsumedSketches() []*sketch.Sketch    { return nonNilSketches(c.def.Sketch) }
func (r *RibFeature) ConsumedSketches() []*sketch.Sketch     { return nonNilSketches(r.def.Sketch) }
func (s *SweepFeature) ConsumedSketches() []*sketch.Sketch   { return nonNilSketches(s.def.Sketch) }

func (r *RevolveFeature) ConsumedSketches() []*sketch.Sketch {
	return nonNilSketches(r.def.Sketch, r.def.AxisCenterlineSketch)
}

func (r *RuledSurfaceFeature) ConsumedSketches() []*sketch.Sketch {
	return nonNilSketches(r.def.Sketch)
}

func (l *LoftFeature) ConsumedSketches() []*sketch.Sketch {
	sks := make([]*sketch.Sketch, 0, len(l.def.Sections))
	for _, s := range l.def.Sections {
		sks = append(sks, s.Sketch) // nil for point/face sections — nonNilSketches drops them
	}
	return nonNilSketches(sks...)
}

func (b *BoundaryPatchFeature) ConsumedSketches() []*sketch.Sketch {
	loops := b.def.Loops
	if loops == nil {
		return nil
	}
	sks := make([]*sketch.Sketch, 0, loops.Count())
	for i := 0; i < loops.Count(); i++ {
		sks = append(sks, loops.Item(i).Sketch)
	}
	return nonNilSketches(sks...)
}
