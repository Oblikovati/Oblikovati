// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSession, PartEvaluationError, TransactionObserver};
use oblikovati_model::{
    feature::part::{PartFeatureEvaluation, SketchPrismEvaluator},
    history::{EvaluationScope, HistoryEvaluationSnapshot},
};
/// A request owns immutable inputs and cannot keep the source history alive.
pub struct PartEvaluationRequest {
    pub(super) document: u64,
    pub(super) snapshot: HistoryEvaluationSnapshot,
}
impl PartEvaluationRequest {
    /// Poll supersession or close without borrowing the session; example: `request.is_superseded()` while queued.
    pub fn is_superseded(&self) -> bool {
        self.snapshot.scope().is_superseded()
    }
    /// Compute one complete generation through the model-owned backend interface.
    /// Example: `let completed = request.evaluate(&mut backend)?;` consumes the request, including cancellation.
    pub fn evaluate<B: SketchPrismEvaluator>(
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
