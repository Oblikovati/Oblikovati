// SPDX-License-Identifier: GPL-2.0-only
//! Construct a named Face Draft during ordered part replay (#3918).
use super::*;
use oblikovati_model::feature::draft::{DraftMode, DraftPartingTool, ResolvedDraft};

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_draft(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        input: ResolvedDraft,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let parting_plane = match &input.parting_tool {
            None => None,
            Some(DraftPartingTool::OriginPlane(plane)) => Some(plane.frame()),
            Some(DraftPartingTool::Face(face_key)) => {
                let frame = body
                    .face_plane(self.history, self.document, face_key)
                    .map_err(|message| draft_failure(feature, message))?;
                Some((frame.recipe().origin, frame.normal()))
            }
            Some(DraftPartingTool::WorkPlane(plane_id)) => {
                return Err(draft_failure(
                    feature,
                    format!("draft parting work plane {plane_id:?}: unsupported custom work plane"),
                ));
            }
        };
        // In FixedEdge (Face Draft) mode, selecting a multi-loop shelled rim face as Pull
        // Direction with an outer shell wall face cannot preserve the inner shell wall thickness
        // cleanly; verify loop topology if the pull face is a hollow rim on a shelled body.
        if input.mode == DraftMode::FixedEdge
            && let Ok(loops) =
                body.face_loop_count(self.history, self.document, &input.pull_direction)
            && loops > 1
        {
            return Err(draft_failure(
                feature,
                format!(
                    "face draft creation failed on multi-loop shelled pull-direction rim ({loops} loops); errors during the part update"
                ),
            ));
        }
        let request = blend::DraftRequest {
            mode: input.mode,
            pull_direction: &input.pull_direction,
            parting_plane,
            faces: &input.faces,
            angle_deg: input.angle_deg,
            angle_mode: input.angle_mode,
            direction_flipped: input.direction_flipped,
        };
        let candidate = match body {
            NamedSolid::Prism(body) => body.draft_candidate(self.history, self.document, &request),
            NamedSolid::Derived(body) => {
                body.draft_candidate(self.history, self.document, &request)
            }
        }
        .map_err(|message| draft_failure(feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| draft_failure(feature, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn draft_failure(feature: FeatureId, message: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Shell
            .refused(Kind::InvalidResult, Entity::Body, message)
            .for_feature(feature),
    )
}
