// SPDX-License-Identifier: GPL-2.0-only
//! Replay appended features against a document-owned named part.
use crate::part_evaluation::{CapturedPartCandidate, PartCandidatePrefix};
use oblikovati_api::Error;
use oblikovati_martensite::{
    CapturedNamedSolid, CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid,
    PrismResolutionPolicy,
};
use oblikovati_model::{
    feature::FeatureId,
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::DocumentGuid,
};

/// An appended part candidate sealed to its source history and generation.
pub type OwnedPartCandidate = CapturedPartCandidate<CapturedNamedSolid, EvaluationFailure>;

/// A current named-part prefix bound to one document and resolution policy.
pub struct OwnedPartReplay {
    guid: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
    prefix: PartCandidatePrefix<NamedSolid, EvaluationFailure>,
}
impl OwnedPartReplay {
    pub(crate) fn new(
        guid: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        prefix: PartCandidatePrefix<NamedSolid, EvaluationFailure>,
    ) -> Self {
        Self {
            guid,
            policy,
            prefix,
        }
    }
    /// Check the persistent identity; example: `replay.owns_identity(guid)`.
    pub fn owns_identity(&self, guid: DocumentGuid) -> bool {
        self.guid == guid
    }
    /// Read the retained resolution policy; example: `replay.policy()`.
    pub fn policy(&self) -> Option<PrismResolutionPolicy> {
        self.policy
    }
    /// Replay and capture a safe append; example: `replay.prepare(history, candidate, feature)?`.
    pub fn prepare(
        &self,
        history: &DocumentEditHistory,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<Option<(usize, OwnedPartCandidate)>, Error> {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(history, self.guid, self.policy);
        let Some(result) = self
            .prefix
            .prepare_tail(history, candidate, feature, &mut evaluator)
            .map_err(Error)?
        else {
            return Ok(None);
        };
        let count = result.body_count();
        let captured = result
            .capture(history, |_, body| body.capture_detached(history, self.guid))
            .map_err(Error)?;
        Ok(Some((count, captured)))
    }
    /// Validate a safe append without capturing it; example: `replay.validate(history, candidate, feature)?`.
    pub fn validate(
        &self,
        history: &DocumentEditHistory,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<Option<usize>, Error> {
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(history, self.guid, self.policy);
        self.prefix
            .validate_tail(history, candidate, feature, &mut evaluator)
            .map_err(Error)
    }
}
