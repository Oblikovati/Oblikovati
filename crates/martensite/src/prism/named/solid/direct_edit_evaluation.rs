// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Direct Edit features during ordered part replay (#3918).
use super::*;
use oblikovati_model::feature::direct_edit::ResolvedDirectEdit;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_direct_edit(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedDirectEdit,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let candidate = body
            .direct_edit_candidate(self.history, self.document, &input)
            .map_err(|message| {
                PartFeatureFailure::Kernel(
                    Phase::MoveBody
                        .refused(Kind::InvalidResult, Entity::Body, message)
                        .for_feature(feature),
                )
            })?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
