// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
)

// Persisting a hole's placement rule (Oblikovati#1861). The placement is the whole reason the hole
// tracks its driving geometry, so a recipe that dropped it would reopen the part with the holes
// frozen wherever they last landed — the exact loss the placements were added to stop. What is
// written is the RULE (which sketch, which edges, what offsets), never the resolved coordinates.

// HolePlacementData is a hole placement's recipe. Kind selects which of the fields matter; the
// placement face itself is not repeated here, since it is already [HoleData.Face]/GeomFace.
type HolePlacementData struct {
	Kind    string  `yaml:"kind"` // sketch|linear|concentric|point
	Sketch  int     `yaml:"sketch,omitempty"`
	Flipped bool    `yaml:"flipped,omitempty"`
	RefEdge string  `yaml:"refEdge,omitempty"` // concentric: the circular edge's lineage key
	Edge1   string  `yaml:"edge1,omitempty"`   // linear: the two reference edges…
	Edge2   string  `yaml:"edge2,omitempty"`
	Offset1 float64 `yaml:"offset1,omitempty"` // …and the offsets measured from them
	Offset2 float64 `yaml:"offset2,omitempty"`
	Point   string  `yaml:"point,omitempty"` // on-point: the work point's WorkRef…
	Axis    string  `yaml:"axis,omitempty"`  // …and the axis it drills along
}

// serializeHolePlacement writes the placement rule, or nil when the hole uses the plain face
// placement (which [HoleData]'s own Face/Center fields already describe).
func serializeHolePlacement(p HolePlacement, sk SketchIndexer) (*HolePlacementData, error) {
	switch v := p.(type) {
	case nil:
		return nil, nil
	case SketchHolePlacement:
		idx, ok := sk.IndexOf(v.Sketch)
		if !ok {
			return nil, fmt.Errorf("hole: sketch placement references a sketch that is not in the part")
		}
		return &HolePlacementData{Kind: v.Kind(), Sketch: idx + 1, Flipped: v.Flipped}, nil
	case ConcentricHolePlacement:
		return &HolePlacementData{Kind: v.Kind(), RefEdge: encodeKey(v.RefEdge)}, nil
	case LinearHolePlacement:
		return &HolePlacementData{Kind: v.Kind(), Edge1: encodeKey(v.Edge1), Edge2: encodeKey(v.Edge2),
			Offset1: evalFloat(v.Offset1), Offset2: evalFloat(v.Offset2)}, nil
	case PointHolePlacement:
		return &HolePlacementData{Kind: v.Kind(), Point: pointRefOf(v.Point), Axis: axisRefOf(v.Axis), Flipped: v.Flipped}, nil
	default:
		return nil, fmt.Errorf("hole: unknown placement kind %q", p.Kind())
	}
}

// restoreHolePlacement rebuilds the rule against the reopened part. face is the hole's own
// placement face, which the face-anchored placements measure on.
func restoreHolePlacement(d *HolePlacementData, face HoleFaceRef, sk SketchIndexer,
	work *WorkGeometry) (HolePlacement, error) {
	if d == nil {
		return nil, nil
	}
	switch d.Kind {
	case "sketch":
		return restoreSketchPlacement(d, sk)
	case "concentric":
		key, err := decodeKey(d.RefEdge)
		return ConcentricHolePlacement{Face: face, RefEdge: key}, err
	case "linear":
		return restoreLinearPlacement(d, face)
	case "point":
		return restorePointPlacement(d, work)
	default:
		return nil, fmt.Errorf("hole: unknown placement kind %q (want sketch, linear, concentric or point)", d.Kind)
	}
}

// restoreSketchPlacement re-binds the placement to the reopened sketch by its 1-based index.
func restoreSketchPlacement(d *HolePlacementData, sk SketchIndexer) (HolePlacement, error) {
	s, ok := sk.At(d.Sketch - 1)
	if !ok {
		return nil, fmt.Errorf("hole: sketch placement references sketch index %d, which does not exist", d.Sketch-1)
	}
	return SketchHolePlacement{Sketch: s, Flipped: d.Flipped}, nil
}

// restoreLinearPlacement re-binds the two reference edges and their offsets.
func restoreLinearPlacement(d *HolePlacementData, face HoleFaceRef) (HolePlacement, error) {
	e1, err := decodeKey(d.Edge1)
	if err != nil {
		return nil, err
	}
	e2, err := decodeKey(d.Edge2)
	if err != nil {
		return nil, err
	}
	return LinearHolePlacement{Face: face, Edge1: e1, Edge2: e2,
		Offset1: constFloat(d.Offset1), Offset2: constFloat(d.Offset2)}, nil
}

// restorePointPlacement re-binds the work point and axis the bore is drilled on.
func restorePointPlacement(d *HolePlacementData, work *WorkGeometry) (HolePlacement, error) {
	if work == nil {
		return nil, fmt.Errorf("hole: an on-point placement needs the part's work geometry to resolve %q", d.Point)
	}
	point, ok := work.WorkPointByRef(WorkRef(d.Point))
	if !ok {
		return nil, fmt.Errorf("hole: on-point placement references work point %q, which does not exist", d.Point)
	}
	axis, ok := work.AxisByRef(WorkRef(d.Axis))
	if !ok {
		return nil, fmt.Errorf("hole: on-point placement references work axis %q, which does not exist", d.Axis)
	}
	return PointHolePlacement{Point: point, Axis: axis, Flipped: d.Flipped}, nil
}

// pointRefOf and axisRefOf are the recipe spellings of an optional work point / axis. They take the
// CONCRETE pointer rather than a Key()-bearing interface, because a typed nil pointer stored in an
// interface is not itself nil — the check would pass and the Key() call would panic.
func pointRefOf(p *WorkPoint) string {
	if p == nil {
		return ""
	}
	return string(p.Key())
}

func axisRefOf(a *WorkAxis) string {
	if a == nil {
		return ""
	}
	return string(a.Key())
}
