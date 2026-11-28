// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Surface Ruled Surface features (creates a new surface body, #3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::feature::{
    FeatureId,
    ruled_surface::{RuledSurfaceDefinition, RuledSurfaceVector},
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_ruled_surface<
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    >(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        definition: &RuledSurfaceDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let target_slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.target_body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let target = self.bodies.get(&target_slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "ruled surface target slot {target_slot:?}: expected an active body"
            ))
        })?;
        let (vector_direction, vector_edge) = match definition.vector() {
            None => (None, None),
            Some(RuledSurfaceVector::OriginX) => (Some([1.0, 0.0, 0.0]), None),
            Some(RuledSurfaceVector::OriginY) => (Some([0.0, 1.0, 0.0]), None),
            Some(RuledSurfaceVector::OriginZ) => (Some([0.0, 0.0, 1.0]), None),
            Some(RuledSurfaceVector::WorkAxis(axis_id)) => {
                let axis = self
                    .datums
                    .axis(*axis_id)
                    .map_err(|cause| PartFeatureFailure::RuledSurfaceInput(cause.to_string()))?;
                (Some(axis.direction()), None)
            }
            Some(RuledSurfaceVector::Edge { edge, .. }) => (None, Some(edge.clone())),
        };
        let resolved = definition.resolve_with_direction(vector_direction, vector_edge);
        let result = backend.evaluate_ruled_surface(feature, target, resolved)?;
        self.body_lineage
            .register(feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.insert(feature.into(), result);
        Ok(())
    }
}
