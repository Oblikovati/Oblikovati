// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    CommandError, DocumentSession, TransactionObserver, sketch_recompute::SketchUpdateMode,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentSketchDeleteError, DocumentSketchDeleteReport},
};

/// Borrow deletion provenance or inspect a decline before history adoption; example: `let reply: SketchDeleteProjectionInput<'_> = Ok(&report);`.
pub type SketchDeleteProjectionInput<'report> =
    Result<&'report DocumentSketchDeleteReport, DocumentSketchDeleteError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Delete and clean construction planes in one history step; example: `session.delete_sketch(id)?`.
    pub fn delete_sketch(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<DocumentSketchDeleteReport, CommandError<DocumentSketchDeleteError>> {
        self.delete_sketch_projected(sketch, |report| report.cloned())
    }
    /// Prepare the reply before adopting deletion and cleanup; example: `session.delete_sketch_projected(id, |report| report.cloned())?`.
    pub fn delete_sketch_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        project: impl FnOnce(SketchDeleteProjectionInput<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run_with_recompute(
            "Delete sketch",
            |snapshot, _| {
                let report = snapshot.delete_sketch(sketch);
                project(report.as_ref().map_err(Clone::clone))
            },
            SketchUpdateMode::Explicit,
        )
    }
}
