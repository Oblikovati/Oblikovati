// SPDX-License-Identifier: GPL-2.0-only
//! Certified Decal evaluation: verify target face and sketch image, preserving solid lineage.
use crate::{
    CertifiedNamedSolidEvaluator, EvaluationEntity, EvaluationFailure, EvaluationFailureKind,
    EvaluationPhase, NamedSolid, RigidMotion,
};
use oblikovati_model::feature::{
    FeatureId,
    part::{PartFeatureEvaluator, PartFeatureFailure},
    sketch_decal::ResolvedSketchDecal,
};

impl CertifiedNamedSolidEvaluator<'_> {
    /// Evaluate a Decal against a live named target solid and face.
    pub(crate) fn evaluate_named_decal(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedSketchDecal<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |message: String| {
            PartFeatureFailure::Kernel(
                EvaluationPhase::Prism
                    .refused(
                        EvaluationFailureKind::InvalidResult,
                        EvaluationEntity::Body,
                        message,
                    )
                    .for_feature(feature),
            )
        };
        let is_planar = self.resolve_planar_face(body, input.target_face).is_ok();
        if !is_planar {
            let valid_face = body
                .face_reference_keys(self.history, self.document)
                .map_err(failure)?
                .iter()
                .any(|key| key.as_slice() == input.target_face);
            if !valid_face {
                return Err(failure(
                    "decal target face: expected a live named face on the target solid".into(),
                ));
            }
        }
        if input.wrap_to_face && is_planar {
            return Err(failure(
                "decal: Wrap to Face is unavailable for planar faces".into(),
            ));
        }
        let candidate = body
            .rigid_candidate(
                self.history,
                self.document,
                &[RigidMotion::Translation([0.0, 0.0, 0.0])],
            )
            .map_err(failure)?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
