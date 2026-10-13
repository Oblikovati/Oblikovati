// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchCopyError, SketchCopyReport, SketchEntityId},
};
/// A complete geometry copy or a typed decline before publication.
/// Example: `let reply: SketchCopyProjectionInput = Ok(report);`.
pub type SketchCopyProjectionInput = Result<SketchCopyReport, SketchCopyError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Copy geometry through history and recompute, retaining consumed identity reservations.
    /// Example: `session.copy_sketch_entities(sketch, &[line], [2.0, 0.0])?;`.
    pub fn copy_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<SketchCopyReport, CommandError<SketchCopyError>> {
        self.copy_sketch_entities_projected(sketch, entities, vector, |result| result)
    }
    /// Prepare a response before adoption; response errors discard geometry, never reservations.
    /// Example: `session.copy_sketch_entities_projected(sketch, &[line], [2.0, 0.0], |result| encode(result?))?;`.
    pub fn copy_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
        project: impl FnOnce(SketchCopyProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Copy sketch entities", |snapshot, reservations| {
            project(snapshot.copy_sketch_entities(sketch, entities, vector, reservations))
        })
    }
}
