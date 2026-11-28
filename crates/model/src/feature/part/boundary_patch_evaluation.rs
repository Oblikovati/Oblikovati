// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Surface Boundary Patch features during ordered part replay (#3918).
use super::{PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure};
use crate::{
    feature::{FeatureId, boundary_patch::BoundaryPatchDefinition},
    history::DocumentSnapshot,
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_boundary_patch<
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    >(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &BoundaryPatchDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let input = definition
            .resolve_for_replay(snapshot, &self.datums)
            .map_err(PartFeatureFailure::BodyReference)?;
        let result = backend.evaluate_boundary_patch(feature, input)?;
        self.body_lineage
            .register(feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.insert(feature.into(), result);
        Ok(())
    }
}
