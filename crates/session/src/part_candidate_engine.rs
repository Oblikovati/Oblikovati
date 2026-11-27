// SPDX-License-Identifier: GPL-2.0-only
//! Certified candidate replay bound to one document and its current checkpoints.
use crate::{
    DocumentSession, TransactionObserver,
    part_evaluation::{
        CapturedPartCheckpointCandidate, CompletedPartCheckpoints, PartCheckpointCandidate,
        PartCheckpointPrefix,
    },
};
use oblikovati_martensite::{
    CapturedNamedSolid, CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid,
    PrismResolutionPolicy,
};
use oblikovati_model::{
    feature::{FeatureEvaluationStatus, FeatureId, part::PartFeatureFailure},
    history::{DocumentEditHistory, DocumentSnapshot, EvaluationTransition},
    identity::DocumentGuid,
};

/// Detached checkpoint candidate; example: pass it to `replay.complete(session, candidate, transition)`.
pub type OwnedCheckpointCandidate =
    CapturedPartCheckpointCandidate<CapturedNamedSolid, EvaluationFailure>;
type NamedPrefix = PartCheckpointPrefix<NamedSolid, EvaluationFailure>;
type NamedCheckpointCandidate = PartCheckpointCandidate<NamedSolid, EvaluationFailure>;
type NamedFeatureStatus = FeatureEvaluationStatus<PartFeatureFailure<EvaluationFailure>>;

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
    /// Check the persistent identity bound to this replay; example: `replay.owns_identity(guid)`.
    pub fn owns_identity(&self, guid: DocumentGuid) -> bool {
        self.guid == guid
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
        self.prepare_projected(candidate, history, |_| Ok(()))
            .map(|result| result.map(|(_, candidate)| candidate))
            .map_err(|error| error.0)
    }
    /// Replay an edit and require a valid current descendant for the changed feature.
    /// Example: `replay.prepare_feature(candidate, history, feature)?` before commit.
    pub fn prepare_feature(
        &self,
        candidate: &DocumentSnapshot,
        history: &DocumentEditHistory,
        feature: FeatureId,
    ) -> Result<Option<OwnedCheckpointCandidate>, oblikovati_api::Error> {
        self.prepare_projected(candidate, history, |candidate| {
            if candidate.has_current_source(feature) {
                return Ok(());
            }
            Err(oblikovati_api::Error(format!(
                "feature {feature:?}: expected a valid current descendant body"
            )))
        })
        .map(|result| result.map(|(_, candidate)| candidate))
    }
    /// Capture a safe changed tail along with the queried feature's status.
    /// Example: `replay.prepare_feature_status(candidate, history, feature)?` for suppression.
    pub fn prepare_feature_status(
        &self,
        candidate: &DocumentSnapshot,
        history: &DocumentEditHistory,
        feature: FeatureId,
    ) -> Result<Option<(NamedFeatureStatus, OwnedCheckpointCandidate)>, oblikovati_api::Error> {
        self.prepare_projected(candidate, history, |candidate| {
            candidate.feature_status(feature).cloned().ok_or_else(|| {
                oblikovati_api::Error(format!(
                    "feature {feature:?}: expected a replay health record"
                ))
            })
        })
    }

    fn prepare_projected<T>(
        &self,
        candidate: &DocumentSnapshot,
        history: &DocumentEditHistory,
        project: impl FnOnce(&NamedCheckpointCandidate) -> Result<T, oblikovati_api::Error>,
    ) -> Result<Option<(T, OwnedCheckpointCandidate)>, oblikovati_api::Error> {
        self.prefix
            .validate_history(history)
            .map_err(oblikovati_api::Error)?;
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
        let Some(candidate) = self
            .prefix
            .prepare_checkpoints(history, candidate, &mut evaluator)
            .map_err(oblikovati_api::Error)?
        else {
            return Ok(None);
        };
        let value = project(&candidate)?;
        candidate
            .capture(history, |_, body| body.capture_detached(history, self.guid))
            .map(|candidate| Some((value, candidate)))
            .map_err(oblikovati_api::Error)
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
