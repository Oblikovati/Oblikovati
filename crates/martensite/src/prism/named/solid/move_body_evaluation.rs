// SPDX-License-Identifier: GPL-2.0-only
//! Replay a named rigid body replacement (#4097).
use super::*;
use oblikovati_model::feature::move_body::ResolvedBodyMotion;
impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_move_body(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        motions: &[ResolvedBodyMotion],
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let operations: Vec<_> = motions.iter().map(to_rigid_motion).collect();
        let candidate = body
            .rigid_candidate(self.history, self.document, &operations)
            .map_err(|message| {
                PartFeatureFailure::Kernel(
                    Phase::MoveBody
                        .refused(Kind::InvalidResult, Entity::Body, message)
                        .for_feature(feature),
                )
            })?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(PartFeatureFailure::BodyReference)?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn to_rigid_motion(motion: &ResolvedBodyMotion) -> crate::RigidMotion {
    match motion {
        ResolvedBodyMotion::Translation(vector) => crate::RigidMotion::Translation(*vector),
        ResolvedBodyMotion::Rotation {
            origin,
            direction,
            radians,
        } => crate::RigidMotion::Rotation {
            origin: *origin,
            direction: *direction,
            radians: *radians,
        },
    }
}
