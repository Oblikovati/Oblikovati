// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Work-feature serialization — WORK AXES (M48 #2238 split of serialize_work.go). The serialize/restore of
// a work axis's definition — reference-driven or an explicit line. The dispatch and shared reference
// encoding live in serialize_work.go.

func serializeAxisDef(def axisDefinition) (WorkFeatureData, error) {
	switch v := def.(type) {
	case fixedAxisDef: // grounded "line" axis: persist its origin + direction (no references)
		p := v.origin
		return WorkFeatureData{
			Collection: "axis", Kind: def.kindName(),
			Position: []float64{float64(p.X), float64(p.Y), float64(p.Z)}, XAxis: unitSlice(v.dir),
		}, nil
	case twoPointsAxisDef, planeIntersectionAxisDef,
		pointAndPlaneAxisDef, lineAndPointAxisDef, lineAndPlaneAxisDef, revolvedFaceAxisDef, edgeAxisDef:
		return WorkFeatureData{Collection: "axis", Kind: def.kindName(), Refs: refStrings(def.refs())}, nil
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work axis definition %q", def.kindName())
	}
}

func restoreAxisFeature(c *WorkAxes, d WorkFeatureData) error {
	if d.Kind == "line" { // grounded axis: rebuilt from its origin + direction, no references
		return restoreLineAxis(c, d)
	}
	switch d.Kind { // single-reference axis kinds (a face or an edge) — #1840
	case "revolved-face", "analytic-edge", "line-by-entity":
		r, err := workRefs(d.Refs, 1)
		if err != nil {
			return err
		}
		switch d.Kind {
		case "revolved-face":
			c.AddByRevolvedFace(r[0])
		case "analytic-edge":
			c.AddByAnalyticEdge(r[0])
		case "line-by-entity":
			c.AddByLineByEntity(r[0])
		}
		return nil
	}
	r, err := workRefs(d.Refs, 2)
	if err != nil {
		return err
	}
	switch d.Kind {
	case "two-points":
		c.AddByTwoPoints(r[0], r[1])
	case "plane-intersection":
		c.AddByPlaneIntersection(r[0], r[1])
	case "point-and-plane":
		c.AddByPointAndPlane(r[0], r[1])
	case "line-and-point":
		c.AddByLineAndPoint(r[0], r[1])
	case "line-and-plane":
		c.AddByLineAndPlane(r[0], r[1])
	default:
		return fmt.Errorf("no restore codec for work axis kind %q", d.Kind)
	}
	return nil
}

// restoreLineAxis rebuilds a grounded "line" axis from its persisted origin + direction.
func restoreLineAxis(c *WorkAxes, d WorkFeatureData) error {
	o, err := point3From(d.Position, "line axis origin")
	if err != nil {
		return err
	}
	dir, err := unit3From(d.XAxis, "line axis direction")
	if err != nil {
		return err
	}
	c.AddByLine(o, dir)
	return nil
}
