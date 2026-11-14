// SPDX-License-Identifier: GPL-2.0-only
//! Generation-checked adoption; retained kernel results are not undo recipe contents.
mod error;
mod request;
use crate::{DocumentSession, TransactionObserver};
pub use error::{PartEvaluationError, PartSketchAdoptionError};
use oblikovati_model::feature::part::PartFeatureEvaluation;
use oblikovati_model::{
    feature::{part::SketchPrismEvaluator, work_plane::WorkPlaneFaceReference},
    sketch::SketchPlane,
};
pub use request::{CompletedPartEvaluation, CompletedPartSketchRecompute, PartEvaluationRequest};
/// One derived result store, injected into the document owner. No graph or model mutation lives here.
pub struct PartEvaluationSlot<Body, Failure> {
    published: Option<CompletedPartEvaluation<Body, Failure>>,
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
        B: SketchPrismEvaluator<Body = Body, Failure = Failure>,
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
        B: SketchPrismEvaluator<Body = Body, Failure = Failure>,
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
            generation: published.scope.generation(),
            is_current: !published.scope.is_superseded(),
            evaluation: &published.evaluation,
        })
    }
    /// Retire retained outputs on their owner thread; example: `slot.clear()` before closing its viewport.
    pub fn clear(&mut self) {
        self.published = None;
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
