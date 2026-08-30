// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/math"
)

// Work-feature serialization — WORK POINTS (M48 #2238 split of serialize_work.go). The serialize/restore
// of a work point's definition (intersection/reference/explicit). The dispatch and shared reference
// encoding live in serialize_work.go.

func serializePointDef(def pointDefinition) (WorkFeatureData, error) {
	d := WorkFeatureData{Collection: "point", Kind: def.kindName(), Refs: refStrings(def.refs())}
	switch v := def.(type) {
	case positionPointDef:
		p := v.at()
		d.Position = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
	case *pointCloudPointDef:
		p := v.FrozenPosition() // last good model position; the source re-derives it after relink (#645)
		d.CloudID = v.cloudID
		d.Position = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
	case curveEntityPointDef:
		if v.proximity != nil { // record the solution-selection point so the side is stable (#1842)
			d.Solution = []float64{float64(v.proximity.X), float64(v.proximity.Y), float64(v.proximity.Z)}
		}
	case planeAxisPointDef, pointRefPointDef, twoLinesPointDef, threePlanesPointDef, faceCenterPointDef,
		edgeMidpointPointDef, centroidPointDef:
		// references only
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work point definition %q", def.kindName())
	}
	return d, nil
}

func restorePointFeature(c *WorkPoints, d WorkFeatureData) error {
	switch d.Kind {
	case "position":
		pos, err := point3From(d.Position, "position point")
		if err != nil {
			return err
		}
		c.AddByPosition(func() math.Point3 { return pos })
		return nil
	case "point-cloud-point":
		pos, err := point3From(d.Position, "point-cloud point")
		if err != nil {
			return err
		}
		// The live cloud source is re-attached after load (RelinkCloudPoints); until then the point
		// holds its frozen position (#645).
		c.addUser(&pointCloudPointDef{cloudID: d.CloudID, frozen: pos, hasPos: true})
		return nil
	case "plane-axis-intersection":
		r, err := workRefs(d.Refs, 2)
		if err != nil {
			return err
		}
		c.AddByPlaneAndAxisIntersection(r[0], r[1])
		return nil
	case "point":
		r, err := workRefs(d.Refs, 1)
		if err != nil {
			return err
		}
		c.AddByPoint(r[0])
		return nil
	case "two-lines":
		r, err := workRefs(d.Refs, 2)
		if err != nil {
			return err
		}
		c.AddByTwoLines(r[0], r[1])
		return nil
	case "three-planes":
		r, err := workRefs(d.Refs, 3)
		if err != nil {
			return err
		}
		c.AddByThreePlanes(r[0], r[1], r[2])
		return nil
	case "face-center":
		r, err := workRefs(d.Refs, 1)
		if err != nil {
			return err
		}
		c.AddByFaceCenter(r[0])
		return nil
	case "edge-midpoint":
		r, err := workRefs(d.Refs, 1)
		if err != nil {
			return err
		}
		c.AddByMidpointOfEdge(r[0])
		return nil
	case "curve-and-entity":
		r, err := workRefs(d.Refs, 2)
		if err != nil {
			return err
		}
		var prox *math.Point3
		if p, ok := solutionPoint(d); ok {
			prox = &p
		}
		c.AddByCurveAndEntity(r[0], r[1], prox)
		return nil
	case "centroid":
		if len(d.Refs) == 0 {
			return fmt.Errorf("centroid work point: no edge references")
		}
		c.AddAtCentroid(allWorkRefs(d.Refs)...)
		return nil
	default:
		return fmt.Errorf("no restore codec for work point kind %q", d.Kind)
	}
}
