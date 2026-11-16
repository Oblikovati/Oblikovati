// SPDX-License-Identifier: GPL-2.0-only
//! Validate fillet candidates using an isolated certified replay (#4045).
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolutionPolicy};
use oblikovati_model::{
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator},
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
    fn body(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        if edges.is_empty() {
            return Err("fillet edges []: expected at least one exact edge key".into());
        }
        let mut evaluator = CertifiedNamedSolidEvaluator::with_policy(history, document, self.0);
        let evaluated = history
            .snapshot()
            .evaluate_part_features(&mut evaluator)
            .map_err(|cause| cause.to_string())?;
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
                "fillet edges {edges:?}, matching bodies {}: expected one current body containing every edge",
                owners.len()
            )),
        }
    }
    fn validate(
        &self,
        document: DocumentGuid,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        CertifiedFilletValidation::new(document, self.0)?.validate_count(candidate, feature)
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
        self.validate_count(candidate, feature).map(|_| ())
    }
}
impl CertifiedFilletValidation {
    fn validate_count(
        &self,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<usize, String> {
        let authored = candidate
            .work_geometry
            .get(feature)
            .map_err(|cause| cause.to_string())?;
        if !matches!(
            authored.definition(),
            oblikovati_model::feature::part::PartFeatureDefinition::Fillet(_)
        ) {
            return Err(format!(
                "feature {feature:?}, kind {}: expected a fillet",
                authored.definition().kind()
            ));
        }
        let ids = RecipeIdAllocator::restore(None, candidate.parameters.ids())
            .map_err(|cause| cause.to_string())?;
        let history =
            DocumentEditHistory::new(candidate.clone(), ids).map_err(|cause| cause.to_string())?;
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_policy(&history, self.document, self.policy);
        let result = history
            .snapshot()
            .evaluate_part_features(&mut evaluator)
            .map_err(|cause| cause.to_string())?;
        if result.body(feature).is_none() {
            return Err(format!(
                "fillet {feature:?}, health {:?}: expected a valid named result",
                result.health()
            ));
        }
        Ok(result.ordered_bodies().count())
    }
}
