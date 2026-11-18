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
/// Solved sketches, their derived part result, and the same generation's checkpoints.
pub struct CompletedPartSketchCheckpoints<Body, Failure> {
    part: CompletedPartSketchRecompute<Body, Failure>,
    checkpoints: CompletedPartCheckpoints<Body, Failure>,
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
/// Private replayed boundaries still bound to the live source generation.
pub struct PartCheckpointCandidate<Body, Failure> {
    document: u64,
    source: EvaluationScope,
    candidate: DocumentSnapshot,
    checkpoints: PartReplayCheckpoints<Body, Failure>,
}
/// Detached boundaries awaiting an accepted document transition.
pub struct CapturedPartCheckpointCandidate<Body, Failure> {
    transfer:
        oblikovati_model::feature::part::CapturedPartTransfer<PartReplayCheckpoints<Body, Failure>>,
}
impl super::request::PartEvaluationRequest {
    /// Solve sketches and capture matching checkpoints in a single ordered replay.
    /// Example: `request.recompute_sketches_with_checkpoints(&mut backend, options)?`.
    pub fn recompute_sketches_with_checkpoints<B: PartFeatureEvaluator>(
        self,
        backend: &mut B,
        options: oblikovati_solve::nonlinear::SolveOptions,
    ) -> Result<CompletedPartSketchCheckpoints<B::Body, B::Failure>, PartEvaluationError>
    where
        B::Body: Clone,
        B::Failure: Clone,
    {
        let scope = self.snapshot.scope().clone();
        let (checkpoints, sketches) = self
            .snapshot
            .snapshot()
            .recompute_part_with_checkpoints_cancellable(backend, options, || scope.is_superseded())
            .map_err(PartEvaluationError::Replay)?;
        let evaluation = checkpoints.result().clone();
        let part = CompletedPartSketchRecompute {
            part: CompletedPartEvaluation {
                document: self.document,
                scope: scope.clone(),
                evaluation,
            },
            sketches,
        };
        Ok(CompletedPartSketchCheckpoints {
            part,
            checkpoints: CompletedPartCheckpoints {
                document: self.document,
                scope,
                checkpoints,
            },
        })
    }
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
impl<O: TransactionObserver> DocumentSession<O> {
    /// Adopt matching sketches, result, and checkpoints; changed sketches require a new replay.
    /// Example: `session.adopt_checkpointed_sketch_replay(completed, &mut result, &mut checkpoints)?`.
    pub fn adopt_checkpointed_sketch_replay<Body, Failure>(
        &mut self,
        completed: CompletedPartSketchCheckpoints<Body, Failure>,
        result: &mut PartEvaluationSlot<Body, Failure>,
        checkpoints: &mut PartCheckpointSlot<Body, Failure>,
    ) -> Result<bool, PartSketchAdoptionError> {
        completed
            .checkpoints
            .validate(self)
            .map_err(PartSketchAdoptionError::Evaluation)?;
        if let Some(previous) = &checkpoints.published {
            previous
                .validate_owner(self)
                .map_err(PartSketchAdoptionError::Evaluation)?;
        }
        let published = self.adopt_sketch_replay(completed.part, result)?;
        if published {
            checkpoints
                .adopt(self, completed.checkpoints)
                .map_err(PartSketchAdoptionError::Evaluation)?;
        }
        Ok(published)
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
    /// Replay an edited tail and retain all resulting boundaries before commit.
    /// Example: `prefix.prepare_checkpoints(history, candidate, &mut backend)?`.
    pub fn prepare_checkpoints<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &self,
        history: &DocumentEditHistory,
        candidate: &DocumentSnapshot,
        backend: &mut B,
    ) -> Result<Option<PartCheckpointCandidate<Body, Failure>>, String>
    where
        Body: Clone,
    {
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
        let checkpoints =
            self.published
                .checkpoints
                .replay_with_checkpoints(candidate, backend, || {
                    self.published.scope.is_superseded()
                })?;
        if !self.published.scope.is_current(history) {
            return Err("part checkpoints: expected an unchanged owner after replay".into());
        }
        Ok(Some(PartCheckpointCandidate {
            document: self.published.document,
            source: self.published.scope.clone(),
            candidate: candidate.clone(),
            checkpoints,
        }))
    }
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
impl<Body, Failure: Clone> PartCheckpointCandidate<Body, Failure> {
    /// Resolve a feature to its live descendant; example: `candidate.has_current_source(id)`.
    pub fn has_current_source(&self, feature: oblikovati_model::feature::FeatureId) -> bool {
        self.checkpoints.result().body_from_source(feature).is_ok()
    }
    /// Capture every boundary while its source generation is current.
    /// Example: `candidate.capture(history, |id, body| capture(id, body))?`.
    pub fn capture<Mapped>(
        &self,
        history: &DocumentEditHistory,
        capture: impl FnMut(oblikovati_model::feature::FeatureId, &Body) -> Result<Mapped, String>,
    ) -> Result<CapturedPartCheckpointCandidate<Mapped, Failure>, String> {
        if !self.source.is_current(history) {
            return Err("part checkpoint capture: expected its current owning history".into());
        }
        let checkpoints = self.checkpoints.try_map_bodies(capture)?;
        if !self.source.is_current(history) {
            return Err("part checkpoint capture: expected unchanged source after capture".into());
        }
        Ok(CapturedPartCheckpointCandidate {
            transfer: oblikovati_model::feature::part::CapturedPartTransfer::capture_checkpoints(
                self.document,
                &self.source,
                &self.candidate,
                checkpoints,
                history,
            )?,
        })
    }
}
impl<Body, Failure> CapturedPartCheckpointCandidate<Body, Failure> {
    /// Move the sealed boundaries to a kernel adapter; example: `candidate.into_transfer()`.
    pub fn into_transfer(
        self,
    ) -> oblikovati_model::feature::part::CapturedPartTransfer<PartReplayCheckpoints<Body, Failure>>
    {
        self.transfer
    }
    /// Restore captured boundaries only for their accepted edit and current inputs.
    /// The mapper must validate its own kernel bodies and names.
    /// Example: `candidate.try_complete(&transition, &session, restore)?`.
    pub fn try_complete<O: TransactionObserver, Mapped>(
        self,
        transition: &oblikovati_model::history::EvaluationTransition,
        session: &DocumentSession<O>,
        mut restore: impl FnMut(
            oblikovati_model::feature::FeatureId,
            Body,
            &DocumentEditHistory,
        ) -> Result<Mapped, String>,
    ) -> Result<CompletedPartCheckpoints<Mapped, Failure>, String> {
        let completed = self.transfer.complete(
            session.document,
            transition,
            &session.history,
            |checkpoints, history| {
                checkpoints.try_map_owned_bodies(|id, body| restore(id, body, history))
            },
        )?;
        session.complete_checkpoint_transfer(completed)
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Seal completed model checkpoints to this session before publication.
    /// Example: `session.complete_checkpoint_transfer(completed)?`.
    pub fn complete_checkpoint_transfer<Body, Failure>(
        &self,
        completed: oblikovati_model::feature::part::CompletedPartTransfer<
            PartReplayCheckpoints<Body, Failure>,
        >,
    ) -> Result<CompletedPartCheckpoints<Body, Failure>, String> {
        let checkpoints = completed.into_current(self.document, &self.history)?;
        Ok(CompletedPartCheckpoints {
            document: self.document,
            scope: self.history.evaluation_scope(),
            checkpoints,
        })
    }
    /// Publish an accepted checkpoint set and its final part result together.
    /// Example: `session.adopt_replayed_checkpoints(completed, &mut part, &mut checkpoints)?`.
    pub fn adopt_replayed_checkpoints<Body: Clone, Failure: Clone>(
        &mut self,
        completed: CompletedPartCheckpoints<Body, Failure>,
        part: &mut PartEvaluationSlot<Body, Failure>,
        checkpoints: &mut PartCheckpointSlot<Body, Failure>,
    ) -> Result<(), PartEvaluationError> {
        completed.validate(self)?;
        if let Some(previous) = &part.published {
            previous.validate_owner(self)?;
        }
        if let Some(previous) = &checkpoints.published {
            previous.validate_owner(self)?;
        }
        let result = CompletedPartEvaluation {
            document: completed.document,
            scope: completed.scope.clone(),
            evaluation: completed.checkpoints.result().clone(),
        };
        part.published = Some(result);
        checkpoints.published = Some(completed);
        Ok(())
    }
}

impl<Body, Failure: Clone> PartCandidateResult<'_, Body, Failure> {
    /// Copy a private preview while retaining the source generation guard.
    /// Example: `candidate.copy_current(history)?` before rendering.
    pub fn copy_current(
        &self,
        history: &oblikovati_model::history::DocumentEditHistory,
    ) -> Result<PartFeatureEvaluation<Body, Failure>, String>
    where
        Body: Clone,
    {
        if !self.source.is_current(history) {
            return Err("part candidate preview: expected its current owning history".into());
        }
        let evaluation = self
            .replay
            .try_map_bodies(|_, body| Ok::<_, String>(body.clone()))?;
        if !self.source.is_current(history) {
            return Err("part candidate preview: expected an unchanged owner after copying".into());
        }
        Ok(evaluation)
    }
}
