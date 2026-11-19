// SPDX-License-Identifier: GPL-2.0-only
//! Replay appended features against a document-owned named part.
use crate::{
    DocumentSession, TransactionObserver,
    part_evaluation::{CapturedPartCandidate, CompletedPartEvaluation, PartCandidatePrefix},
};
use oblikovati_api::Error;
use oblikovati_martensite::{
    CapturedNamedSolid, CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid,
    PrismResolutionPolicy,
};
use oblikovati_model::{
    feature::FeatureId,
    history::{DocumentEditHistory, DocumentSnapshot, EvaluationTransition},
    identity::DocumentGuid,
};

/// An appended candidate sealed to its document, policy, history, and generation.
pub struct OwnedPartCandidate {
    guid: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
    captured: CapturedPartCandidate<CapturedNamedSolid, EvaluationFailure>,
}
impl OwnedPartCandidate {
    pub(crate) fn complete<O: TransactionObserver>(
        self,
        document: u64,
        guid: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        session: &DocumentSession<O>,
        transition: &EvaluationTransition,
    ) -> Result<CompletedPartEvaluation<NamedSolid, EvaluationFailure>, Error> {
        if self.guid != guid || self.policy != policy {
            return Err(Error(format!(
                "document {document} policy {policy:?}: expected candidate identity {:?} and policy {:?}",
                self.guid, self.policy
            )));
        }
        let completed = oblikovati_martensite::complete_named_candidate(
            self.captured.into_transfer(),
            document,
            transition,
            session.history(),
            guid,
        )
        .map_err(Error)?;
        session.complete_part_transfer(completed).map_err(Error)
    }
}

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
        Ok(Some((
            count,
            OwnedPartCandidate {
                guid: self.guid,
                policy: self.policy,
                captured,
            },
        )))
    }
    /// Capture unchanged named bodies for a metadata-only accepted edit.
    /// Example: `replay.capture_unchanged(history, candidate)?` before committing.
    pub fn capture_unchanged(
        &self,
        history: &DocumentEditHistory,
        candidate: &DocumentSnapshot,
    ) -> Result<OwnedPartCandidate, Error> {
        let captured = self
            .prefix
            .capture_unchanged(history, candidate, |_, body| {
                body.capture_detached(history, self.guid)
            })
            .map_err(Error)?;
        Ok(OwnedPartCandidate {
            guid: self.guid,
            policy: self.policy,
            captured,
        })
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
