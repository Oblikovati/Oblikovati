// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Surface Ruled Surface features during ordered part replay (#3918).
use super::super::blend::extrusion_boolean::BooleanFailure;
use super::*;
use oblikovati_model::feature::ruled_surface::ResolvedRuledSurface;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_ruled_surface(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedRuledSurface,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        body.evaluate_ruled_surface_result(self.history, self.document, feature, &input)
            .map_err(|decline| {
                let message = match decline {
                    BooleanFailure::UnchangedCut { faces, .. } => {
                        format!("Ruled Surface produced no geometry ({faces} faces).")
                    }
                    BooleanFailure::Refused(msg) => msg,
                };
                PartFeatureFailure::Kernel(
                    crate::EvaluationPhase::Prism
                        .refused(
                            crate::EvaluationFailureKind::InvalidResult,
                            crate::EvaluationEntity::Body,
                            message,
                        )
                        .for_feature(feature),
                )
            })
    }
}
