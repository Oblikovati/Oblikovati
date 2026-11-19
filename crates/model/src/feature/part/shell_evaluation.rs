// SPDX-License-Identifier: GPL-2.0-only
//! Replace a body only after exact Shell evaluation succeeds (#4106).
use super::{
    PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::{
    feature::{FeatureId, shell::ShellDefinition},
    history::DocumentSnapshot,
};

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_shell<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &ShellDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let input = definition
            .resolve(&snapshot.parameters)
            .map_err(PartFeatureFailure::ShellInput)?;
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
                "shell body slot {slot:?}: expected a current body"
            ))
        })?;
        let result = backend.evaluate_shell(feature, before, definition.openings(), input)?;
        backend
            .verify_body_reference(&result, definition.body())
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature, result);
        Ok(())
    }
}
