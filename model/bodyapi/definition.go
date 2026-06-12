// SPDX-License-Identifier: GPL-2.0-only

package bodyapi

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Decoding the wire definition graph (types.BrepBodyDefinition) into the
// kernel's brep.SurfaceBodyDefinition: tagged analytic curve/surface
// definitions become geom values; geometric problems are reported by graph
// path like the compiler's own issues.

// DecodeBodyDefinition maps the contract graph onto the kernel graph. A
// non-empty issue list rejects the graph (no kernel definition is returned).
func DecodeBodyDefinition(def types.BrepBodyDefinition) (brep.SurfaceBodyDefinition, []types.BrepDefinitionIssue) {
	var out brep.SurfaceBodyDefinition
	var issues []types.BrepDefinitionIssue
	out.Solid = def.Solid
	issues = decodeVertices(&out, def, issues)
	issues = decodeEdges(&out, def, issues)
	issues = decodeFaces(&out, def, issues)
	decodeLumps(&out, def)
	if len(issues) > 0 {
		return brep.SurfaceBodyDefinition{}, issues
	}
	return out, nil
}

func decodeVertices(out *brep.SurfaceBodyDefinition, def types.BrepBodyDefinition, issues []types.BrepDefinitionIssue) []types.BrepDefinitionIssue {
	for i, v := range def.Vertices {
		p, ok := triplet(v.Position)
		if !ok {
			issues = append(issues, defIssue("vertices", i, "position needs [x, y, z], got %v", v.Position))
			continue
		}
		out.Vertices = append(out.Vertices, brep.VertexDefinition{Position: p, AssociativeID: v.AssociativeID})
	}
	return issues
}

func decodeEdges(out *brep.SurfaceBodyDefinition, def types.BrepBodyDefinition, issues []types.BrepDefinitionIssue) []types.BrepDefinitionIssue {
	for i, e := range def.Edges {
		curve, err := decodeCurve(e.Curve)
		if err != nil {
			issues = append(issues, defIssue("edges", i, "%v", err))
			continue
		}
		out.Edges = append(out.Edges, brep.EdgeDefinition{
			Curve: curve, StartVertex: e.StartVertex, EndVertex: e.EndVertex, AssociativeID: e.AssociativeID,
		})
	}
	return issues
}

func decodeFaces(out *brep.SurfaceBodyDefinition, def types.BrepBodyDefinition, issues []types.BrepDefinitionIssue) []types.BrepDefinitionIssue {
	for i, f := range def.Faces {
		surf, err := decodeSurface(f.Surface)
		if err != nil {
			issues = append(issues, defIssue("faces", i, "%v", err))
			continue
		}
		kf := brep.FaceDefinition{Surface: surf, ParamReversed: f.ParamReversed, AssociativeID: f.AssociativeID}
		for _, l := range f.Loops {
			kl := brep.EdgeLoopDefinition{}
			for _, u := range l.Uses {
				kl.Uses = append(kl.Uses, brep.EdgeUseDefinition{Edge: u.Edge, Opposed: u.Opposed})
			}
			kf.Loops = append(kf.Loops, kl)
		}
		out.Faces = append(out.Faces, kf)
	}
	return issues
}

func decodeLumps(out *brep.SurfaceBodyDefinition, def types.BrepBodyDefinition) {
	for _, lump := range def.Lumps {
		kl := brep.LumpDefinition{}
		for _, sh := range lump.Shells {
			kl.Shells = append(kl.Shells, brep.FaceShellDefinition{Faces: sh.Faces})
		}
		for _, w := range lump.Wires {
			kl.Wires = append(kl.Wires, brep.WireDefinition{Edges: w.Edges})
		}
		out.Lumps = append(out.Lumps, kl)
	}
}

// decodeCurve builds the kernel curve from a tagged definition.
func decodeCurve(c types.BrepCurveDef) (geom.Curve3, error) {
	switch c.Kind {
	case "lineSegment":
		pts, ok := triplets(c.Points, 2)
		if !ok {
			return nil, fmt.Errorf("lineSegment needs 2 points (6 floats), got %d floats", len(c.Points))
		}
		return geom.NewLineSegment(pts[0], pts[1]), nil
	case "arc":
		return decodeArc(c)
	case "polyline":
		pts, ok := triplets(c.Points, -1)
		if !ok || len(pts) < 2 {
			return nil, fmt.Errorf("polyline needs 2+ xyz triplets, got %d floats", len(c.Points))
		}
		return geom.NewPolyline(pts)
	default:
		return nil, fmt.Errorf("unknown curve kind %q (want lineSegment, arc or polyline)", c.Kind)
	}
}

func decodeArc(c types.BrepCurveDef) (geom.Curve3, error) {
	center, okC := triplet(c.Center)
	normal, okN := tripletV(c.Normal)
	refDir, okR := tripletV(c.RefDir)
	if !okC || !okN || !okR {
		return nil, fmt.Errorf("arc needs center, normal and refDir as [x, y, z]")
	}
	return geom.NewArc3d(center, normal, refDir, c.Radius, c.StartAngle, c.SweepAngle)
}

// decodeSurface builds the kernel surface from a tagged definition.
func decodeSurface(s types.BrepSurfaceDef) (geom.Surface, error) {
	switch s.Kind {
	case "plane":
		return decodePlane(s)
	case "cylinder":
		return decodeAxial(s, func(o math.Point3, a math.Vector3) (geom.Surface, error) {
			return geom.NewCylinder(o, a, s.Radius)
		})
	case "cone":
		return decodeAxial(s, func(o math.Point3, a math.Vector3) (geom.Surface, error) {
			return geom.NewCone(o, a, s.HalfAngle)
		})
	case "sphere":
		o, ok := triplet(s.Origin)
		if !ok {
			return nil, fmt.Errorf("sphere needs origin as [x, y, z]")
		}
		return geom.NewSphere(o, s.Radius)
	case "torus":
		return decodeAxial(s, func(o math.Point3, a math.Vector3) (geom.Surface, error) {
			return geom.NewTorus(o, a, s.MajorRadius, s.MinorRadius)
		})
	default:
		return nil, fmt.Errorf("unknown surface kind %q (want plane, cylinder, cone, sphere or torus)", s.Kind)
	}
}

func decodePlane(s types.BrepSurfaceDef) (geom.Surface, error) {
	o, okO := triplet(s.Origin)
	n, okN := tripletV(s.Normal)
	if !okO || !okN {
		return nil, fmt.Errorf("plane needs origin and normal as [x, y, z]")
	}
	return geom.NewPlane(o, n)
}

func decodeAxial(s types.BrepSurfaceDef, build func(math.Point3, math.Vector3) (geom.Surface, error)) (geom.Surface, error) {
	o, okO := triplet(s.Origin)
	a, okA := tripletV(s.Axis)
	if !okO || !okA {
		return nil, fmt.Errorf("%s needs origin and axis as [x, y, z]", s.Kind)
	}
	return build(o, a)
}

// definitionIssues maps kernel compiler issues onto the contract form.
func definitionIssues(in []brep.DefinitionIssue) []types.BrepDefinitionIssue {
	out := make([]types.BrepDefinitionIssue, len(in))
	for i, di := range in {
		out[i] = types.BrepDefinitionIssue{Path: di.Path, Problem: di.Problem}
	}
	return out
}

func defIssue(section string, index int, format string, args ...interface{}) types.BrepDefinitionIssue {
	return types.BrepDefinitionIssue{
		Path:    fmt.Sprintf("%s[%d]", section, index),
		Problem: fmt.Sprintf(format, args...),
	}
}

// triplet decodes one [x, y, z] point; tripletV one vector.
func triplet(v []float64) (math.Point3, bool) {
	if len(v) != 3 {
		return math.Point3{}, false
	}
	return math.P3(math.Scalar(v[0]), math.Scalar(v[1]), math.Scalar(v[2])), true
}

func tripletV(v []float64) (math.Vector3, bool) {
	if len(v) != 3 {
		return math.Vector3{}, false
	}
	return math.V3(math.Scalar(v[0]), math.Scalar(v[1]), math.Scalar(v[2])), true
}

// triplets decodes a flattened xyz list; want -1 accepts any count.
func triplets(v []float64, want int) ([]math.Point3, bool) {
	if len(v)%3 != 0 {
		return nil, false
	}
	n := len(v) / 3
	if want >= 0 && n != want {
		return nil, false
	}
	out := make([]math.Point3, n)
	for i := 0; i < n; i++ {
		out[i] = math.P3(math.Scalar(v[3*i]), math.Scalar(v[3*i+1]), math.Scalar(v[3*i+2]))
	}
	return out, true
}
