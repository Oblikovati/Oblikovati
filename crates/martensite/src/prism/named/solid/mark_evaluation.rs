// SPDX-License-Identifier: GPL-2.0-only
//! Connected face-chain query and Mark feature evaluation during ordered Part replay (#3918).
use super::blend::mark::{build_mark_overlay_triangles, connected_face_chain_shapes};
use super::*;
use oblikovati_model::feature::mark::{MarkMethod, ResolvedMark};

const MIN_SOLID_VOLUME_CM3: f64 = 1e-9;

impl NamedSolid {
    /// Expand a seed face to all edge-connected faces on this solid via B-rep topology traversal.
    /// Example: `solid.connected_face_chain(&history, document, &face_key)?`.
    pub fn connected_face_chain(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face_key: &[u8],
    ) -> Result<Vec<Vec<u8>>, String> {
        let keys = self.face_reference_keys(history, document)?;
        let mut faces = Vec::with_capacity(keys.len());
        for key in keys {
            let shape = self.native_face(history, document, &key)?.clone();
            faces.push((key, shape));
        }
        connected_face_chain_shapes(&faces, face_key)
    }

    /// Borrow mark imprint overlay triangles attached to this solid by Mark features.
    pub fn mark_overlay_triangles(&self) -> &[crate::DisplayTriangle] {
        match self {
            Self::Prism(_) => &[],
            Self::Derived(body) => body.mark_overlay_triangles(),
        }
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_mark(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedMark,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        if body.is_surface() {
            return Err(mark_failure(
                feature,
                "Mark failed. The selected component has no volume.".into(),
            ));
        }
        let volume = body
            .volume(self.history, self.document)
            .map_err(|cause| mark_failure(feature, cause))?;
        if !volume.is_finite() || volume <= MIN_SOLID_VOLUME_CM3 {
            return Err(mark_failure(
                feature,
                "Mark failed. The selected component has no volume.".into(),
            ));
        }
        let face_keys = match input.method {
            MarkMethod::WrapToFace => {
                if input.faces.is_empty() {
                    return Err(mark_failure(
                        feature,
                        "mark wrap faces: expected at least one selected face".into(),
                    ));
                }
                input.faces.clone()
            }
            MarkMethod::ProjectToClosestFace => body
                .face_reference_keys(self.history, self.document)
                .map_err(PartFeatureFailure::BodyReference)?,
        };
        let mut target_faces = Vec::with_capacity(face_keys.len());
        for key in &face_keys {
            let face = body
                .native_face(self.history, self.document, key)
                .map_err(PartFeatureFailure::BodyReference)?;
            target_faces.push(face.clone());
        }
        let overlays = build_mark_overlay_triangles(&target_faces, &input)
            .map_err(|message| mark_failure(feature, message))?;
        let candidate = body
            .rigid_candidate(
                self.history,
                self.document,
                &[RigidMotion::Translation([0.0, 0.0, 0.0])],
            )
            .map_err(|message| mark_failure(feature, message))?;
        let mut named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| mark_failure(feature, message))?;
        let mut combined_marks = body.mark_overlay_triangles().to_vec();
        combined_marks.extend(overlays);
        named.set_mark_overlays(combined_marks);
        if !body.thread_overlay_triangles().is_empty() {
            named.set_thread_overlays(body.thread_overlay_triangles().to_vec());
        }
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn mark_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Hole
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
