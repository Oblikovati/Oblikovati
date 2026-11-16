// SPDX-License-Identifier: GPL-2.0-only
//! Validate an isolated replacement before changing the current body set.
use super::{
    PartFeatureEvaluation, PartFeatureFailure, SketchPrismEvaluator,
    body_resolution::resolve_body_slot,
};
use crate::{
    feature::{FeatureId, fillet::FilletDefinition},
    history::DocumentSnapshot,
};

/// A backend that constructs and verifies a replacement fillet body.
pub trait FilletEvaluator<Body, Failure> {
    /// Construct a detached result; example: `backend.evaluate_fillet(id, body, edges, radius)?`.
    fn evaluate_fillet(
        &mut self,
        feature: FeatureId,
        body: &Body,
        edges: &[Vec<u8>],
        radius: f64,
    ) -> Result<Body, Failure>;
    /// Verify input names and their lineage, including deleted-entity records.
    /// Example: `backend.verify_fillet_lineage(before, after)?`.
    fn verify_fillet_lineage(&self, before: &Body, after: &Body) -> Result<(), String>;
}

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_fillet<B: SketchPrismEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &FilletDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            definition.body_source(),
            definition.body(),
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let radius = definition
            .resolved_radius(&snapshot.parameters)
            .map_err(PartFeatureFailure::FilletInput)?;
        let result = self.fillet_candidate(backend, slot, feature, definition, radius)?;
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature, result);
        Ok(())
    }

    fn fillet_candidate<B: SketchPrismEvaluator<Body = Body, Failure = Failure>>(
        &self,
        backend: &mut B,
        slot: FeatureId,
        feature: FeatureId,
        definition: &FilletDefinition,
        radius: f64,
    ) -> Result<Body, PartFeatureFailure<Failure>> {
        let before = self.bodies.get(&slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "body slot {slot:?}: expected a current body"
            ))
        })?;
        let evaluator = backend.fillet_evaluator().ok_or_else(|| {
            PartFeatureFailure::Unsupported(
                "fillet: expected an available named-edge evaluator".into(),
            )
        })?;
        let result = evaluator
            .evaluate_fillet(feature, before, definition.edges(), radius)
            .map_err(PartFeatureFailure::Kernel)?;
        evaluator
            .verify_fillet_lineage(before, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        backend
            .verify_body_reference(&result, definition.body())
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
