// SPDX-License-Identifier: GPL-2.0-only
//! Validate named Finish inputs and update body finish appearance lineage (#3918).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::feature::{FeatureId, finish::FinishDefinition};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_finish<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        definition: &FinishDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let input = definition
            .resolve()
            .map_err(PartFeatureFailure::FinishInput)?;
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let before = self.bodies.get(&slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "finish body slot {slot:?}: expected a current body"
            ))
        })?;
        let result = backend.evaluate_finish(feature, before, input)?;
        backend
            .verify_body_reference(&result, definition.body())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature.into(), result);
        Ok(())
    }
}
