// SPDX-License-Identifier: GPL-2.0-only
//! Carry evaluated candidate inputs and results across one accepted edit.
use super::{PartFeatureEvaluation, PartReplayCheckpoints};
use crate::history::{
    DocumentEditHistory, DocumentSnapshot, EvaluationScope, EvaluationTransition,
    PartCandidateGuard,
};

/// An evaluated candidate awaiting its exact accepted edit.
pub struct CapturedPartTransfer<Payload> {
    document: u64,
    guard: PartCandidateGuard,
    payload: Payload,
}
/// A completed transfer that remains bound to its current document generation.
pub struct CompletedPartTransfer<Payload> {
    document: u64,
    scope: EvaluationScope,
    payload: Payload,
}
impl<Body, Failure> CapturedPartTransfer<PartFeatureEvaluation<Body, Failure>> {
    /// Seal evaluated bodies to their actual candidate inputs and source generation.
    /// Example: `CapturedPartTransfer::capture_result(id, &scope, &candidate, result, &history)?`.
    pub fn capture_result(
        document: u64,
        source: &EvaluationScope,
        candidate: &DocumentSnapshot,
        evaluation: PartFeatureEvaluation<Body, Failure>,
        history: &DocumentEditHistory,
    ) -> Result<Self, String> {
        validate_result_inputs(&evaluation, candidate)?;
        Ok(Self {
            document,
            guard: PartCandidateGuard::capture(source, candidate, history)?,
            payload: evaluation,
        })
    }
    /// Count live captured bodies; example: `capture.body_count()`.
    pub fn body_count(&self) -> usize {
        self.payload.ordered_bodies().count()
    }
}
impl<Body, Failure> CapturedPartTransfer<PartReplayCheckpoints<Body, Failure>> {
    /// Seal feature boundaries to their actual candidate inputs and source generation.
    /// Example: `CapturedPartTransfer::capture_checkpoints(id, &scope, &candidate, checkpoints, &history)?`.
    pub fn capture_checkpoints(
        document: u64,
        source: &EvaluationScope,
        candidate: &DocumentSnapshot,
        checkpoints: PartReplayCheckpoints<Body, Failure>,
        history: &DocumentEditHistory,
    ) -> Result<Self, String> {
        validate_result_inputs(checkpoints.result(), candidate)?;
        Ok(Self {
            document,
            guard: PartCandidateGuard::capture(source, candidate, history)?,
            payload: checkpoints,
        })
    }
}
fn validate_result_inputs<Body, Failure>(
    evaluation: &PartFeatureEvaluation<Body, Failure>,
    candidate: &DocumentSnapshot,
) -> Result<(), String> {
    let source = evaluation
        .source
        .as_ref()
        .ok_or("candidate result: expected evaluated inputs")?;
    if source.part_replay_change(candidate).is_some() {
        return Err("candidate result: expected matching evaluated inputs".into());
    }
    Ok(())
}
impl<Payload> CapturedPartTransfer<Payload> {
    /// Check the accepted edit and candidate inputs; example: `capture.matches(&edit, &history)`.
    pub fn matches(
        &self,
        transition: &EvaluationTransition,
        history: &DocumentEditHistory,
    ) -> bool {
        self.guard.matches(transition, history)
    }
    /// Complete an accepted candidate while its owning document remains current.
    /// Example: `capture.complete(document, &edit, &history, restore)?`.
    pub fn complete<Mapped>(
        self,
        document: u64,
        transition: &EvaluationTransition,
        history: &DocumentEditHistory,
        restore: impl FnOnce(Payload, &DocumentEditHistory) -> Result<Mapped, String>,
    ) -> Result<CompletedPartTransfer<Mapped>, String> {
        if self.document != document || !self.guard.matches(transition, history) {
            return Err(format!(
                "candidate document {}: expected accepted edit for document {document}",
                self.document
            ));
        }
        let payload = restore(self.payload, history)?;
        if !self.guard.matches(transition, history) {
            return Err("candidate transfer: expected unchanged inputs after restore".into());
        }
        Ok(CompletedPartTransfer {
            document,
            scope: history.evaluation_scope(),
            payload,
        })
    }
}
impl<Payload> CompletedPartTransfer<Payload> {
    /// Take a completed result only while its document generation remains current.
    /// Example: `completed.into_current(document, &history)?` before publication.
    pub fn into_current(
        self,
        document: u64,
        history: &DocumentEditHistory,
    ) -> Result<Payload, String> {
        if self.document != document || !self.scope.is_current(history) {
            return Err(format!(
                "completed candidate document {}: expected current document {document}",
                self.document
            ));
        }
        Ok(self.payload)
    }
}
