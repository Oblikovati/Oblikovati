// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Modify Combine solid Booleans during ordered part replay (#3918).
use super::super::blend::extrusion_boolean::BooleanFailure;
use super::*;
use oblikovati_model::feature::combine::ResolvedCombine;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_combine(
        &mut self,
        feature: FeatureId,
        base: &NamedSolid,
        tools: &[&NamedSolid],
        input: ResolvedCombine,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |kind, message| {
            PartFeatureFailure::Kernel(
                crate::EvaluationPhase::Prism
                    .refused(kind, crate::EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let candidate = base
            .combine_candidate(self.history, self.document, tools, input.operation)
            .map_err(|decline| match decline {
                BooleanFailure::UnchangedCut { faces, .. } => failure(
                    crate::EvaluationFailureKind::UnchangedCut,
                    format!(
                        "The Combine operation did not change the target body ({faces} faces)."
                    ),
                ),
                BooleanFailure::Refused(message) => {
                    failure(crate::EvaluationFailureKind::InvalidResult, message)
                }
            })?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(crate::EvaluationFailureKind::InvalidResult, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(base, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
