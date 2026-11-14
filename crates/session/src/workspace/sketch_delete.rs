// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchDeleteProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentSketchDeleteError, DocumentSketchDeleteReport},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Delete this document's unused sketch and orphan construction chain; example: `commands.delete_sketch(id)?`.
    pub fn delete_sketch(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<DocumentSketchDeleteReport, CommandError<DocumentSketchDeleteError>> {
        self.session.delete_sketch(sketch)
    }
    /// Prepare the canonical reply before owner adoption; example: `commands.delete_sketch_projected(id, |report| report.cloned())?`.
    pub fn delete_sketch_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        project: impl FnOnce(SketchDeleteProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session.delete_sketch_projected(sketch, project)
    }
}
