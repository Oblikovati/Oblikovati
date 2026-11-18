// SPDX-License-Identifier: GPL-2.0-only
//! Validate isolated blend replacements before changing the current body set.
use super::{
    PartBlendKind, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    body_resolution::resolve_body_slot,
};
use crate::{
    feature::{FeatureId, chamfer::ChamferDefinition, fillet::FilletDefinition},
    history::DocumentSnapshot,
};

struct BlendInput<'a> {
    body_source: FeatureId,
    body: &'a [u8],
    edges: &'a [Vec<u8>],
    distance: f64,
    kind: PartBlendKind,
}

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_fillet<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &FilletDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let distance = definition
            .resolved_radius(&snapshot.parameters)
            .map_err(PartFeatureFailure::FilletInput)?;
        self.evaluate_blend(
            backend,
            feature,
            BlendInput {
                body_source: definition.body_source(),
                body: definition.body(),
                edges: definition.edges(),
                distance,
                kind: PartBlendKind::Fillet,
            },
        )
    }
    pub(super) fn evaluate_chamfer<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        snapshot: &DocumentSnapshot,
        backend: &mut B,
        feature: FeatureId,
        definition: &ChamferDefinition,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let distance = definition
            .resolved_distance(&snapshot.parameters)
            .map_err(PartFeatureFailure::ChamferInput)?;
        self.evaluate_blend(
            backend,
            feature,
            BlendInput {
                body_source: definition.body_source(),
                body: definition.body(),
                edges: definition.edges(),
                distance,
                kind: PartBlendKind::Chamfer,
            },
        )
    }
    fn evaluate_blend<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &mut self,
        backend: &mut B,
        feature: FeatureId,
        input: BlendInput<'_>,
    ) -> Result<(), PartFeatureFailure<Failure>> {
        let slot = resolve_body_slot(
            &self.bodies,
            &self.body_lineage,
            backend,
            input.body_source,
            input.body,
        )
        .map_err(PartFeatureFailure::BodyReference)?;
        let before = self.bodies.get(&slot).ok_or_else(|| {
            PartFeatureFailure::BodyReference(format!(
                "body slot {slot:?}: expected a current body"
            ))
        })?;
        let result =
            backend.evaluate_blend(input.kind, feature, before, input.edges, input.distance)?;
        backend
            .verify_body_reference(&result, input.body)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.body_lineage
            .replace(slot, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        self.bodies.remove(&slot);
        self.bodies.insert(feature, result);
        Ok(())
    }
}
