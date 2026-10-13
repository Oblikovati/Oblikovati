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
    type Failure = String;
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, String> {
        let sketch = input.sketch.sketch.id().0;
        let snapshot = crate::retain_resolved_sketch_prism(input)?;
        let named = attach_names(snapshot, self.document, feature, sketch, self.scope.clone())?;
        named.verify_written_names(self.history, self.document, feature, sketch)?;
        Ok(named)
    }
}
