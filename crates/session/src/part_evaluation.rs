// SPDX-License-Identifier: GPL-2.0-only
//! Generation-checked adoption; retained kernel results are not undo recipe contents.
mod error;
mod request;
use crate::{DocumentSession, TransactionObserver};
pub use error::PartEvaluationError;
use oblikovati_model::feature::part::PartFeatureEvaluation;
pub use request::{CompletedPartEvaluation, PartEvaluationRequest};
/// One derived result store, injected into the document owner. No graph or model mutation lives here.
pub struct PartEvaluationSlot<Body, Failure> {
    published: Option<CompletedPartEvaluation<Body, Failure>>,
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
