// SPDX-License-Identifier: GPL-2.0-only
use super::*;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_prism_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedSketchPrism<'_>,
        kind: oblikovati_model::feature::sketch_prism::PrismBoolean,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let prism = self.prism.as_mut().ok_or_else(|| {
            PartFeatureFailure::Unsupported(format!(
                "extrude {kind}: expected certified prism replay"
            ))
        })?;
        let tool = prism
            .evaluate_prism(feature, input)
            .map_err(PartFeatureFailure::Kernel)?;
        let boolean_failure = |message| {
            PartFeatureFailure::Kernel(
                Phase::Prism
                    .refused(Kind::InvalidResult, Entity::Body, message)
                    .for_feature(feature),
            )
        };
        let candidate = body
            .extrusion_boolean_candidate(self.history, self.document, &tool, kind)
            .map_err(boolean_failure)?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(boolean_failure)?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
