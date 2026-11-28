// SPDX-License-Identifier: GPL-2.0-only
//! Non-destructive Finish evaluation, exact analytic area queries, and face color overrides (#3918).
use super::blend::finish::{compute_finish_area_mm2, finish_appearance_rgba};
use super::*;
use oblikovati_model::feature::finish::ResolvedFinish;
use std::collections::BTreeMap;

impl NamedSolid {
    /// Borrow per-face linear RGBA finish appearance overrides attached to this solid.
    /// Example: `solid.finish_face_colors()`.
    pub fn finish_face_colors(&self) -> Option<&BTreeMap<Vec<u8>, [f32; 4]>> {
        match self {
            Self::Prism(_) => None,
            Self::Derived(body) => {
                let colors = body.finish_face_colors();
                (!colors.is_empty()).then_some(colors)
            }
        }
    }

    /// Compute exact analytic finish area in `mm²` for `included \ excluded` faces on this solid.
    /// Example: `solid.finish_area_mm2(&history, document, &included, &excluded)?`.
    pub fn finish_area_mm2(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        included: &[Vec<u8>],
        excluded: &[Vec<u8>],
    ) -> Result<f64, String> {
        compute_finish_area_mm2(self, history, document, included, excluded)
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_finish(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedFinish,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let effective = input.effective_faces();
        if effective.is_empty() {
            return Err(finish_failure(
                feature,
                "finish faces: expected at least one included face remaining after exclusion"
                    .into(),
            ));
        }
        for key in &input.included_faces {
            body.native_face(self.history, self.document, key)
                .map_err(|cause| {
                    finish_failure(feature, format!("missing finish face: {cause}"))
                })?;
        }
        let area_mm2 = compute_finish_area_mm2(
            body,
            self.history,
            self.document,
            &input.included_faces,
            &input.excluded_faces,
        )
        .map_err(|cause| finish_failure(feature, cause))?;
        if !area_mm2.is_finite() || area_mm2 <= 0.0 {
            return Err(finish_failure(
                feature,
                format!("finish area {area_mm2}: expected finite positive area"),
            ));
        }
        let candidate = body
            .rigid_candidate(
                self.history,
                self.document,
                &[RigidMotion::Translation([0.0, 0.0, 0.0])],
            )
            .map_err(|message| finish_failure(feature, message))?;
        let mut named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| finish_failure(feature, message))?;
        let mut colors = body.finish_face_colors().cloned().unwrap_or_default();
        let rgba = finish_appearance_rgba(&input.appearance);
        for key in effective {
            colors.insert(key.to_vec(), rgba);
        }
        named.set_finish_face_colors(colors);
        if !body.thread_overlay_triangles().is_empty() {
            named.set_thread_overlays(body.thread_overlay_triangles().to_vec());
        }
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn finish_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Hole
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
