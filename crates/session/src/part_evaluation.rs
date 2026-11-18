// SPDX-License-Identifier: GPL-2.0-only
//! Generation-checked adoption; retained kernel results are not undo recipe contents.
mod checkpoints;
mod error;
mod owner;
mod request;
use crate::{DocumentSession, TransactionObserver};
pub use checkpoints::{
    CompletedPartCheckpoints, CompletedPartSketchCheckpoints, PartCheckpointPrefix,
    PartCheckpointSlot,
};
pub use error::{PartEvaluationError, PartSketchAdoptionError};
use oblikovati_model::feature::part::PartFeatureEvaluation;
use oblikovati_model::{
    feature::{part::PartFeatureEvaluator, work_plane::WorkPlaneFaceReference},
    sketch::SketchPlane,
};
pub use request::{CompletedPartEvaluation, CompletedPartSketchRecompute, PartEvaluationRequest};
/// One derived result store, injected into the document owner. No graph or model mutation lives here.
pub struct PartEvaluationSlot<Body, Failure> {
    published: Option<CompletedPartEvaluation<Body, Failure>>,
}
/// A retained prefix whose source generation is checked when it is used.
pub struct PartCandidatePrefix<'a, Body, Failure> {
    published: &'a CompletedPartEvaluation<Body, Failure>,
}
/// A private candidate replay retained until its transaction finishes.
pub struct PartCandidateResult<'a, Body, Failure> {
    document: u64,
    source: oblikovati_model::history::EvaluationScope,
    candidate: oblikovati_model::history::DocumentSnapshot,
    replay: oblikovati_model::feature::part::BorrowedPartReplay<'a, Body, Failure>,
}
/// An independently captured candidate that is not yet a published result.
pub struct CapturedPartCandidate<Body, Failure> {
    document: u64,
    source: oblikovati_model::history::EvaluationScope,
    candidate: oblikovati_model::history::DocumentSnapshot,
    evaluation: PartFeatureEvaluation<Body, Failure>,
}
impl<Body, Failure> CapturedPartCandidate<Body, Failure> {
    /// Complete a captured result only for its accepted edit and unchanged inputs.
    /// The mapper must validate its own kernel bodies and names.
    /// Example: `candidate.try_complete(&transition, &session, restore)?`.
    pub fn try_complete<O: TransactionObserver, Mapped>(
        self,
        transition: &oblikovati_model::history::EvaluationTransition,
        session: &DocumentSession<O>,
        mut restore: impl FnMut(
            oblikovati_model::feature::FeatureId,
            Body,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<Mapped, String>,
    ) -> Result<CompletedPartEvaluation<Mapped, Failure>, String> {
        if self.document != session.document
            || !self.matches_transition(transition, &session.history)
        {
            return Err(format!(
                "part candidate document {}, session {}: expected its accepted edit and current inputs",
                self.document, session.document
            ));
        }
        let evaluation = self
            .evaluation
            .try_map_owned_bodies(|id, body| restore(id, body, &session.history))?;
        if !candidate_matches_transition(
            &self.source,
            &self.candidate,
            transition,
            &session.history,
        ) {
            return Err(
                "part candidate completion: expected unchanged inputs after restore".into(),
            );
        }
        Ok(CompletedPartEvaluation {
            document: session.document,
            scope: session.history.evaluation_scope(),
            evaluation,
        })
    }
    /// Count captured live bodies; example: `captured.body_count()`.
    pub fn body_count(&self) -> usize {
        self.evaluation.ordered_bodies().count()
    }
    /// Match the committed generation and candidate inputs, without authorizing name transfer.
    /// Example: `captured.matches_transition(&transition, &history)` after the edit.
    pub fn matches_transition(
        &self,
        transition: &oblikovati_model::history::EvaluationTransition,
        history: &oblikovati_model::history::DocumentEditHistory,
    ) -> bool {
        candidate_matches_transition(&self.source, &self.candidate, transition, history)
    }
}
impl<Body, Failure> PartCandidateResult<'_, Body, Failure> {
    /// Count current bodies in the candidate; example: `result.body_count()`.
    pub fn body_count(&self) -> usize {
        self.replay.body_count()
    }
    /// Match the accepted edit and current evaluated inputs, without authorizing name transfer.
    /// Example: `result.matches_transition(&transition, &history)` after commit.
    pub fn matches_transition(
        &self,
        transition: &oblikovati_model::history::EvaluationTransition,
        history: &oblikovati_model::history::DocumentEditHistory,
    ) -> bool {
        candidate_matches_transition(&self.source, &self.candidate, transition, history)
    }
}
fn candidate_matches_transition(
    source: &oblikovati_model::history::EvaluationScope,
    candidate: &oblikovati_model::history::DocumentSnapshot,
    transition: &oblikovati_model::history::EvaluationTransition,
    history: &oblikovati_model::history::DocumentEditHistory,
) -> bool {
    transition.matches(source, history)
        && candidate.part_replay_change(history.snapshot()).is_none()
}
impl<Body, Failure: Clone> PartCandidateResult<'_, Body, Failure> {
    /// Capture all candidate bodies while their source generation remains current.
    /// Example: `candidate.capture(history, |feature, body| copy(feature, body))?`.
    pub fn capture<Mapped>(
        &self,
        history: &oblikovati_model::history::DocumentEditHistory,
        capture: impl FnMut(oblikovati_model::feature::FeatureId, &Body) -> Result<Mapped, String>,
    ) -> Result<CapturedPartCandidate<Mapped, Failure>, String> {
        if !self.source.is_current(history) {
            return Err("part candidate capture: expected its current owning history".into());
        }
        let evaluation = self.replay.try_map_bodies(capture)?;
        if !self.source.is_current(history) {
            return Err("part candidate capture: expected an unchanged owner after capture".into());
        }
        Ok(CapturedPartCandidate {
            document: self.document,
            source: self.source.clone(),
            candidate: self.candidate.clone(),
            evaluation,
        })
    }
}
impl<Body, Failure: Clone + std::fmt::Debug> PartCandidatePrefix<'_, Body, Failure> {
    /// Validate an appended candidate, or return none when the prefix is ineligible.
    /// Example: `prefix.validate_tail(history, candidate, feature, &mut backend)?`.
    pub fn validate_tail<B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &self,
        history: &oblikovati_model::history::DocumentEditHistory,
        candidate: &oblikovati_model::history::DocumentSnapshot,
        feature: oblikovati_model::feature::FeatureId,
        backend: &mut B,
    ) -> Result<Option<usize>, String> {
        Ok(self
            .prepare_tail(history, candidate, feature, backend)?
            .map(|result| result.body_count()))
    }
    /// Prepare an isolated candidate result for a later accepted transaction.
    /// Example: `prefix.prepare_tail(history, candidate, feature, &mut backend)?`.
    pub fn prepare_tail<'a, B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &'a self,
        history: &oblikovati_model::history::DocumentEditHistory,
        candidate: &oblikovati_model::history::DocumentSnapshot,
        feature: oblikovati_model::feature::FeatureId,
        backend: &mut B,
    ) -> Result<Option<PartCandidateResult<'a, Body, Failure>>, String> {
        if !self.published.scope().is_current(history) {
            return Err("part candidate prefix: expected its current owning history".into());
        }
        let evaluation = &self.published.evaluation;
        let Some(change) = evaluation.replay_change(candidate) else {
            return Ok(None);
        };
        if change.start != evaluation.health().len() {
            return Ok(None);
        }
        let result = evaluation.replay_candidate_tail(candidate, backend)?;
        if !result.has_body(feature) {
            return Err(format!(
                "feature {feature:?}, health {:?}: expected a valid candidate body",
                result.health()
            ));
        }
        if !self.published.scope().is_current(history) {
            return Err("part candidate prefix: expected an unchanged owner after replay".into());
        }
        Ok(Some(PartCandidateResult {
            document: self.published.document,
            source: self.published.scope().clone(),
            candidate: candidate.clone(),
            replay: result,
        }))
    }
}
/// Owned datum results sealed to one document history and generation.
pub struct CurrentPartDatums {
    document: u64,
    scope: oblikovati_model::history::EvaluationScope,
    datums: oblikovati_model::feature::work_geometry::WorkGeometryEvaluation,
}
impl CurrentPartDatums {
    /// Require the same current session; example: `datums.current(&session)?` before use.
    /// ```compile_fail
    /// use oblikovati_session::{DocumentSession, TransactionObserver, part_evaluation::CurrentPartDatums};
    /// fn query_then_edit<O: TransactionObserver>(datums: &CurrentPartDatums, session: &mut DocumentSession<O>) {
    ///     let current = datums.current(session).unwrap();
    ///     session.undo().unwrap();
    ///     let _ = &current.health;
    /// }
    /// ```
    pub fn current<'a, O: TransactionObserver>(
        &'a self,
        session: &'a DocumentSession<O>,
    ) -> Result<
        &'a oblikovati_model::feature::work_geometry::WorkGeometryEvaluation,
        PartEvaluationError,
    > {
        if self.document != session.document {
            return Err(PartEvaluationError::ForeignDocument {
                supplied: self.document,
                expected: session.document,
            });
        }
        if !self.scope.belongs_to(&session.history) {
            return Err(PartEvaluationError::ForeignHistory {
                document: self.document,
                generation: self.scope.generation(),
            });
        }
        if !self.scope.is_current(&session.history) {
            return Err(PartEvaluationError::Superseded {
                document: self.document,
                supplied: self.scope.generation(),
                expected: session.generation(),
            });
        }
        Ok(&self.datums)
    }
}
impl<Body, Failure> Default for PartEvaluationSlot<Body, Failure> {
    fn default() -> Self {
        Self { published: None }
    }
}
/// The latest presentation has explicit provenance and freshness; exact queries use `current`.
pub struct PartEvaluationView<'a, Body, Failure> {
    pub document: u64,
    pub generation: u64,
    pub is_current: bool,
    pub evaluation: &'a PartFeatureEvaluation<Body, Failure>,
}
impl<Body, Failure> PartEvaluationSlot<Body, Failure> {
    /// Capture a current prefix with a freshness check at each later use.
    /// Example: `slot.candidate_prefix(&session)?` before a transaction.
    pub fn candidate_prefix<'a, O: TransactionObserver>(
        &'a self,
        session: &DocumentSession<O>,
    ) -> Result<PartCandidatePrefix<'a, Body, Failure>, PartEvaluationError> {
        let published = self
            .published
            .as_ref()
            .ok_or(PartEvaluationError::Unavailable {
                document: session.document,
            })?;
        published.validate(session)?;
        Ok(PartCandidatePrefix { published })
    }
    /// Seal current datums including one prospective face support.
    /// Example: `slot.current_datums_for_face(&session, &backend, &face)?`.
    pub fn current_datums_for_face<O, B>(
        &self,
        session: &DocumentSession<O>,
        backend: &B,
        reference: &WorkPlaneFaceReference,
    ) -> Result<CurrentPartDatums, String>
    where
        O: TransactionObserver,
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    {
        let part = self.current(session).map_err(|error| error.to_string())?;
        let datums = part
            .datum_evaluation_for_face(backend, reference)
            .map_err(|error| error.to_string())?;
        Ok(CurrentPartDatums {
            document: session.document,
            scope: session.history.evaluation_scope(),
            datums,
        })
    }
    /// Seal the current part datums for a later session command.
    /// Example: `let datums = slot.current_datums(&session)?`.
    pub fn current_datums<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<CurrentPartDatums, PartEvaluationError> {
        let datums = self.current(session)?.datum_evaluation();
        Ok(CurrentPartDatums {
            document: session.document,
            scope: session.history.evaluation_scope(),
            datums,
        })
    }
    /// Resolve a face in the current body set; example: `slot.resolve_planar_face(session, backend, reference)?`.
    pub fn resolve_planar_face<O, B>(
        &self,
        session: &DocumentSession<O>,
        backend: &B,
        reference: &WorkPlaneFaceReference,
    ) -> Result<SketchPlane, String>
    where
        O: TransactionObserver,
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
    {
        let current = self.current(session).map_err(|error| error.to_string())?;
        let body = current.body(reference.source()).ok_or_else(|| {
            format!(
                "face key {:?}, source {:?}: expected an available unconsumed body",
                reference.key(),
                reference.source()
            )
        })?;
        backend.resolve_planar_face(body, reference.key())
    }
    /// Require a current result for this exact session; example: `slot.current(&session)?.body(feature_id)`.
    /// ```compile_fail
    /// use oblikovati_session::{DocumentSession, TransactionObserver, part_evaluation::PartEvaluationSlot};
    /// fn query_then_edit<O: TransactionObserver, B, F>(slot: &PartEvaluationSlot<B, F>, session: &mut DocumentSession<O>) {
    ///     let exact = slot.current(session).unwrap();
    ///     session.undo().unwrap();
    ///     let _ = exact.health();
    /// }
    /// ```
    pub fn current<'a, O: TransactionObserver>(
        &'a self,
        session: &'a DocumentSession<O>,
    ) -> Result<&'a PartFeatureEvaluation<Body, Failure>, PartEvaluationError> {
        let published = self
            .published
            .as_ref()
            .ok_or(PartEvaluationError::Unavailable {
                document: session.document,
            })?;
        published.validate(session)?;
        Ok(&published.evaluation)
    }
    /// Borrow latest presentation with a visible generation and stale flag; example: `slot.latest().map(|view| view.is_current)`.
    pub fn latest(&self) -> Option<PartEvaluationView<'_, Body, Failure>> {
        self.published.as_ref().map(|published| PartEvaluationView {
            document: published.document,
            generation: published.scope().generation(),
            is_current: !published.scope().is_superseded(),
            evaluation: &published.evaluation,
        })
    }
    /// Retire retained outputs on their owner thread; example: `slot.clear()` before closing its viewport.
    pub fn clear(&mut self) {
        self.published = None;
    }
    /// Find changes since this owner's retained recipe, without authorizing reuse.
    /// Example: `slot.replay_change(&session)?` reports a possible dirty tail.
    pub fn replay_change<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<Option<oblikovati_model::feature::FeatureReplayRange>, PartEvaluationError> {
        let published = self
            .published
            .as_ref()
            .ok_or(PartEvaluationError::Unavailable {
                document: session.document,
            })?;
        published.validate_owner(session)?;
        Ok(published.evaluation.replay_change(session.snapshot()))
    }
    /// Validate a candidate tail using only a current retained prefix.
    /// Example: `slot.preview_candidate_tail(&session, &candidate, &mut backend)?`.
    pub fn preview_candidate_tail<'a, O, B>(
        &'a self,
        session: &'a DocumentSession<O>,
        candidate: &oblikovati_model::history::DocumentSnapshot,
        backend: &mut B,
    ) -> Result<oblikovati_model::feature::part::BorrowedPartReplay<'a, Body, Failure>, String>
    where
        O: TransactionObserver,
        B: PartFeatureEvaluator<Body = Body, Failure = Failure>,
        Failure: Clone,
    {
        self.current(session)
            .map_err(|error| error.to_string())?
            .replay_candidate_tail(candidate, backend)
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Adopt recomputed sketch consequences into the open group or as a derived refresh.
    /// Example: `session.adopt_recomputed_part_sketches(completed)?` after a worker finishes.
    pub fn adopt_recomputed_part_sketches<Body, Failure>(
        &mut self,
        completed: CompletedPartSketchRecompute<Body, Failure>,
    ) -> Result<bool, PartSketchAdoptionError> {
        completed
            .part
            .validate(self)
            .map_err(PartSketchAdoptionError::Evaluation)?;
        if completed
            .sketches
            .same_evaluated_state(&self.history.snapshot().sketches)
        {
            return Ok(false);
        }
        if let Some(group) = &self.group {
            return self
                .history
                .edit_sketch_consequence::<crate::MutationError<std::convert::Infallible>>(
                    &group.token,
                    completed.sketches,
                )
                .map_err(crate::CommandError::Edit)
                .map_err(PartSketchAdoptionError::Refresh);
        }
        let changed = self
            .history
            .refresh(|snapshot| {
                snapshot.sketches = completed.sketches;
                Ok::<(), crate::MutationError<std::convert::Infallible>>(())
            })
            .map_err(crate::CommandError::Edit)
            .map_err(PartSketchAdoptionError::Refresh)?;
        if changed {
            self.sketch_continuity.refresh(&self.history);
            self.observer.changed(self.document, self.generation());
        }
        Ok(changed)
    }
    /// Publish an unchanged sketch replay, or refresh changed sketches for a new replay.
    /// Example: `session.adopt_sketch_replay(completed, &mut slot)?` returns true if published.
    pub fn adopt_sketch_replay<Body, Failure>(
        &mut self,
        completed: CompletedPartSketchRecompute<Body, Failure>,
        slot: &mut PartEvaluationSlot<Body, Failure>,
    ) -> Result<bool, PartSketchAdoptionError> {
        completed
            .part
            .validate(self)
            .map_err(PartSketchAdoptionError::Evaluation)?;
        if let Some(previous) = &slot.published {
            previous
                .validate_owner(self)
                .map_err(PartSketchAdoptionError::Evaluation)?;
        }
        if completed
            .sketches
            .same_evaluated_state(&self.history.snapshot().sketches)
        {
            self.adopt_part_evaluation(completed.part, slot)
                .map_err(PartSketchAdoptionError::Evaluation)?;
            return Ok(true);
        }
        self.adopt_recomputed_part_sketches(completed)?;
        Ok(false)
    }
    /// Capture a sealed immutable request without holding the session borrow.
    /// Example: `let request = session.prepare_part_evaluation();` before an owner-thread kernel run.
    pub fn prepare_part_evaluation(&self) -> PartEvaluationRequest {
        PartEvaluationRequest {
            document: self.document,
            snapshot: self.history.evaluation_snapshot(),
        }
    }
    /// Replace derived outputs only after checking identity and current generation.
    /// Example: `session.adopt_part_evaluation(completed, &mut slot)?`; rejection preserves the previous output.
    pub fn adopt_part_evaluation<Body, Failure>(
        &mut self,
        completed: CompletedPartEvaluation<Body, Failure>,
        slot: &mut PartEvaluationSlot<Body, Failure>,
    ) -> Result<(), PartEvaluationError> {
        completed.validate(self)?;
        if let Some(previous) = &slot.published {
            previous.validate_owner(self)?;
        }
        slot.published = Some(completed);
        Ok(())
    }
}
