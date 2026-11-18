// SPDX-License-Identifier: GPL-2.0-only
//! Adopt accepted part candidates through their policy owner.
use super::*;

pub(super) type NamedSketchRecompute =
    oblikovati_session::part_evaluation::CompletedPartSketchRecompute<
        NamedSolid,
        EvaluationFailure,
    >;

/// A current part prefix bound to the policy that produced it.
pub struct CertifiedPartPrefix {
    pub(crate) prefix:
        oblikovati_session::part_evaluation::PartCandidatePrefix<NamedSolid, EvaluationFailure>,
    pub(crate) policy: Option<PrismResolutionPolicy>,
}
/// A candidate sealed to the policy that produced its exact bodies.
pub struct CertifiedPartCandidate {
    pub(crate) policy: PrismResolutionPolicy,
    pub(crate) candidate: oblikovati_session::part_evaluation::CapturedPartCandidate<
        oblikovati_martensite::CapturedNamedSolid,
        EvaluationFailure,
    >,
}
/// A current checkpoint set bound to the policy that produced it.
pub struct CertifiedCheckpointPrefix {
    pub(crate) prefix:
        oblikovati_session::part_evaluation::PartCheckpointPrefix<NamedSolid, EvaluationFailure>,
    pub(crate) policy: Option<PrismResolutionPolicy>,
}
/// A replayed checkpoint set sealed to its certified resolution policy.
pub struct CertifiedCheckpointCandidate {
    pub(crate) policy: PrismResolutionPolicy,
    pub(crate) candidate: oblikovati_session::part_evaluation::CapturedPartCheckpointCandidate<
        oblikovati_martensite::CapturedNamedSolid,
        EvaluationFailure,
    >,
}

impl PartReplayService {
    pub(super) fn adopt_named_checkpoints<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        candidate: CertifiedCheckpointCandidate,
        transition: &oblikovati_model::history::EvaluationTransition,
    ) -> Result<(), String> {
        if self.resolution != Some(candidate.policy) {
            return Err(format!(
                "checkpoint policy {:?}: expected {:?}",
                candidate.policy, self.resolution
            ));
        }
        let workspace = files.workspace();
        let guid = workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let session = workspace
            .session(document)
            .map_err(|error| error.to_string())?;
        let completed = oblikovati_martensite::complete_named_checkpoints(
            candidate.candidate.into_transfer(),
            document,
            transition,
            session.history(),
            guid,
        )?;
        let completed = session.complete_checkpoint_transfer(completed)?;
        files
            .commands(document)
            .map_err(|error| error.to_string())?
            .adopt_owned_replayed_checkpoints(self.resolution, completed)
    }
    /// Retain feature boundaries for the current document on demand.
    /// Example: `owner.ensure_checkpoints(files, document)?` before interactive edits.
    pub fn ensure_checkpoints<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String> {
        if self.checkpoint_prefix(files, document)?.is_some() {
            return Ok(());
        }
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        let needs_sketch_recompute = crate::datums::requires_part_replay(session.snapshot());
        let current = self
            .slot(files, document)
            .is_some_and(|slot| slot.current(session).is_ok());
        if needs_sketch_recompute {
            self.ensure_current(files, document)?;
        }
        let policy = self
            .resolution
            .ok_or("part checkpoints: expected a certified resolution policy")?;
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|error| error.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_policy(session.history(), guid, policy);
        self.replay_count = self.replay_count.saturating_add(1);
        let completed = session
            .prepare_part_evaluation()
            .evaluate_checkpoints(&mut evaluator)
            .map_err(|error| error.to_string())?;
        let mut commands = files
            .commands(document)
            .map_err(|error| error.to_string())?;
        if current || needs_sketch_recompute {
            commands.adopt_owned_checkpoints(self.resolution, completed)?;
        } else {
            commands.adopt_owned_full_checkpoints(self.resolution, completed)?;
        }
        Ok(())
    }
    /// Borrow current checkpoints when available; example: `owner.checkpoint_prefix(files, id)?`.
    pub fn checkpoint_prefix<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<
        Option<
            oblikovati_session::part_evaluation::PartCheckpointPrefix<
                NamedSolid,
                EvaluationFailure,
            >,
        >,
        String,
    > {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        Ok(session
            .owned_checkpoint_slot(self.resolution)
            .and_then(|slot| slot.candidate_prefix(session).ok()))
    }
    pub(super) fn adopt_named_candidate<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        candidate: CertifiedPartCandidate,
        transition: &oblikovati_model::history::EvaluationTransition,
    ) -> Result<(), String> {
        if self.resolution != Some(candidate.policy) {
            return Err(format!(
                "candidate policy {:?}: expected {:?}",
                candidate.policy, self.resolution
            ));
        }
        let workspace = files.workspace();
        let guid = workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let session = workspace
            .session(document)
            .map_err(|error| error.to_string())?;
        let completed = oblikovati_martensite::complete_named_candidate(
            candidate.candidate.into_transfer(),
            document,
            transition,
            session.history(),
            guid,
        )?;
        let completed = session.complete_part_transfer(completed)?;
        files
            .commands(document)
            .map_err(|error| error.to_string())?
            .adopt_owned_part(self.resolution, completed)
    }
}

impl PartReplayService {
    pub(super) fn ensure_current<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        if self
            .slot(files, document)
            .is_none_or(|slot| slot.current(session).is_err())
        {
            self.update_part(files, document)?;
        }
        Ok(())
    }
}
