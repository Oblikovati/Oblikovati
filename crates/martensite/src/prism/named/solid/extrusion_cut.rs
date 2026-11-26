// SPDX-License-Identifier: GPL-2.0-only
use super::*;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_prism_cut(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let prism = self.prism.as_mut().ok_or_else(|| {
            PartFeatureFailure::Unsupported("extrude Cut: expected certified prism replay".into())
        })?;
        let tool = prism
            .evaluate_prism(feature, input)
            .map_err(PartFeatureFailure::Kernel)?;
        let cut_failure = |message| {
            PartFeatureFailure::Kernel(
                Phase::Prism
                    .refused(Kind::InvalidResult, Entity::Body, message)
                    .for_feature(feature),
            )
        };
        let candidate = body
            .extrusion_cut_candidate(self.history, self.document, &tool)
            .map_err(cut_failure)?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(cut_failure)?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
