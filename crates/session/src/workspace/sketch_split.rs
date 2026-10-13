// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchSplitProjectionInput, TransactionObserver};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Split in the owning document without changing workspace focus.
    /// Example: `commands.split_sketch_line_projected(sketch, &[line], [2.0, 0.0], |result| result)?;`.
    pub fn split_sketch_line_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        pick: [f64; 2],
        project: impl FnOnce(SketchSplitProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .split_sketch_line_projected(sketch, entities, pick, project)
    }
}
