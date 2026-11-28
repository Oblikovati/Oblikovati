// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Surface Stitch features during ordered part replay (#3918).
use super::super::blend::stitch::evaluate_stitch_result;
use super::*;
use oblikovati_model::feature::stitch::ResolvedStitch;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_stitch(
        &mut self,
        feature: FeatureId,
        surfaces: &[&NamedSolid],
        input: ResolvedStitch,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        evaluate_stitch_result(surfaces, self.history, self.document, feature, &input).map_err(
            |message| {
                PartFeatureFailure::Kernel(
                    crate::EvaluationPhase::Prism
                        .refused(
                            crate::EvaluationFailureKind::InvalidResult,
                            crate::EvaluationEntity::Body,
                            message,
                        )
                        .for_feature(feature),
                )
            },
        )
    }
}
