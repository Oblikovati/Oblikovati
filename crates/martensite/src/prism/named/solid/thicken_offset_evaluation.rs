// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Modify Thicken/Offset features during ordered part replay (#3918).
use super::super::blend::extrusion_boolean::BooleanFailure;
use super::*;
use oblikovati_model::feature::thicken_offset::{
    ResolvedThickenOffset, ThickenBoolean, ThickenOutputMode,
};

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_thicken_offset(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedThickenOffset,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |kind, message| {
            PartFeatureFailure::Kernel(
                crate::EvaluationPhase::Prism
                    .refused(kind, crate::EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let result = body
            .evaluate_thicken_offset_result(self.history, self.document, feature, &input)
            .map_err(|decline| match decline {
                BooleanFailure::UnchangedCut { faces, .. } => failure(
                    crate::EvaluationFailureKind::UnchangedCut,
                    format!(
                        "The Thicken operation did not change the target body ({faces} faces)."
                    ),
                ),
                BooleanFailure::Refused(message) => {
                    failure(crate::EvaluationFailureKind::InvalidResult, message)
                }
            })?;
        let replaces_target = input.output == ThickenOutputMode::Solid
            && (input.automatic_blending || input.boolean != ThickenBoolean::NewSolid);
        if replaces_target {
            self.verify_result_lineage(body, &result)
                .map_err(PartFeatureFailure::BodyReference)?;
        }
        Ok(result)
    }
}
