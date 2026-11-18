// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSession, PartEvaluationError, TransactionObserver};
use oblikovati_model::{
    feature::part::{PartFeatureEvaluation, PartFeatureEvaluator},
    history::{EvaluationScope, HistoryEvaluationSnapshot},
};
/// A request owns immutable inputs and cannot keep the source history alive.
pub struct PartEvaluationRequest {
    pub(super) document: u64,
    pub(super) snapshot: HistoryEvaluationSnapshot,
}
impl PartEvaluationRequest {
    /// Recompute sketches and part features in dependency order without adopting them.
    /// Example: `request.recompute_sketches(&mut backend, options)?`.
    pub fn recompute_sketches<B: PartFeatureEvaluator>(
        self,
        backend: &mut B,
        options: oblikovati_solve::nonlinear::SolveOptions,
    ) -> Result<CompletedPartSketchRecompute<B::Body, B::Failure>, PartEvaluationError> {
        let scope = self.snapshot.scope().clone();
        let (evaluation, sketches) = self
            .snapshot
            .snapshot()
            .recompute_part_with_sketches_cancellable(backend, options, || scope.is_superseded())
            .map_err(PartEvaluationError::Replay)?;
        Ok(CompletedPartSketchRecompute {
            part: CompletedPartEvaluation {
                document: self.document,
                scope,
                evaluation,
            },
            sketches,
        })
    }
    /// Poll supersession or close without borrowing the session; example: `request.is_superseded()` while queued.
    pub fn is_superseded(&self) -> bool {
        self.snapshot.scope().is_superseded()
    }
    /// Compute one complete generation through the model-owned backend interface.
    /// Example: `let completed = request.evaluate(&mut backend)?;` consumes the request, including cancellation.
    pub fn evaluate<B: PartFeatureEvaluator>(
        self,
        backend: &mut B,
    ) -> Result<CompletedPartEvaluation<B::Body, B::Failure>, PartEvaluationError> {
        let scope = self.snapshot.scope().clone();
        let evaluation = self
            .snapshot
            .snapshot()
            .evaluate_part_features_cancellable(backend, || scope.is_superseded())
            .map_err(PartEvaluationError::Replay)?;
        Ok(CompletedPartEvaluation {
            document: self.document,
            scope,
            evaluation,
        })
    }
}
/// A generation-sealed candidate; only its originating session may adopt its sketches.
pub struct CompletedPartSketchRecompute<Body, Failure> {
    pub(super) part: CompletedPartEvaluation<Body, Failure>,
    pub(super) sketches: oblikovati_model::document_sketch::DocumentSketches,
}
/// A sealed completed receipt. Only its session can publish it into result storage.
pub struct CompletedPartEvaluation<Body, Failure> {
    pub(super) document: u64,
    pub(super) scope: EvaluationScope,
    pub(super) evaluation: PartFeatureEvaluation<Body, Failure>,
}
impl<Body, Failure> CompletedPartEvaluation<Body, Failure> {
    pub(super) fn validate<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<(), PartEvaluationError> {
        self.validate_owner(session)?;
        if !self.scope.is_current(&session.history) {
            return Err(PartEvaluationError::Superseded {
                document: self.document,
                supplied: self.scope.generation(),
                expected: session.generation(),
            });
        }
        Ok(())
    }
    pub(super) fn validate_owner<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<(), PartEvaluationError> {
        self.validate_document(session.document)?;
        if !self.scope.belongs_to(&session.history) {
            return Err(PartEvaluationError::ForeignHistory {
                document: self.document,
                generation: self.scope.generation(),
            });
        }
        Ok(())
    }
    fn validate_document(&self, expected: u64) -> Result<(), PartEvaluationError> {
        if self.document != expected {
            return Err(PartEvaluationError::ForeignDocument {
                supplied: self.document,
                expected,
            });
        }
        Ok(())
    }
}
