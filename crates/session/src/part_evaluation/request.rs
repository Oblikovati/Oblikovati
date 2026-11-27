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
impl<Body, Failure> CompletedPartSketchRecompute<Body, Failure> {
    pub(crate) fn inspection_datums<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, PartEvaluationError>
    {
        self.part.validate(session)?;
        Ok(self.part.evaluation.datum_evaluation())
    }
}
/// A sealed completed receipt. Only its session can publish it into result storage.
pub struct CompletedPartEvaluation<Body, Failure> {
    pub(super) document: u64,
    pub(super) scope: EvaluationScope,
    pub(super) evaluation: PartFeatureEvaluation<Body, Failure>,
}
impl<Body, Failure> CompletedPartEvaluation<Body, Failure> {
    pub(super) fn scope(&self) -> &EvaluationScope {
        &self.scope
    }
    pub(super) fn validate<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<(), PartEvaluationError> {
        super::owner::validate_current(self.document, self.scope(), session)
    }
    pub(super) fn validate_owner<O: TransactionObserver>(
        &self,
        session: &DocumentSession<O>,
    ) -> Result<(), PartEvaluationError> {
        super::owner::validate_owner(self.document, self.scope(), session)
    }
}
