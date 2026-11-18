// SPDX-License-Identifier: GPL-2.0-only
//! Validate fillet candidates using an isolated certified replay (#4045).
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolutionPolicy};
use oblikovati_model::{
    history::{DocumentEditHistory, SnapshotReplay},
    identity::DocumentGuid,
};

/// A document-scoped validator for private, solved candidates.
pub struct CertifiedFilletValidation {
    document: DocumentGuid,
    policy: PrismResolutionPolicy,
}
/// Document-independent policy for typed Fillet operations.
pub struct CertifiedFilletAccess(PrismResolutionPolicy);
impl CertifiedFilletAccess {
    /// Bind a validated policy; example: `CertifiedFilletAccess::new(policy)?`.
    pub fn new(policy: PrismResolutionPolicy) -> Result<Self, String> {
        Ok(Self(policy.validate()?))
    }
}
impl FilletAccess for CertifiedFilletAccess {
    fn accepts_policy(&self, policy: Option<PrismResolutionPolicy>) -> bool {
        policy == Some(self.0)
    }
    fn body(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        if edges.is_empty() {
            return Err("blend edges []: expected at least one exact edge key".into());
        }
        let mut evaluator = CertifiedNamedSolidEvaluator::with_policy(history, document, self.0);
        let evaluated = history
            .snapshot()
            .evaluate_part_features(&mut evaluator)
            .map_err(|cause| cause.to_string())?;
        resolve_blend_body(&evaluated, history, document, edges)
    }
    fn validate(
        &self,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        CertifiedFilletValidation::new(document, self.0)?
            .validate_count(candidate, feature, "fillet")
    }
    fn validate_chamfer(
        &self,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        CertifiedFilletValidation::new(document, self.0)?
            .validate_count(candidate, feature, "chamfer")
    }
}

pub(crate) fn resolve_blend_body(
    evaluated: &oblikovati_model::feature::part::PartFeatureEvaluation<
        oblikovati_martensite::NamedSolid,
        oblikovati_martensite::EvaluationFailure,
    >,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    edges: &[Vec<u8>],
) -> Result<Vec<u8>, String> {
    if edges.is_empty() {
        return Err("blend edges []: expected at least one exact edge key".into());
    }
    let mut owners = Vec::new();
    for body in evaluated.ordered_bodies() {
        let keys = body.subentity_reference_keys(history, document)?;
        if edges.iter().all(|edge| keys.contains(edge)) {
            owners.push(body.body_reference_key(history, document)?.to_vec());
        }
    }
    match owners.as_slice() {
        [body] => Ok(body.clone()),
        _ => Err(format!(
            "blend edges {edges:?}, matching bodies {}: expected one current body containing every edge",
            owners.len()
        )),
    }
}
impl CertifiedFilletValidation {
    /// Bind the document and explicit kernel policy; example: `CertifiedFilletValidation::new(document, policy)?`.
    pub fn new(document: DocumentGuid, policy: PrismResolutionPolicy) -> Result<Self, String> {
        Ok(Self {
            document,
            policy: policy.validate()?,
        })
    }
}
impl FilletValidation for CertifiedFilletValidation {
    fn validate(&self, candidate: &DocumentSnapshot, feature: FeatureId) -> Result<(), String> {
        self.validate_count(candidate, feature, "fillet")
            .map(|_| ())
    }
}
impl CertifiedFilletValidation {
    fn validate_count(
        &self,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
        expected: &str,
    ) -> Result<usize, String> {
        let authored = candidate
            .work_geometry
            .get(feature)
            .map_err(|cause| cause.to_string())?;
        if authored.definition().kind() != expected {
            return Err(format!(
                "feature {feature:?}, kind {}: expected {expected}",
                authored.definition().kind()
            ));
        }
        let history = SnapshotReplay::new(candidate).map_err(|cause| cause.to_string())?;
        let mut evaluator = CertifiedNamedSolidEvaluator::with_policy(
            history.history(),
            self.document,
            self.policy,
        );
        let result = history
            .history()
            .snapshot()
            .evaluate_part_features(&mut evaluator)
            .map_err(|cause| cause.to_string())?;
        if result.body(feature).is_none() {
            return Err(format!(
                "{expected} {feature:?}, health {:?}: expected a valid named result",
                result.health()
            ));
        }
        Ok(result.ordered_bodies().count())
    }
}
