// SPDX-License-Identifier: GPL-2.0-only
//! Cylindrical/conical face eligibility queries and Thread evaluation during ordered Part replay (#3918).
use super::blend::thread::{
    ThreadableFaceInfo, build_thread_overlay_triangles, inspect_threadable_face,
    is_threadable_surface, validate_thread_compatibility,
};
use super::*;
use oblikovati_model::feature::thread::ResolvedThread;

impl NamedSolid {
    /// Inspect one named face and return its cylindrical/conical thread geometry if eligible.
    /// Example: `solid.threadable_face_info(&history, document, &face_key)?`.
    pub fn threadable_face_info(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face_key: &[u8],
    ) -> Result<Option<ThreadableFaceInfo>, String> {
        let shape = self.native_shape(history, document)?;
        let face = self.native_face(history, document, face_key)?;
        inspect_threadable_face(Some(shape), face)
    }

    /// Return whether this solid contains at least one cylindrical or conical face.
    /// Example: `solid.has_threadable_faces(&history, document)`.
    pub fn has_threadable_faces(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> bool {
        let Ok(keys) = self.face_reference_keys(history, document) else {
            return false;
        };
        keys.iter().any(|key| {
            self.native_face(history, document, key)
                .is_ok_and(is_threadable_surface)
        })
    }

    /// Borrow cosmetic thread overlay triangles attached to this solid by Thread features.
    pub fn thread_overlay_triangles(&self) -> &[crate::DisplayTriangle] {
        match self {
            Self::Prism(_) => &[],
            Self::Derived(body) => body.thread_overlay_triangles(),
        }
    }

    /// Compute cosmetic thread overlay triangles for a live preview before committing.
    pub fn thread_preview_triangles(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        input: &ResolvedThread,
    ) -> Result<Vec<crate::DisplayTriangle>, String> {
        let shape = self.native_shape(history, document)?;
        let face = self.native_face(history, document, &input.face)?;
        let info = inspect_threadable_face(Some(shape), face)?
            .ok_or_else(|| "thread preview: expected a cylindrical or conical face".to_owned())?;
        validate_thread_compatibility(&info, input)?;
        build_thread_overlay_triangles(face, &info, input)
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_thread(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedThread,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let shape = body
            .native_shape(self.history, self.document)
            .map_err(PartFeatureFailure::BodyReference)?;
        let face = body
            .native_face(self.history, self.document, &input.face)
            .map_err(PartFeatureFailure::BodyReference)?;
        let info = inspect_threadable_face(Some(shape), face)
            .map_err(|message| thread_failure(feature, message))?
            .ok_or_else(|| {
                thread_failure(
                    feature,
                    "thread face: expected a cylindrical or conical face".into(),
                )
            })?;
        validate_thread_compatibility(&info, &input)
            .map_err(|message| thread_failure(feature, message))?;
        let overlays = build_thread_overlay_triangles(face, &info, &input)
            .map_err(|message| thread_failure(feature, message))?;
        let candidate = body
            .rigid_candidate(
                self.history,
                self.document,
                &[RigidMotion::Translation([0.0, 0.0, 0.0])],
            )
            .map_err(|message| thread_failure(feature, message))?;
        let mut named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| thread_failure(feature, message))?;
        named.set_thread_overlays(overlays);
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn thread_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Hole
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
