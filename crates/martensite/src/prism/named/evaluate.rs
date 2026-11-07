// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{part::SketchPrismEvaluator, sketch_prism::ResolvedSketchPrism};
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
impl SketchPrismEvaluator for NamedPrismEvaluator<'_> {
    type Body = NamedPrismSnapshot;
    type Failure = EvaluationFailure;
    fn verify_body_reference(&self, body: &Self::Body, key: &[u8]) -> Result<(), String> {
        body.body_report(self.history, self.document, key)
            .map(|_| ())
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
    resolution: martensite_tkbo::profile_extrusion::Resolution,
}
impl<'a> CertifiedNamedPrismEvaluator<'a> {
    /// Bind the document and its explicit resolution; example: `CertifiedNamedPrismEvaluator::new(&history, guid, resolution)`.
    pub fn new(
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        resolution: martensite_tkbo::profile_extrusion::Resolution,
    ) -> Self {
        Self {
            names: NamedPrismEvaluator::new(history, document),
            resolution,
        }
    }
}
impl SketchPrismEvaluator for CertifiedNamedPrismEvaluator<'_> {
    type Body = NamedPrismSnapshot;
    type Failure = EvaluationFailure;
    fn verify_body_reference(&self, body: &Self::Body, key: &[u8]) -> Result<(), String> {
        self.names.verify_body_reference(body, key)
    }
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure> {
        let sketch = input.sketch.sketch.id();
        let result = crate::sketch_prism::certified::evaluate(input, self.resolution)
            .map_err(|error| error.with_sketch_identity(sketch).for_feature(feature))?;
        let mut named = self
            .names
            .attach_verified(feature, sketch, result.body)
            .map_err(|error| error.for_feature(feature))?;
        named.certification = Some(CertifiedPrismEvidence {
            achieved_tolerance: result.achieved_tolerance,
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
    resolution: martensite_tkbo::profile_extrusion::Resolution,
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
