// SPDX-License-Identifier: GPL-2.0-only
//! Retain ordered replay state before each feature without reevaluating its prefix.
use super::{
    BorrowedPartReplay, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    PartFeatureProgramError,
};
use crate::feature::{FeatureHealthRecord, FeatureReplayController};
use crate::history::DocumentSnapshot;
use std::borrow::Cow;
use std::sync::Arc;

/// Complete prefix states from one replay; the owner must guard document and policy freshness.
pub struct PartReplayCheckpoints<Body, Failure> {
    states: Vec<PartFeatureEvaluation<Body, Failure>>,
}
impl<Body, Failure> PartReplayCheckpoints<Body, Failure> {
    /// Borrow the final state; example: `checkpoints.result()` after a complete replay.
    pub fn result(&self) -> &PartFeatureEvaluation<Body, Failure> {
        self.states
            .last()
            .expect("a replay always has its empty prefix")
    }
    /// Count feature boundaries, including the empty prefix; example: `checkpoints.len()`.
    pub fn len(&self) -> usize {
        self.states.len()
    }
    /// Report whether there are no boundaries; example: `checkpoints.is_empty()`.
    pub fn is_empty(&self) -> bool {
        self.states.is_empty()
    }
}
impl<Body, Failure: Clone> PartReplayCheckpoints<Body, Failure> {
    /// Replay from the earliest changed boundary, retaining earlier bodies by reference.
    /// Example: `checkpoints.replay_candidate_tail(&edited, &mut backend)?`.
    pub fn replay_candidate_tail<'a, B: PartFeatureEvaluator<Body = Body, Failure = Failure>>(
        &'a self,
        candidate: &DocumentSnapshot,
        backend: &mut B,
    ) -> Result<BorrowedPartReplay<'a, Body, Failure>, String> {
        let change = self
            .result()
            .replay_change(candidate)
            .ok_or("part checkpoint: expected changed evaluated inputs")?;
        let checkpoint = self.states.get(change.start).ok_or_else(|| {
            format!(
                "part checkpoint {}: expected a retained prefix boundary",
                change.start
            )
        })?;
        checkpoint.replay_candidate_tail(candidate, backend)
    }
}
impl DocumentSnapshot {
    /// Evaluate once and retain each ordered prefix; example: `snapshot.evaluate_part_with_checkpoints(&mut backend)?`.
    /// Native bodies must clone as immutable handles, and the caller owns generation and policy checks.
    pub fn evaluate_part_with_checkpoints<B: PartFeatureEvaluator>(
        &self,
        backend: &mut B,
    ) -> Result<PartReplayCheckpoints<B::Body, B::Failure>, PartFeatureProgramError>
    where
        B::Body: Clone,
        B::Failure: Clone,
    {
        self.evaluate_part_with_checkpoints_cancellable(backend, || false)
    }
    /// Discard all checkpoints when the owning generation is cancelled.
    /// Example: `snapshot.evaluate_part_with_checkpoints_cancellable(&mut backend, || superseded())?`.
    pub fn evaluate_part_with_checkpoints_cancellable<B: PartFeatureEvaluator>(
        &self,
        backend: &mut B,
        mut cancelled: impl FnMut() -> bool,
    ) -> Result<PartReplayCheckpoints<B::Body, B::Failure>, PartFeatureProgramError>
    where
        B::Body: Clone,
        B::Failure: Clone,
    {
        let sequence =
            super::body_dependencies::with_replacement_reads(self.effective_work_geometry())?;
        let mut controller = FeatureReplayController::new_with_parameters(
            &sequence,
            self.part_replay_range()?,
            vec![],
            &self.parameters,
        )
        .map_err(PartFeatureProgramError::Reads)?;
        let mut evaluation = PartFeatureEvaluation::empty();
        let source = Arc::new(self.clone());
        let mut states = vec![copy_state(&evaluation, &[], &source)];
        let mut candidate = Cow::Borrowed(self);
        evaluation.replay_observed(
            &mut candidate,
            backend,
            &mut controller,
            None,
            &mut cancelled,
            &mut |state, health| states.push(copy_state(state, health, &source)),
        )?;
        evaluation.refresh_sketch_faces(&candidate, backend);
        let health = controller
            .finish()
            .map_err(PartFeatureProgramError::Reads)?;
        if cancelled() {
            return Err(PartFeatureProgramError::Cancelled {
                completed: health.len(),
            });
        }
        *states.last_mut().expect("empty prefix exists") =
            copy_state(&evaluation, &health, &source);
        Ok(PartReplayCheckpoints { states })
    }
}
fn copy_state<Body: Clone, Failure: Clone>(
    state: &PartFeatureEvaluation<Body, Failure>,
    health: &[FeatureHealthRecord<PartFeatureFailure<Failure>>],
    source: &Arc<DocumentSnapshot>,
) -> PartFeatureEvaluation<Body, Failure> {
    PartFeatureEvaluation {
        datums: state.datums.clone(),
        bodies: state.bodies.clone(),
        body_lineage: state.body_lineage.clone(),
        deleted: state.deleted.clone(),
        health: health.to_vec(),
        source: Some(Arc::clone(source)),
    }
}
