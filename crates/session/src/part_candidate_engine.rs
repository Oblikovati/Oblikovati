// SPDX-License-Identifier: GPL-2.0-only
//! Certified candidate replay bound to one document and its current checkpoints.
use crate::{
    DocumentSession, TransactionObserver,
    part_evaluation::{
        CapturedPartCheckpointCandidate, CompletedPartCheckpoints, PartCheckpointPrefix,
    },
};
use oblikovati_martensite::{
    CapturedNamedSolid, CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid,
    PrismResolutionPolicy,
};
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot, EvaluationTransition},
    identity::DocumentGuid,
};

/// Detached checkpoint candidate; example: pass it to `replay.complete(session, candidate, transition)`.
pub type OwnedCheckpointCandidate =
    CapturedPartCheckpointCandidate<CapturedNamedSolid, EvaluationFailure>;
type NamedPrefix = PartCheckpointPrefix<NamedSolid, EvaluationFailure>;

/// Current policy-scoped checkpoints bound to their document identity.
pub struct OwnedCheckpointReplay {
    document: u64,
    guid: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
    prefix: NamedPrefix,
}
impl OwnedCheckpointReplay {
    pub(crate) fn new(
        document: u64,
        guid: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
        prefix: NamedPrefix,
    ) -> Self {
        Self {
            document,
            guid,
            policy,
            prefix,
        }
    }
    /// Read the document that issued this capability; example: `replay.document()`.
    pub fn document(&self) -> u64 {
        self.document
    }
    /// Read the policy used by its checkpoints; example: `replay.policy()`.
    pub fn policy(&self) -> Option<PrismResolutionPolicy> {
        self.policy
    }
    /// Replay a safe changed tail on the original history.
    /// Example: `replay.prepare(candidate, history)?` before commit.
    pub fn prepare(
        &self,
        candidate: &DocumentSnapshot,
        history: &DocumentEditHistory,
    ) -> Result<Option<OwnedCheckpointCandidate>, String> {
        self.prefix.validate_history(history)?;
        if candidate.sketches.requires_part_dependency_recompute() {
            return Ok(None);
        }
        let Some(change) = history.snapshot().part_replay_change(candidate) else {
            return Ok(None);
        };
        if change.start == 0 {
            return Ok(None);
        }
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(history, self.guid, self.policy);
        let Some(candidate) =
            self.prefix
                .prepare_checkpoints(history, candidate, &mut evaluator)?
        else {
            return Ok(None);
        };
        candidate
            .capture(history, |_, body| body.capture_detached(history, self.guid))
            .map(Some)
    }
    /// Restore a captured candidate only after the exact accepted transition.
    /// Example: `replay.complete(session, candidate, &transition)?`.
    pub fn complete<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
        candidate: OwnedCheckpointCandidate,
        transition: &EvaluationTransition,
    ) -> Result<CompletedPartCheckpoints<NamedSolid, EvaluationFailure>, String> {
        let completed = oblikovati_martensite::complete_named_checkpoints(
            candidate.into_transfer(),
            self.document,
            transition,
            session.history(),
            self.guid,
        )?;
        session.complete_checkpoint_transfer(completed)
    }
}
