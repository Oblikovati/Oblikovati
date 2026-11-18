// SPDX-License-Identifier: GPL-2.0-only
//! Retain ordered replay state before each feature without reevaluating its prefix.
use super::{
    BorrowedPartReplay, PartFeatureEvaluation, PartFeatureEvaluator, PartFeatureFailure,
    PartFeatureProgramError,
};
use crate::feature::FeatureHealthRecord;
use crate::history::DocumentSnapshot;
use std::sync::Arc;

/// A checkpoint set and the solved sketches that produced it.
pub type CheckpointedPart<Body, Failure> = (
    PartReplayCheckpoints<Body, Failure>,
    crate::document_sketch::DocumentSketches,
);

/// Complete prefix states from one replay; the owner must guard document and policy freshness.
pub struct PartReplayCheckpoints<Body, Failure> {
    pub(super) states: Vec<PartFeatureEvaluation<Body, Failure>>,
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
    /// Capture every boundary without changing its source or lineage.
    /// Example: `checkpoints.try_map_bodies(|feature, body| capture(feature, body))?`.
    pub fn try_map_bodies<Mapped, E>(
        &self,
        mut capture: impl FnMut(crate::feature::FeatureId, &Body) -> Result<Mapped, E>,
    ) -> Result<PartReplayCheckpoints<Mapped, Failure>, E> {
        let states = self
            .states
            .iter()
            .map(|state| state.try_map_bodies(&mut capture))
            .collect::<Result<_, _>>()?;
        Ok(PartReplayCheckpoints { states })
    }
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
impl<Body, Failure> PartReplayCheckpoints<Body, Failure> {
    /// Restore captured bodies across all boundaries as one fallible operation.
    /// Example: `captured.try_map_owned_bodies(|feature, body| restore(feature, body))?`.
    pub fn try_map_owned_bodies<Mapped, E>(
        self,
        mut restore: impl FnMut(crate::feature::FeatureId, Body) -> Result<Mapped, E>,
    ) -> Result<PartReplayCheckpoints<Mapped, Failure>, E> {
        let states = self
            .states
            .into_iter()
            .map(|state| state.try_map_owned_bodies(&mut restore))
            .collect::<Result<_, _>>()?;
        Ok(PartReplayCheckpoints { states })
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
        cancelled: impl FnMut() -> bool,
    ) -> Result<PartReplayCheckpoints<B::Body, B::Failure>, PartFeatureProgramError>
    where
        B::Body: Clone,
        B::Failure: Clone,
    {
        self.checkpoints_with_options(backend, None, cancelled)
            .map(|(checkpoints, _)| checkpoints)
    }
    /// Solve sketches in feature order and retain checkpoints for the solved inputs.
    /// Example: `snapshot.recompute_part_with_checkpoints_cancellable(&mut backend, options, || superseded())?`.
    pub fn recompute_part_with_checkpoints_cancellable<B: PartFeatureEvaluator>(
        &self,
        backend: &mut B,
        options: oblikovati_solve::nonlinear::SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<CheckpointedPart<B::Body, B::Failure>, PartFeatureProgramError>
    where
        B::Body: Clone,
        B::Failure: Clone,
    {
        self.checkpoints_with_options(backend, Some(options), cancelled)
    }
    fn checkpoints_with_options<B: PartFeatureEvaluator>(
        &self,
        backend: &mut B,
        options: Option<oblikovati_solve::nonlinear::SolveOptions>,
        mut cancelled: impl FnMut() -> bool,
    ) -> Result<CheckpointedPart<B::Body, B::Failure>, PartFeatureProgramError>
    where
        B::Body: Clone,
        B::Failure: Clone,
    {
        let empty = PartFeatureEvaluation::empty();
        let source = Arc::new(self.clone());
        let mut states = vec![copy_state(&empty, &[], &source)];
        let (evaluation, candidate) = self.replay_part_with_sketches_observed(
            backend,
            options,
            &mut cancelled,
            &mut |state, health| states.push(copy_state(state, health, &source)),
        )?;
        if cancelled() {
            return Err(PartFeatureProgramError::Cancelled {
                completed: evaluation.health().len(),
            });
        }
        let source = evaluation
            .source
            .as_ref()
            .expect("completed replay has its source");
        for state in &mut states {
            state.source = Some(Arc::clone(source));
        }
        *states.last_mut().expect("empty prefix exists") = evaluation;
        Ok((
            PartReplayCheckpoints { states },
            candidate.into_owned().sketches,
        ))
    }
}
pub(super) fn copy_state<Body: Clone, Failure: Clone>(
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
