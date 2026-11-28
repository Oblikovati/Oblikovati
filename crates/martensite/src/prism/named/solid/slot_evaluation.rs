// SPDX-License-Identifier: GPL-2.0-only
//! Construct a named Slot cut during ordered part replay and render red cut previews.
use super::hole_evaluation::{analytic_bounds, surface_depth, through_depth};
use super::*;
use oblikovati_model::feature::slot::ResolvedSlot;

impl NamedSolid {
    /// Build display triangles for the Slot cut volume (red Cut preview rendering).
    pub fn slot_preview_triangles(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face: &[u8],
        input: &ResolvedSlot,
        options: crate::PrismMeshOptions,
    ) -> Result<Vec<crate::DisplayTriangle>, String> {
        let resolution = match self {
            NamedSolid::Prism(body) => body.certification.as_ref().map(|value| value.resolution),
            NamedSolid::Derived(body) => body.resolution(),
        }
        .ok_or_else(|| {
            "slot preview: expected the source body's certified resolution".to_string()
        })?;
        let shape = self
            .native_shape(history, document)
            .map_err(|cause| cause.to_string())?;
        let bounds = analytic_bounds(shape, "slot preview")?;
        let mut axis = self.hole_axis(history, document, face, input.position, resolution)?;
        let inward_depth = through_depth(bounds, axis).unwrap_or(input.width * 2.0);
        if input.direction_flipped {
            axis[1] = axis[1].map(|component| -component);
        }
        let depth = match (input.depth, input.termination_surface.as_ref()) {
            (Some(depth), _) => depth,
            (None, Some(surface)) => {
                surface_depth(history, document, self, shape, axis, surface).unwrap_or(inward_depth)
            }
            (None, None) => through_depth(bounds, axis).unwrap_or(inward_depth),
        };
        let request = blend::SlotRequest {
            feature: FeatureId(1),
            face,
            axis,
            shape: input.shape,
            length: input.length,
            width: input.width,
            rotation_deg: input.rotation_deg,
            depth,
            seat: input.seat,
            extend_start: input.extend_start,
        };
        let tools = blend::build_slot_tools(document, &request, Some(resolution))?;
        let mut triangles = Vec::new();
        for tool in tools {
            triangles.extend(
                crate::prism::mesh::body_face_display_triangles(&tool.shape, options)?
                    .into_iter()
                    .flatten(),
            );
        }
        Ok(triangles)
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_slot(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        face: &[u8],
        input: ResolvedSlot,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let resolution = match body {
            NamedSolid::Prism(body) => body.certification.as_ref().map(|value| value.resolution),
            NamedSolid::Derived(body) => body.resolution(),
        }
        .ok_or_else(|| {
            PartFeatureFailure::Unsupported(
                "slot: expected the source body's certified resolution".into(),
            )
        })?;
        let shape = body
            .native_shape(self.history, self.document)
            .map_err(PartFeatureFailure::BodyReference)?;
        let bounds =
            analytic_bounds(shape, "slot").map_err(|message| slot_failure(feature, message))?;
        let mut axis = body
            .hole_axis(
                self.history,
                self.document,
                face,
                input.position,
                resolution,
            )
            .map_err(|message| slot_failure(feature, message))?;
        if input.direction_flipped {
            axis[1] = axis[1].map(|component| -component);
        }
        let depth = match (input.depth, input.termination_surface.as_ref()) {
            (Some(depth), _) => depth,
            (None, Some(surface)) => {
                surface_depth(self.history, self.document, body, shape, axis, surface)
                    .map_err(|message| slot_failure(feature, message))?
            }
            (None, None) => {
                through_depth(bounds, axis).map_err(|message| slot_failure(feature, message))?
            }
        };
        let request = blend::SlotRequest {
            feature,
            face,
            axis,
            shape: input.shape,
            length: input.length,
            width: input.width,
            rotation_deg: input.rotation_deg,
            depth,
            seat: input.seat,
            extend_start: input.extend_start,
        };
        let candidate = match body {
            NamedSolid::Prism(body) => body.slot_candidate(self.history, self.document, &request),
            NamedSolid::Derived(body) => body.slot_candidate(self.history, self.document, &request),
        }
        .map_err(|message| slot_failure(feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| slot_failure(feature, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn slot_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Hole
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
