// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::sketch_prism::ResolvedSketchPrism;
/// Name native results inside the common replay before the session publishes them.
pub(super) struct NamedPrismEvaluator<'a> {
    history: &'a DocumentEditHistory,
    document: DocumentGuid,
    scope: EvaluationScope,
}
impl<'a> NamedPrismEvaluator<'a> {
    /// Bind one document namespace and live owner; example: `NamedPrismEvaluator::new(session.history(), guid)`.
    pub(super) fn new(history: &'a DocumentEditHistory, document: DocumentGuid) -> Self {
        Self {
            history,
            document,
            scope: history.evaluation_scope(),
        }
    }
}
impl NamedPrismEvaluator<'_> {
    pub(super) fn verify_body_reference(
        &self,
        body: &NamedPrismSnapshot,
        key: &[u8],
    ) -> Result<(), String> {
        body.body_report(self.history, self.document, key)
            .map(|_| ())
    }
    pub(super) fn resolve_planar_face(
        &self,
        body: &NamedPrismSnapshot,
        key: &[u8],
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        body.face_plane(self.history, self.document, key)
    }
    pub(super) fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<NamedPrismSnapshot, EvaluationFailure> {
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
pub(super) struct CertifiedNamedPrismEvaluator<'a> {
    names: NamedPrismEvaluator<'a>,
    resolution: PrismResolutionPolicy,
}
impl<'a> CertifiedNamedPrismEvaluator<'a> {
    /// Bind the document and its explicit resolution; example: `CertifiedNamedPrismEvaluator::new(&history, guid, resolution)`.
    pub(super) fn new(
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
    pub(super) fn with_policy(
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
impl CertifiedNamedPrismEvaluator<'_> {
    pub(super) fn resolution_policy(&self) -> PrismResolutionPolicy {
        self.resolution
    }
    pub(super) fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<NamedPrismSnapshot, EvaluationFailure> {
        let sketch = input.sketch.sketch.id();
        let (result, resolution) =
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
            resolution,
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
        .evaluate_part_features(&mut CertifiedNamedSolidEvaluator::new(
            history, document, resolution,
        ))
        .map_err(|cause| cause.to_string())?;
    current_prism_result(evaluation, feature)
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
