// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Surface Extend features during ordered part replay (#3918).
use super::*;
use oblikovati_model::feature::extend_surface::ResolvedExtendSurface;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_extend_surface(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedExtendSurface,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let result = body
            .evaluate_extend_surface_result(self.history, self.document, feature, &input)
            .map_err(|message| {
                PartFeatureFailure::Kernel(
                    crate::EvaluationPhase::Prism
                        .refused(
                            crate::EvaluationFailureKind::InvalidResult,
                            crate::EvaluationEntity::Body,
                            message,
                        )
                        .for_feature(feature),
                )
            })?;
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
