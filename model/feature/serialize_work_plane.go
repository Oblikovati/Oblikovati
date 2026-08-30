// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/math"
)

// Work-feature serialization — WORK PLANES (M48 #2238 split of serialize_work.go). The YAML shape and
// serialize/restore of a work plane's definition — reference-driven (offset/angle/tangent/…), a fixed
// frame, or a point-cloud best-fit. The dispatch and shared reference encoding live in serialize_work.go.

func serializePlaneDef(def planeDefinition) (WorkFeatureData, error) {
	d := WorkFeatureData{Collection: "plane", Kind: def.kindName(), Refs: refStrings(def.refs())}
	switch v := def.(type) {
	case *offsetPlaneDef:
		d.Offset = v.distance() // persist the effective distance, including any browser edit
	case *fixedFramePlaneDef:
		p := v.origin()
		d.Position = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
		d.XAxis, d.YAxis = unitSlice(v.x), unitSlice(v.y)
	case *pointCloudFitPlaneDef:
		// Persist the provenance link and the last good fit (the frozen frame), so the plane has
		// geometry on load even before the cloud source is re-attached (#645).
		d.CloudID = v.cloudID
		d.Position = []float64{float64(v.origin.X), float64(v.origin.Y), float64(v.origin.Z)}
		d.XAxis, d.YAxis = unitSlice(v.x), unitSlice(v.y)
	case *linePlaneAnglePlaneDef:
		d.Angle = v.angle()
	case *twoPlanesPlaneDef:
		if v.quadrant != nil { // persist the chosen bisector quadrant (#1844)
			d.Solution = p3Slice(*v.quadrant)
		}
	case *planeAndTangentPlaneDef:
		if v.proximity != nil { // persist the chosen tangent side (#1844)
			d.Solution = p3Slice(*v.proximity)
		}
	case *lineAndTangentPlaneDef:
		if v.proximity != nil {
			d.Solution = p3Slice(*v.proximity)
		}
	case *threePointPlaneDef, *planeAndPointPlaneDef, *twoLinesPlaneDef,
		*lineAndPointPlaneDef, *normalToCurvePlaneDef, *torusMidPlaneDef, *pointAndTangentPlaneDef:
		// references only
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work plane definition %q", def.kindName())
	}
	return d, nil
}

// restorePlaneFeature rebuilds one user work plane from its recipe. Reference-only kinds
// resolve through workRefs; fixed-frame/offset/angle kinds also carry scalar parameters,
// re-installed as closures so a recompute re-reads them.
func restorePlaneFeature(c *WorkPlanes, d WorkFeatureData) error {
	switch d.Kind {
	case "plane-offset":
		return restoreRefPlane(d, 1, func(r []WorkRef) {
			off := d.Offset
			c.AddByPlaneAndOffset(r[0], func() float64 { return off })
		})
	case "three-points":
		return restoreRefPlane(d, 3, func(r []WorkRef) { c.AddByThreePoints(r[0], r[1], r[2]) })
	case "fixed-frame":
		return restoreFixedFrame(c, d)
	case "point-cloud-fit":
		return restorePointCloudFit(c, d)
	case "plane-point":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByPlaneAndPoint(r[0], r[1]) })
	case "two-planes":
		return restoreRefPlane(d, 2, func(r []WorkRef) {
			if pt, ok := solutionPoint(d); ok {
				c.AddByTwoPlanesToward(r[0], r[1], pt)
				return
			}
			c.AddByTwoPlanes(r[0], r[1])
		})
	case "line-plane-angle":
		return restoreRefPlane(d, 2, func(r []WorkRef) {
			ang := d.Angle
			c.AddByLinePlaneAndAngle(r[0], r[1], func() float64 { return ang })
		})
	case "two-lines":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByTwoLines(r[0], r[1]) })
	case "line-point":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByLineAndPoint(r[0], r[1]) })
	case "normal-to-curve":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByNormalToCurve(r[0], r[1]) })
	case "torus-midplane":
		return restoreRefPlane(d, 1, func(r []WorkRef) { c.AddByTorusMidPlane(r[0]) })
	case "point-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByPointAndTangent(r[0], r[1]) })
	case "plane-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) {
			if pt, ok := solutionPoint(d); ok {
				c.AddByPlaneAndTangentToward(r[0], r[1], pt)
				return
			}
			c.AddByPlaneAndTangent(r[0], r[1])
		})
	case "line-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) {
			if pt, ok := solutionPoint(d); ok {
				c.AddByLineAndTangentToward(r[0], r[1], pt)
				return
			}
			c.AddByLineAndTangent(r[0], r[1])
		})
	default:
		return fmt.Errorf("no restore codec for work plane kind %q", d.Kind)
	}
}

// restoreRefPlane resolves d's n references and calls add with them, centralizing the
// arity check so each plane kind above stays a single line.
func restoreRefPlane(d WorkFeatureData, n int, add func([]WorkRef)) error {
	r, err := workRefs(d.Refs, n)
	if err != nil {
		return err
	}
	add(r)
	return nil
}

// restoreFixedFrame rebuilds an AddFixed plane from its origin and two in-plane axes.
func restoreFixedFrame(c *WorkPlanes, d WorkFeatureData) error {
	origin, err := point3From(d.Position, "fixed-frame origin")
	if err != nil {
		return err
	}
	x, err := unit3From(d.XAxis, "fixed-frame X axis")
	if err != nil {
		return err
	}
	y, err := unit3From(d.YAxis, "fixed-frame Y axis")
	if err != nil {
		return err
	}
	c.AddFixed(func() math.Point3 { return origin }, x, y)
	return nil
}

// restorePointCloudFit rebuilds an associative point-cloud-fit plane from its provenance id and
// last good fit (the frozen frame). The live cloud source is re-attached after load by the host
// (RelinkCloudFits), so until then the plane holds its frozen geometry (#645).
func restorePointCloudFit(c *WorkPlanes, d WorkFeatureData) error {
	origin, err := point3From(d.Position, "point-cloud-fit origin")
	if err != nil {
		return err
	}
	x, err := unit3From(d.XAxis, "point-cloud-fit X axis")
	if err != nil {
		return err
	}
	y, err := unit3From(d.YAxis, "point-cloud-fit Y axis")
	if err != nil {
		return err
	}
	c.addUser(&pointCloudFitPlaneDef{cloudID: d.CloudID, origin: origin, x: x, y: y, hasFit: true})
	return nil
}
