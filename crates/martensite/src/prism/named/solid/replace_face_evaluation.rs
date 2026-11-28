// SPDX-License-Identifier: GPL-2.0-only
//! Evaluate Part Surface Replace Face features during ordered part replay (#3918).
use super::*;
use oblikovati_model::feature::replace_face::ResolvedReplaceFace;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_replace_face(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        new_bodies: &[&NamedSolid],
        input: ResolvedReplaceFace,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let result = body
            .evaluate_replace_face_result(self.history, self.document, feature, new_bodies, &input)
            .map_err(|message| replace_face_failure(feature, message))?;
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn replace_face_failure(
    feature: FeatureId,
    message: String,
) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Shell
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
