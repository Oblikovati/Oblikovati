// SPDX-License-Identifier: GPL-2.0-only
//! Adopt accepted part candidates through their policy owner.
use super::*;

pub(super) type NamedSketchRecompute =
    oblikovati_session::part_evaluation::CompletedPartSketchRecompute<
        NamedSolid,
        EvaluationFailure,
    >;

/// A current part prefix bound to the policy that produced it.
pub struct CertifiedPartPrefix<'a> {
    pub(crate) prefix:
        oblikovati_session::part_evaluation::PartCandidatePrefix<'a, NamedSolid, EvaluationFailure>,
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
pub struct CertifiedCheckpointPrefix<'a> {
    pub(crate) prefix: oblikovati_session::part_evaluation::PartCheckpointPrefix<
        'a,
        NamedSolid,
        EvaluationFailure,
    >,
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
            candidate.candidate,
            transition,
            session,
            guid,
        )?;
        let part = self
            .slots
            .get_mut(&document)
            .ok_or_else(|| format!("document {document}: expected its retained part slot"))?;
        let checkpoints = self
            .checkpoints
            .get_mut(&document)
            .ok_or_else(|| format!("document {document}: expected its retained checkpoint slot"))?;
        files
            .commands(document)
            .map_err(|error| error.to_string())?
            .adopt_replayed_checkpoints(completed, part, checkpoints)
            .map_err(|error| error.to_string())
    }
    /// Retain feature boundaries for the current document on demand.
    /// Example: `owner.ensure_checkpoints(files, document)?` before interactive edits.
    pub fn ensure_checkpoints<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String> {
        self.current_slot(files, document)?;
        if self.checkpoint_prefix(files, document)?.is_some() {
            return Ok(());
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
        let mut slot = oblikovati_session::part_evaluation::PartCheckpointSlot::default();
        slot.adopt(session, completed)
            .map_err(|error| error.to_string())?;
        self.checkpoints.insert(document, slot);
        Ok(())
    }
    /// Borrow current checkpoints when available; example: `owner.checkpoint_prefix(files, id)?`.
    pub fn checkpoint_prefix<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &'a self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<
        Option<
            oblikovati_session::part_evaluation::PartCheckpointPrefix<
                'a,
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
        Ok(self
            .checkpoints
            .get(&document)
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
            candidate.candidate,
            transition,
            session,
            guid,
        )?;
        let slot = self
            .slots
            .get_mut(&document)
            .ok_or_else(|| format!("document {document}: expected its retained part slot"))?;
        files
            .commands(document)
            .map_err(|error| error.to_string())?
            .adopt_part_evaluation(completed, slot)
            .map_err(|error| error.to_string())?;
        self.checkpoints.remove(&document);
        Ok(())
    }
}

impl PartReplayService {
    /// Retire results for closed documents; example: `service.retain_open_documents(ids)`.
    pub fn retain_open_documents(&mut self, documents: &[u64]) {
        self.slots.retain(|id, _| documents.contains(id));
        self.checkpoints.retain(|id, _| documents.contains(id));
    }
    /// Retire all retained kernel outputs; example: `service.clear()` on policy change.
    pub fn clear(&mut self) {
        self.slots.clear();
        self.checkpoints.clear();
    }
}

impl PartReplayService {
    pub(super) fn current_slot<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<&PartEvaluationSlot<NamedSolid, EvaluationFailure>, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        if self
            .slots
            .get(&document)
            .is_none_or(|slot| slot.current(session).is_err())
        {
            self.update_part(files, document)?;
        }
        self.slots.get(&document).ok_or_else(|| {
            format!("document {document} part replay: expected an adopted current result")
        })
    }
}
