// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{
    TransactionObserver,
    part_evaluation::{
        CompletedPartEvaluation, CompletedPartSketchRecompute, PartEvaluationError,
        PartEvaluationRequest, PartEvaluationSlot, PartSketchAdoptionError,
    },
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Publish current recomputed sketches without a new undo step.
    /// Example: `files.commands(id)?.adopt_recomputed_part_sketches(completed)?`.
    pub fn adopt_recomputed_part_sketches<Body, Failure>(
        &mut self,
        completed: CompletedPartSketchRecompute<Body, Failure>,
    ) -> Result<bool, PartSketchAdoptionError> {
        self.session.adopt_recomputed_part_sketches(completed)
    }
    /// Adopt an unchanged replay, or refresh sketches for another replay.
    /// Example: `files.commands(id)?.adopt_sketch_replay(completed, &mut slot)?`.
    pub fn adopt_sketch_replay<Body, Failure>(
        &mut self,
        completed: CompletedPartSketchRecompute<Body, Failure>,
        slot: &mut PartEvaluationSlot<Body, Failure>,
    ) -> Result<bool, PartSketchAdoptionError> {
        self.session.adopt_sketch_replay(completed, slot)
    }
    /// Capture from a fixed file-owned document without keeping the borrowed capability.
    /// Example: `files.commands(id)?.prepare_part_evaluation()` before kernel work.
    pub fn prepare_part_evaluation(&self) -> PartEvaluationRequest {
        self.session.prepare_part_evaluation()
    }
    /// Publish a derived result through its existing owner without creating an undo command.
    /// Example: `files.commands(id)?.adopt_part_evaluation(completed, &mut slot)?`.
    pub fn adopt_part_evaluation<Body, Failure>(
        &mut self,
        completed: CompletedPartEvaluation<Body, Failure>,
        slot: &mut PartEvaluationSlot<Body, Failure>,
    ) -> Result<(), PartEvaluationError> {
        self.session.adopt_part_evaluation(completed, slot)
    }
}
