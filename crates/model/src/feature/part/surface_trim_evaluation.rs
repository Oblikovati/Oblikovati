// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Surface Trim features during ordered Part replay (#3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::{
    feature::{
        FeatureId,
        hole::HoleOriginPlane,
        surface_trim::{
            ResolvedSurfaceTrim, ResolvedSurfaceTrimTool, SurfaceTrimDefinition, SurfaceTrimTool,
        },
        work_plane::WorkPlaneResolver,
    },
    sketch::SketchPlane,
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_surface_trim<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        definition: &SurfaceTrimDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let target_slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.target_body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let (tool, tool_slot) = match definition.tool() {
            SurfaceTrimTool::OriginPlane(origin) => {
                (ResolvedSurfaceTrimTool::Plane(origin_plane(*origin)), None)
            }
            SurfaceTrimTool::WorkPlane(plane_id) => {
                let plane = self.datums.plane(*plane_id).map_err(|cause| {
                    PartFeatureFailure::SurfaceTrimInput(format!(
                        "surface trim work plane {plane_id:?}: {cause}"
                    ))
                })?;
                (ResolvedSurfaceTrimTool::Plane(plane), None)
            }
            SurfaceTrimTool::SurfaceFace { body, face } => {
                let source = definition.tool_body_source().ok_or_else(|| {
                    PartFeatureFailure::SurfaceTrimInput(
                        "surface trim tool face: expected a tool body source".into(),
                    )
                })?;
                let slot =
                    resolve_body_slot(&self.bodies, &self.body_lineage, backend, source, body)
                        .map_err(PartFeatureFailure::BodyReference)?;
                (
                    ResolvedSurfaceTrimTool::SurfaceFace {
                        face_key: face.clone(),
                    },
                    Some(slot),
                )
            }
        };
        let target = self.bodies.get(&target_slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "surface trim target slot {target_slot:?}: expected an active surface body"
            ))
        })?;
        let tool_body = match tool_slot {
            Some(slot) => Some(self.bodies.get(&slot).ok_or_else(|| {
                PartFeatureFailure::BodyReference(format!(
                    "surface trim tool slot {slot:?}: expected an active tool body"
                ))
            })?),
            None => None,
        };
        let input = ResolvedSurfaceTrim {
            tool,
            remove_side: definition.remove_side(),
        };
        let result = backend.evaluate_surface_trim(feature, target, tool_body, input)?;
        backend
            .verify_body_reference(&result, definition.target_body())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(target_slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&target_slot);
        self.bodies.insert(feature.into(), result);
        Ok(())
    }
}

fn origin_plane(plane: HoleOriginPlane) -> SketchPlane {
    match plane {
        HoleOriginPlane::Xy => SketchPlane::xy(),
        HoleOriginPlane::Xz => SketchPlane::xz(),
        HoleOriginPlane::Yz => SketchPlane::yz(),
    }
}
