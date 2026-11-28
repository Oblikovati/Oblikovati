// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Surface Trim features during ordered part replay (#3918).
use super::*;
use oblikovati_model::feature::surface_trim::ResolvedSurfaceTrim;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_surface_trim(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        tool_body: Option<&NamedSolid>,
        input: ResolvedSurfaceTrim,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let result = body
            .evaluate_surface_trim_result(self.history, self.document, feature, tool_body, &input)
            .map_err(|message| surface_trim_failure(feature, message))?;
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn surface_trim_failure(
    feature: FeatureId,
    message: String,
) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Prism
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
