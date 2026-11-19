// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, geometry::require_position};
use crate::sketch_settings::SketchSettings;
use oblikovati_api::types::ConstraintInferenceKind;
mod direction;
mod policy;
mod scope;
mod snap;
pub use policy::SketchInferencePolicy;
pub use scope::SketchInferenceCandidates;

/// A proposed endpoint coincidence targets an existing point, never another fresh endpoint.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchEndpointInference {
    pub endpoint: usize,
    pub target: SketchEntityId,
}
/// One proposed direction relation, with the canonical kind and integer quality score.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchDirectionInference {
    pub kind: ConstraintInferenceKind,
    pub target: Option<SketchEntityId>,
    pub score: u8,
}
/// Immutable placement evidence; candidate endpoints include snaps only when constraining is on.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchLinePlacementInference {
    pub candidate_endpoints: [[f64; 2]; 2],
    pub points: Vec<SketchEndpointInference>,
    pub direction: Option<SketchDirectionInference>,
}
impl Sketch2D {
    /// Propose coincidence with an existing point under the shared interaction policy.
    /// Example: `sketch.infer_point_coincidence([0.,0.], &new_points, &settings, policy)?`.
    pub fn infer_point_coincidence(
        &self,
        position: [f64; 2],
        excluded: &[SketchEntityId],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
    ) -> Result<Option<SketchEntityId>, SketchError> {
        self.infer_point_from_candidates(
            position,
            excluded,
            settings,
            policy,
            &scope::ResolvedCandidates::all(),
        )
    }
    /// Infer point coincidence using the chosen candidate geometry.
    /// Example: `sketch.infer_point_coincidence_scoped(at, &[], &settings, policy, &scope, &[])?`.
    pub fn infer_point_coincidence_scoped(
        &self,
        position: [f64; 2],
        excluded: &[SketchEntityId],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        scope: &SketchInferenceCandidates,
        current: &[SketchEntityId],
    ) -> Result<Option<SketchEntityId>, SketchError> {
        let candidates = scope.resolve(self, current)?;
        self.infer_point_from_candidates(position, excluded, settings, policy, &candidates)
    }
    fn infer_point_from_candidates(
        &self,
        position: [f64; 2],
        excluded: &[SketchEntityId],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        candidates: &scope::ResolvedCandidates,
    ) -> Result<Option<SketchEntityId>, SketchError> {
        policy.require_valid()?;
        require_position(position)?;
        if !settings.infer_constraints {
            return Ok(None);
        }
        Ok(snap::nearest_point(
            self.points()
                .iter()
                .filter(|point| !excluded.contains(&point.id) && candidates.contains(point.id)),
            position,
            policy.snap_distance_working,
        )
        .map(|point| point.id))
    }
    /// Propose relations for a fresh segment without altering existing recipe state.
    /// Example: `sketch.infer_line_placement([[0.0, 0.0], [4.0, 0.01]], &settings, policy)?;`.
    pub fn infer_line_placement(
        &self,
        endpoints: [[f64; 2]; 2],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
    ) -> Result<SketchLinePlacementInference, SketchError> {
        let candidates = scope::ResolvedCandidates::all();
        let report = self.line_point_proposals(endpoints, &[], settings, policy, &candidates)?;
        self.complete_line_inference(report, settings, policy, &candidates)
    }
    /// Infer using only the chosen candidate geometry and current command members.
    /// Example: `sketch.infer_line_placement_scoped(points, &settings, policy, &scope, &[])?`.
    pub fn infer_line_placement_scoped(
        &self,
        endpoints: [[f64; 2]; 2],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        scope: &SketchInferenceCandidates,
        current: &[SketchEntityId],
    ) -> Result<SketchLinePlacementInference, SketchError> {
        let candidates = scope.resolve(self, current)?;
        let report = self.line_point_proposals(endpoints, &[], settings, policy, &candidates)?;
        self.complete_line_inference(report, settings, policy, &candidates)
    }
    pub(crate) fn infer_owned_line_placement(
        &mut self,
        points: [SketchEntityId; 2],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
    ) -> Result<SketchLinePlacementInference, SketchError> {
        self.infer_owned_line_from_candidates(
            points,
            settings,
            policy,
            &scope::ResolvedCandidates::all(),
        )
    }
    pub(crate) fn infer_owned_line_placement_scoped(
        &mut self,
        points: [SketchEntityId; 2],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        scope: &SketchInferenceCandidates,
        current: &[SketchEntityId],
    ) -> Result<SketchLinePlacementInference, SketchError> {
        let candidates = scope.resolve(self, current)?;
        self.infer_owned_line_from_candidates(points, settings, policy, &candidates)
    }
    fn infer_owned_line_from_candidates(
        &mut self,
        points: [SketchEntityId; 2],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        candidates: &scope::ResolvedCandidates,
    ) -> Result<SketchLinePlacementInference, SketchError> {
        let endpoints = [
            self.point(points[0])?.position,
            self.point(points[1])?.position,
        ];
        let report = self.line_point_proposals(endpoints, &points, settings, policy, candidates)?;
        // Go aa8297d1 applies snaps before ranking directions, including shared reference endpoints.
        for (point, position) in points.into_iter().zip(report.candidate_endpoints) {
            self.set_position(point, position)?;
        }
        self.complete_line_inference(report, settings, policy, candidates)
    }
    fn line_point_proposals(
        &self,
        endpoints: [[f64; 2]; 2],
        excluded: &[SketchEntityId],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        candidates: &scope::ResolvedCandidates,
    ) -> Result<SketchLinePlacementInference, SketchError> {
        policy.require_valid()?;
        for endpoint in endpoints {
            require_position(endpoint)?;
        }
        let mut report = empty_report(endpoints);
        if settings.infer_constraints {
            snap::propose(
                self,
                excluded,
                &mut report,
                settings.auto_apply_constraints,
                policy,
                candidates,
            );
        }
        Ok(report)
    }
    fn complete_line_inference(
        &self,
        mut report: SketchLinePlacementInference,
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
        candidates: &scope::ResolvedCandidates,
    ) -> Result<SketchLinePlacementInference, SketchError> {
        if settings.infer_constraints {
            report.direction = direction::winner(
                self,
                report.candidate_endpoints,
                settings.constraint_priority,
                policy,
                candidates,
            )?;
        }
        Ok(report)
    }
}

fn empty_report(endpoints: [[f64; 2]; 2]) -> SketchLinePlacementInference {
    SketchLinePlacementInference {
        candidate_endpoints: endpoints,
        points: Vec::new(),
        direction: None,
    }
}
