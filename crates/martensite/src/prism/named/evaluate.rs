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
