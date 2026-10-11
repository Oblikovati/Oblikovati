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
        policy.require_valid()?;
        for endpoint in endpoints {
            require_position(endpoint)?;
        }
        let mut report = empty_report(endpoints);
        if !settings.infer_constraints {
            return Ok(report);
        }
        snap::propose(self, &mut report, settings.auto_apply_constraints, policy);
        report.direction = direction::winner(
            self,
            report.candidate_endpoints,
            settings.constraint_priority,
            policy,
        )?;
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
