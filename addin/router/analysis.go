// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/analysis"
)

// Analysis (M18-F01 #423): engineering analysis on the active part — mass properties.

// registerAnalysisHandlers wires the analysis.* methods.
func (r *Router) registerAnalysisHandlers() {
	r.handlers[wire.MethodAnalysisMassProperties] = analysisMassProperties
}

func analysisMassProperties(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.MassPropertiesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	accuracy := types.MassPropertiesMedium
	if in.Accuracy != "" {
		a, ok := types.ParseMassPropertiesAccuracy(in.Accuracy)
		if !ok {
			return nil, fmt.Errorf("analysis.massProperties: unknown accuracy %q (want low|medium|high)", in.Accuracy)
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
	return json.Marshal(massPropertiesResult(mp))
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
