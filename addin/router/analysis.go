// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
)

// Analysis (M18-F01 #423): engineering analysis on the active part — mass properties.

// registerAnalysisHandlers wires the analysis.* methods.
func (r *Router) registerAnalysisHandlers() {
	r.readOnly(wire.MethodAnalysisMassProperties, typedPart(analysisMassProperties))
	r.readOnly(wire.MethodAnalysisMeasure, typedPart(analysisMeasure))
	r.readOnly(wire.MethodAnalysisModelHealth, partQuery(analysisModelHealth))
}

func analysisModelHealth(_ *app.Session, part *compdef.PartComponentDefinition) (wire.ModelHealthResult, error) {
	mh := analysis.ModelHealthOf(part.Features())
	return modelHealthResult(mh), nil
}

// modelHealthResult flattens the aggregated health into the wire DTO, carrying each status as its
// stable lowercase name.
func modelHealthResult(mh analysis.ModelHealth) wire.ModelHealthResult {
	out := wire.ModelHealthResult{Overall: mh.Overall.String(), SickCount: mh.SickCount}
	for _, f := range mh.Unhealthy {
		out.Unhealthy = append(out.Unhealthy, wire.FeatureHealth{Name: f.Name, Status: f.Status.String(), Reason: f.Reason})
	}
	return out
}

func analysisMeasure(_ *app.Session, part *compdef.PartComponentDefinition, in wire.MeasureArgs) (wire.MeasureResult, error) {
	mt, ok := types.ParseMeasureType(in.Type)
	if !ok {
		return wire.MeasureResult{}, fmt.Errorf("analysis.measure: unknown measure type %q (want length|area|distance|minDistance|angle)", in.Type)
	}
	bodies := part.SurfaceBodies().All()
	if in.BodyIndex < 0 || in.BodyIndex >= len(bodies) {
		return wire.MeasureResult{}, fmt.Errorf("analysis.measure: body index %d out of range [0,%d)", in.BodyIndex, len(bodies))
	}
	value, unit, err := measureEntity(bodies[in.BodyIndex], mt, in.KeyA, in.KeyB, in.KeyC)
	if err != nil {
		return wire.MeasureResult{}, err
	}
	return wire.MeasureResult{Type: mt.String(), Value: value, Unit: unit}, nil
}

// measureEntity resolves the entity (or pair) by reference key and computes the requested quantity.
func measureEntity(body *topo.Body, mt types.MeasureType, keyA, keyB, keyC string) (float64, string, error) {
	q := ops.DefaultQuality()
	switch mt {
	case types.MeasureLength:
		return measureEdgeLength(body, keyA, q)
	case types.MeasureArea:
		return measureFaceArea(body, keyA, q)
	case types.MeasureDistance:
		return measureVertexDistance(body, keyA, keyB)
	case types.MeasureMinDistance:
		return measureMinDistance(body, keyA, keyB, q)
	case types.MeasureAngle:
		return measureAngle(body, keyA, keyB, keyC, q)
	case types.MeasureLoopLength:
		return measureLoopLength(body, keyA, q)
	}
	return 0, "", fmt.Errorf("analysis.measure: unsupported measure type %v", mt)
}

func measureLoopLength(body *topo.Body, keyA string, q ops.Quality) (float64, string, error) {
	f, ok := body.FindFaceByKey(decodeKey(keyA))
	if !ok {
		return 0, "", fmt.Errorf("analysis.measure: no face for key %q", keyA)
	}
	return analysis.FaceLoopLengthMm(f, q), "mm", nil
}

func measureEdgeLength(body *topo.Body, keyA string, q ops.Quality) (float64, string, error) {
	e, ok := body.FindEdgeByKey(decodeKey(keyA))
	if !ok {
		return 0, "", fmt.Errorf("analysis.measure: no edge for key %q", keyA)
	}
	return analysis.EdgeLengthMm(e, q), "mm", nil
}

func measureFaceArea(body *topo.Body, keyA string, q ops.Quality) (float64, string, error) {
	f, ok := body.FindFaceByKey(decodeKey(keyA))
	if !ok {
		return 0, "", fmt.Errorf("analysis.measure: no face for key %q", keyA)
	}
	return analysis.FaceAreaMm2(f, q), "mm²", nil
}

func measureVertexDistance(body *topo.Body, keyA, keyB string) (float64, string, error) {
	a, okA := body.FindVertexByKey(decodeKey(keyA))
	b, okB := body.FindVertexByKey(decodeKey(keyB))
	if !okA || !okB {
		return 0, "", fmt.Errorf("analysis.measure: distance needs two vertex keys (keyA=%q, keyB=%q)", keyA, keyB)
	}
	return analysis.VertexDistanceMm(a, b), "mm", nil
}

func measureMinDistance(body *topo.Body, keyA, keyB string, q ops.Quality) (float64, string, error) {
	a, errA := resolveMeasureEntity(body, keyA)
	b, errB := resolveMeasureEntity(body, keyB)
	if errA != nil || errB != nil {
		return 0, "", fmt.Errorf("analysis.measure: minDistance needs two entity keys (keyA=%q, keyB=%q)", keyA, keyB)
	}
	return analysis.MinDistanceMm(a, b, q), "mm", nil
}

// measureAngle returns a two-entity angle (edge/face), or a three-vertex angle when keyC is given.
func measureAngle(body *topo.Body, keyA, keyB, keyC string, q ops.Quality) (float64, string, error) {
	if keyC != "" {
		return threePointAngle(body, keyA, keyB, keyC)
	}
	a, errA := resolveMeasureEntity(body, keyA)
	b, errB := resolveMeasureEntity(body, keyB)
	if errA != nil || errB != nil {
		return 0, "", fmt.Errorf("analysis.measure: angle needs two entity keys (keyA=%q, keyB=%q)", keyA, keyB)
	}
	deg, err := analysis.AngleDegrees(a, b, q)
	if err != nil {
		return 0, "", err
	}
	return deg, "deg", nil
}

// threePointAngle returns the angle at apex vertex keyB between vertices keyA and keyC.
func threePointAngle(body *topo.Body, keyA, keyB, keyC string) (float64, string, error) {
	a, okA := body.FindVertexByKey(decodeKey(keyA))
	apex, okB := body.FindVertexByKey(decodeKey(keyB))
	c, okC := body.FindVertexByKey(decodeKey(keyC))
	if !okA || !okB || !okC {
		return 0, "", fmt.Errorf("analysis.measure: angle needs three vertex keys (keyA=%q, keyB=%q, keyC=%q)", keyA, keyB, keyC)
	}
	return analysis.ThreePointAngleDegrees(a, apex, c), "deg", nil
}

// resolveMeasureEntity resolves a reference key to a vertex, edge or face of the body (whichever it
// names), for entity-to-entity measurement.
func resolveMeasureEntity(body *topo.Body, key string) (analysis.MeasureEntity, error) {
	k := decodeKey(key)
	if f, ok := body.FindFaceByKey(k); ok {
		return analysis.MeasureEntity{Face: f}, nil
	}
	if e, ok := body.FindEdgeByKey(k); ok {
		return analysis.MeasureEntity{Edge: e}, nil
	}
	if v, ok := body.FindVertexByKey(k); ok {
		return analysis.MeasureEntity{Vertex: v}, nil
	}
	return analysis.MeasureEntity{}, fmt.Errorf("analysis.measure: no entity for key %q", key)
}

// decodeKey carries a reference key as raw bytes — matching get_reference_keys and every other
// key-consuming wire method (body_facets, brep, assembly_features); an unknown key matches nothing.
func decodeKey(s string) []byte {
	return []byte(s)
}

func analysisMassProperties(s *app.Session, part *compdef.PartComponentDefinition, in wire.MassPropertiesArgs) (wire.MassPropertiesResult, error) {
	accuracy := types.MassPropertiesMedium
	if in.Accuracy != "" {
		a, ok := types.ParseMassPropertiesAccuracy(in.Accuracy)
		if !ok {
			return wire.MassPropertiesResult{}, fmt.Errorf("analysis.massProperties: unknown accuracy %q (want low|medium|high)", in.Accuracy)
		}
		accuracy = a
	}
	density := in.DensityGCm3
	if density == 0 { // default to the part's assigned material density
		if props, ok := s.PhysicalProperties(); ok && props.Density > 0 {
			density = props.Density
		}
	}
	mp := analysis.MassPropertiesOf(part.SurfaceBodies().All(), density, accuracy)
	return massPropertiesResult(mp), nil
}

// massPropertiesResult flattens the model mass properties into the wire DTO.
func massPropertiesResult(mp analysis.MassProperties) wire.MassPropertiesResult {
	return wire.MassPropertiesResult{
		VolumeMm3: mp.VolumeMm3, SurfaceAreaMm2: mp.SurfaceAreaMm2, MassG: mp.MassG, DensityGCm3: mp.DensityGCm3,
		CentroidXMm: mp.CentroidXMm, CentroidYMm: mp.CentroidYMm, CentroidZMm: mp.CentroidZMm,
		InertiaXxGmm2: mp.InertiaXxGmm2, InertiaYyGmm2: mp.InertiaYyGmm2, InertiaZzGmm2: mp.InertiaZzGmm2,
		InertiaXyGmm2: mp.InertiaXyGmm2, InertiaYzGmm2: mp.InertiaYzGmm2, InertiaZxGmm2: mp.InertiaZxGmm2,
		PrincipalMomentsGmm2: mp.PrincipalMomentsGmm2, PrincipalAxes: mp.PrincipalAxes,
	}
}
