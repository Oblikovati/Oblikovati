// SPDX-License-Identifier: GPL-2.0-only
//! Session-owned generation guards for retained feature-boundary checkpoints.
use super::*;
use oblikovati_model::{
    feature::part::PartReplayCheckpoints,
    history::{DocumentEditHistory, DocumentSnapshot, EvaluationScope},
};

/// A complete checkpoint set sealed to its document and source generation.
pub struct CompletedPartCheckpoints<Body, Failure> {
    document: u64,
    scope: EvaluationScope,
    checkpoints: PartReplayCheckpoints<Body, Failure>,
}
/// An injected result store for one document's checkpoint set.
pub struct PartCheckpointSlot<Body, Failure> {
    published: Option<CompletedPartCheckpoints<Body, Failure>>,
}
impl<Body, Failure> Default for PartCheckpointSlot<Body, Failure> {
    fn default() -> Self {
        Self { published: None }
    }
}
/// A source-checked checkpoint set for a private projected edit.
pub struct PartCheckpointPrefix<'a, Body, Failure> {
    published: &'a CompletedPartCheckpoints<Body, Failure>,
}
impl super::request::PartEvaluationRequest {
    /// Evaluate a generation and seal all feature boundaries to its owner.
    /// Example: `session.prepare_part_evaluation().evaluate_checkpoints(&mut backend)?`.
    pub fn evaluate_checkpoints<B: PartFeatureEvaluator>(
        self,
        backend: &mut B,
    ) -> Result<CompletedPartCheckpoints<B::Body, B::Failure>, PartEvaluationError>
    where
        B::Body: Clone,
        B::Failure: Clone,
    {
        let scope = self.snapshot.scope().clone();
        let checkpoints = self
            .snapshot
            .snapshot()
            .evaluate_part_with_checkpoints_cancellable(backend, || scope.is_superseded())
            .map_err(PartEvaluationError::Replay)?;
        Ok(CompletedPartCheckpoints {
            document: self.document,
            scope,
            checkpoints,
        })
    }
}
impl<Body, Failure> CompletedPartCheckpoints<Body, Failure> {
    fn validate_owner<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<(), PartEvaluationError> {
        super::owner::validate_owner(self.document, &self.scope, session)
    }
    fn validate<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<(), PartEvaluationError> {
        super::owner::validate_current(self.document, &self.scope, session)
    }
}
impl<Body, Failure> PartCheckpointSlot<Body, Failure> {
    /// Adopt only current checkpoints from the same history; rejection leaves the slot intact.
    /// Example: `slot.adopt(&session, completed)?`.
    pub fn adopt<O: TransactionObserver>(
        &mut self,
        session: &DocumentSession<O>,
        completed: CompletedPartCheckpoints<Body, Failure>,
    ) -> Result<(), PartEvaluationError> {
        completed.validate(session)?;
        if let Some(previous) = &self.published {
            previous.validate_owner(session)?;
        }
        self.published = Some(completed);
        Ok(())
    }
    /// Borrow a current prefix; example: `slot.candidate_prefix(&session)?` before an edit.
    pub fn candidate_prefix<'a, O: TransactionObserver>(
        &'a self,
        session: &DocumentSession<O>,
    ) -> Result<PartCheckpointPrefix<'a, Body, Failure>, PartEvaluationError> {
        let published = self
            .published
            .as_ref()
            .ok_or(PartEvaluationError::Unavailable {
                document: session.document,
            })?;
        published.validate(session)?;
        Ok(PartCheckpointPrefix { published })
    }
}
impl<Body, Failure: Clone> PartCheckpointPrefix<'_, Body, Failure> {
    /// Replay the earliest dirty tail while the source history is still current.
    /// Example: `prefix.prepare_candidate(history, candidate, &mut backend)?`.
    pub fn prepare_candidate<'a, B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &'a self,
        history: &DocumentEditHistory,
        candidate: &DocumentSnapshot,
        backend: &mut B,
    ) -> Result<Option<PartCandidateResult<'a, Body, Failure>>, String> {
        if !self.published.scope.is_current(history) {
            return Err("part checkpoints: expected the current owning history".into());
        }
        if self
            .published
            .checkpoints
            .result()
            .replay_change(candidate)
            .is_none()
        {
            return Ok(None);
        }
        let replay = self
            .published
            .checkpoints
            .replay_candidate_tail(candidate, backend)?;
        if !self.published.scope.is_current(history) {
            return Err("part checkpoints: expected an unchanged owner after replay".into());
        }
        Ok(Some(PartCandidateResult {
            document: self.published.document,
            source: self.published.scope.clone(),
            candidate: candidate.clone(),
            replay,
        }))
    }
}
