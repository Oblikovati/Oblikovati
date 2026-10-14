// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, geometry::require_position};
use crate::sketch_settings::SketchSettings;
use oblikovati_api::types::ConstraintInferenceKind;
mod direction;
mod policy;
mod snap;
pub use policy::SketchInferencePolicy;

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
    /// Propose relations for a fresh segment without altering existing recipe state.
    /// Example: `sketch.infer_line_placement([[0.0, 0.0], [4.0, 0.01]], &settings, policy)?;`.
    pub fn infer_line_placement(
        &self,
        endpoints: [[f64; 2]; 2],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
    ) -> Result<SketchLinePlacementInference, SketchError> {
        let report = self.line_point_proposals(endpoints, &[], settings, policy)?;
        self.complete_line_inference(report, settings, policy)
    }
    pub(crate) fn infer_owned_line_placement(
        &mut self,
        points: [SketchEntityId; 2],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
    ) -> Result<SketchLinePlacementInference, SketchError> {
        let endpoints = [
            self.point(points[0])?.position,
            self.point(points[1])?.position,
        ];
        let report = self.line_point_proposals(endpoints, &points, settings, policy)?;
        // Go aa8297d1 applies snaps before ranking directions, including shared reference endpoints.
        for (point, position) in points.into_iter().zip(report.candidate_endpoints) {
            self.set_position(point, position)?;
        }
        self.complete_line_inference(report, settings, policy)
    }
    fn line_point_proposals(
        &self,
        endpoints: [[f64; 2]; 2],
        excluded: &[SketchEntityId],
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
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
            );
        }
        Ok(report)
    }
    fn complete_line_inference(
        &self,
        mut report: SketchLinePlacementInference,
        settings: &SketchSettings,
        policy: SketchInferencePolicy,
    ) -> Result<SketchLinePlacementInference, SketchError> {
        if settings.infer_constraints {
            report.direction = direction::winner(
                self,
                report.candidate_endpoints,
                settings.constraint_priority,
                policy,
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
