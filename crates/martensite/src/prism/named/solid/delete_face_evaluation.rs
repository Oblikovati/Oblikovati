// SPDX-License-Identifier: GPL-2.0-only
//! Construct a named Delete Face result during ordered part replay (#3918).
use super::*;
use oblikovati_model::feature::delete_face::ResolvedDeleteFace;

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_delete_face(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedDeleteFace,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let request = blend::DeleteFaceRequest {
            faces: &input.faces,
            heal_remaining_faces: input.heal_remaining_faces,
            select_lump_or_void: input.select_lump_or_void,
        };
        let candidate = match body {
            NamedSolid::Prism(body) => {
                body.delete_face_candidate(self.history, self.document, &request)
            }
            NamedSolid::Derived(body) => {
                body.delete_face_candidate(self.history, self.document, &request)
            }
        }
        .map_err(|message| delete_face_failure(feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| delete_face_failure(feature, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn delete_face_failure(
    feature: FeatureId,
    message: String,
) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Shell
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
