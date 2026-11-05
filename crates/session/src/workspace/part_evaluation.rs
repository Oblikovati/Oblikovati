// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{
    TransactionObserver,
    part_evaluation::{
        CompletedPartEvaluation, PartEvaluationError, PartEvaluationRequest, PartEvaluationSlot,
    },
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
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
