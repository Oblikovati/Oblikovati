// SPDX-License-Identifier: GPL-2.0-only
//! Replace exactly one named body after a validated rigid motion (#4097).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::{
    feature::{FeatureId, move_body::MoveBodyDefinition},
    history::DocumentSnapshot,
};
impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_move_body<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &MoveBodyDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let motions = definition
            .resolve(&snapshot.parameters)
            .map_err(PartFeatureFailure::MoveBodyInput)?;
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
                "move body slot {slot:?}: expected a current body"
            ))
        })?;
        let result = backend.evaluate_move_body(feature, before, &motions)?;
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
