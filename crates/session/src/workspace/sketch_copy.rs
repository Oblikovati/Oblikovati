// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchCopyProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchCopyError, SketchCopyReport, SketchEntityId},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Copy geometry in this document without changing workspace focus.
    /// Example: `commands.copy_sketch_entities(sketch, &[line], [2.0, 0.0])?;`.
    pub fn copy_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<SketchCopyReport, CommandError<SketchCopyError>> {
        self.session.copy_sketch_entities(sketch, entities, vector)
    }
    /// Prepare the typed reply before the shared copy command publishes its candidate.
    /// Example: `commands.copy_sketch_entities_projected(sketch, &[line], [2.0, 0.0], |result| encode(result?))?;`.
    pub fn copy_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
        project: impl FnOnce(SketchCopyProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .copy_sketch_entities_projected(sketch, entities, vector, project)
    }
}
