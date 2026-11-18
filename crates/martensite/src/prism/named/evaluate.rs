// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{part::PartFeatureEvaluator, sketch_prism::ResolvedSketchPrism};
/// Name native results inside the common replay before the session publishes them.
pub struct NamedPrismEvaluator<'a> {
    history: &'a DocumentEditHistory,
    document: DocumentGuid,
    scope: EvaluationScope,
}
impl<'a> NamedPrismEvaluator<'a> {
    /// Bind one document namespace and live owner; example: `NamedPrismEvaluator::new(session.history(), guid)`.
    pub fn new(history: &'a DocumentEditHistory, document: DocumentGuid) -> Self {
        Self {
            history,
            document,
            scope: history.evaluation_scope(),
        }
    }
}
impl PartFeatureEvaluator for NamedPrismEvaluator<'_> {
    type Body = NamedPrismSnapshot;
    type Failure = EvaluationFailure;
    fn evaluate_blend(
        &mut self,
        kind: oblikovati_model::feature::part::PartBlendKind,
        _feature: oblikovati_model::feature::FeatureId,
        _body: &Self::Body,
        _edges: &[Vec<u8>],
        _distance: f64,
    ) -> Result<Self::Body, oblikovati_model::feature::part::PartFeatureFailure<Self::Failure>>
    {
        Err(kind.unsupported())
    }

    fn verify_body_reference(&self, body: &Self::Body, key: &[u8]) -> Result<(), String> {
        body.body_report(self.history, self.document, key)
            .map(|_| ())
    }
    fn resolve_planar_face(
        &self,
        body: &Self::Body,
        key: &[u8],
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        body.face_plane(self.history, self.document, key)
    }
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, EvaluationFailure> {
        self.evaluate_named(feature, input)
            .map_err(|cause| cause.for_feature(feature))
    }
}
impl NamedPrismEvaluator<'_> {
    fn evaluate_named(
        &self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<NamedPrismSnapshot, EvaluationFailure> {
        let sketch = input.sketch.sketch.id();
        let snapshot = crate::sketch_prism::evaluate_resolved_prism(input)?;
        self.attach_verified(feature, sketch, snapshot)
    }
    fn attach_verified(
        &self,
        feature: FeatureId,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        snapshot: PrismProbeSnapshot,
    ) -> Result<NamedPrismSnapshot, EvaluationFailure> {
        let named = attach_names(
            snapshot,
            self.document,
            feature,
            sketch.0,
            self.scope.clone(),
        )
        .map_err(|message| naming_decline(sketch, message))?;
        named
            .verify_written_names(self.history, self.document, feature, sketch.0)
            .map_err(|message| naming_decline(sketch, message))?;
        Ok(named)
    }
}

/// Inject an explicit model resolution into certified, named feature replay.
pub struct CertifiedNamedPrismEvaluator<'a> {
    names: NamedPrismEvaluator<'a>,
    resolution: PrismResolutionPolicy,
}
impl<'a> CertifiedNamedPrismEvaluator<'a> {
    /// Bind the document and its explicit resolution; example: `CertifiedNamedPrismEvaluator::new(&history, guid, resolution)`.
    pub fn new(
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        resolution: PrismResolution,
    ) -> Self {
        Self {
            names: NamedPrismEvaluator::new(history, document),
            resolution: PrismResolutionPolicy::Fixed(resolution),
        }
    }
    /// Bind the same per-sketch policy used by profile discovery.
    /// Example: `CertifiedNamedPrismEvaluator::with_policy(history, guid, policy)`.
    pub fn with_policy(
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        resolution: PrismResolutionPolicy,
    ) -> Self {
        Self {
            names: NamedPrismEvaluator::new(history, document),
            resolution,
        }
    }
}
impl PartFeatureEvaluator for CertifiedNamedPrismEvaluator<'_> {
    type Body = NamedPrismSnapshot;
    type Failure = EvaluationFailure;
    fn evaluate_blend(
        &mut self,
        kind: oblikovati_model::feature::part::PartBlendKind,
        _feature: oblikovati_model::feature::FeatureId,
        _body: &Self::Body,
        _edges: &[Vec<u8>],
        _distance: f64,
    ) -> Result<Self::Body, oblikovati_model::feature::part::PartFeatureFailure<Self::Failure>>
    {
        Err(kind.unsupported())
    }

    fn verify_body_reference(&self, body: &Self::Body, key: &[u8]) -> Result<(), String> {
        self.names.verify_body_reference(body, key)
    }
    fn resolve_planar_face(
        &self,
        body: &Self::Body,
        key: &[u8],
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        self.names.resolve_planar_face(body, key)
    }
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure> {
        let sketch = input.sketch.sketch.id();
        let result =
            crate::sketch_prism::certified::evaluate_feature(input, self.resolution, feature)?;
        let mut named = self
            .names
            .attach_verified(feature, sketch, result.body)
            .map_err(|error| error.for_feature(feature))?;
        named
            .bind_subentities(
                self.names.history,
                self.names.document,
                feature,
                sketch.0,
                &result.subentities,
                result.subentity_shapes,
            )
            .map_err(|message| naming_decline(sketch, message).for_feature(feature))?;
        named.certification = Some(CertifiedPrismEvidence {
            achieved_tolerance: result.achieved_tolerance,
            subentities: result.subentities,
            vertex_sources: result.vertex_sources,
        });
        Ok(named)
    }
}

/// Replay a stored feature through the certified evaluator; example: `retain_certified_named_sketch_prism(&history, guid, feature, resolution)?`.
pub fn retain_certified_named_sketch_prism(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    feature: FeatureId,
    resolution: PrismResolution,
) -> Result<NamedPrismSnapshot, String> {
    let job = history.evaluation_snapshot();
    let evaluation = job
        .snapshot()
        .evaluate_part_features(&mut CertifiedNamedPrismEvaluator::new(
            history, document, resolution,
        ))
        .map_err(|cause| cause.to_string())?;
    let decline = crate::stored_prism::stored_body_decline(&evaluation, feature);
    evaluation.into_body(feature).ok_or(decline)
}
fn naming_decline(
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    message: String,
) -> EvaluationFailure {
    Phase::Naming
        .refused(Kind::InvalidReference, Entity::Body, message)
        .with_sketch_identity(sketch)
}

#[cfg(test)]
#[path = "../../../tests/unit/named_evaluation_failure.rs"]
mod tests;
