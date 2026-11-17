// SPDX-License-Identifier: GPL-2.0-only
//! Validate isolated blend replacements before changing the current body set.
use super::{
    PartFeatureEvaluation, PartFeatureFailure, SketchPrismEvaluator,
    body_resolution::resolve_body_slot,
};
use crate::{
    feature::{FeatureId, chamfer::ChamferDefinition, fillet::FilletDefinition},
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
    /// Verify input names and their lineage; example: `backend.verify_fillet_lineage(before, after)?`.
    fn verify_fillet_lineage(&self, before: &Body, after: &Body) -> Result<(), String>;
}

struct BlendInput<'a> {
    body_source: FeatureId,
    body: &'a [u8],
    edges: &'a [Vec<u8>],
    distance: f64,
    kind: BlendKind,
}
#[derive(Clone, Copy)]
enum BlendKind {
    Fillet,
    Chamfer,
}

impl<Body, Failure> PartFeatureEvaluation<Body, Failure> {
    pub(super) fn evaluate_fillet<B: SketchPrismEvaluator<Body = Body, Failure = Failure>>(
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
                kind: BlendKind::Fillet,
            },
        )
    }
    pub(super) fn evaluate_chamfer<B: SketchPrismEvaluator<Body = Body, Failure = Failure>>(
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
                kind: BlendKind::Chamfer,
            },
        )
    }
    fn evaluate_blend<B: SketchPrismEvaluator<Body = Body, Failure = Failure>>(
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
        let result = blend_candidate(backend, feature, before, &input)?;
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

fn blend_candidate<B: SketchPrismEvaluator>(
    backend: &mut B,
    feature: FeatureId,
    before: &B::Body,
    input: &BlendInput<'_>,
) -> Result<B::Body, PartFeatureFailure<B::Failure>> {
    match input.kind {
        BlendKind::Fillet => fillet_candidate(backend, feature, before, input),
        BlendKind::Chamfer => chamfer_candidate(backend, feature, before, input),
    }
}

fn fillet_candidate<B: SketchPrismEvaluator>(
    backend: &mut B,
    feature: FeatureId,
    before: &B::Body,
    input: &BlendInput<'_>,
) -> Result<B::Body, PartFeatureFailure<B::Failure>> {
    let evaluator = backend.fillet_evaluator().ok_or_else(|| {
        PartFeatureFailure::Unsupported("fillet: expected an available named-edge evaluator".into())
    })?;
    let result = evaluator
        .evaluate_fillet(feature, before, input.edges, input.distance)
        .map_err(PartFeatureFailure::Kernel)?;
    evaluator
        .verify_fillet_lineage(before, &result)
        .map_err(PartFeatureFailure::BodyReference)?;
    Ok(result)
}

fn chamfer_candidate<B: SketchPrismEvaluator>(
    backend: &mut B,
    feature: FeatureId,
    before: &B::Body,
    input: &BlendInput<'_>,
) -> Result<B::Body, PartFeatureFailure<B::Failure>> {
    let evaluator = backend.chamfer_evaluator().ok_or_else(|| {
        PartFeatureFailure::Unsupported(
            "chamfer: expected an available named-edge evaluator".into(),
        )
    })?;
    let result = evaluator
        .evaluate_chamfer(feature, before, input.edges, input.distance)
        .map_err(PartFeatureFailure::Kernel)?;
    evaluator
        .verify_chamfer_lineage(before, &result)
        .map_err(PartFeatureFailure::BodyReference)?;
    Ok(result)
}
