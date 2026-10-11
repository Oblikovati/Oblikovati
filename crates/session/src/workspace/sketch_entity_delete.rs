// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchEntityDeleteProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchDeletionSelection, SketchEntityDeletion, SketchEntityId, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Delete a mixed editor selection through its borrowed document capability.
    /// Example: `commands.delete_sketch_selection(sketch, &selection)?;`.
    pub fn delete_sketch_selection(
        &mut self,
        sketch: DocumentSketchId,
        selection: &SketchDeletionSelection,
    ) -> Result<SketchEntityDeletion, CommandError<SketchError>> {
        self.session.delete_sketch_selection(sketch, selection)
    }
    /// Prepare the caller's response before the mixed selection is adopted.
    /// Example: `commands.delete_sketch_selection_projected(sketch, &selection, |reply| encode(reply?))?;`.
    pub fn delete_sketch_selection_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        selection: &SketchDeletionSelection,
        project: impl FnOnce(SketchEntityDeleteProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .delete_sketch_selection_projected(sketch, selection, project)
    }
    /// Delete through the borrowed document capability without changing workspace focus.
    /// Example: `commands.delete_sketch_entities(sketch, &[line])?;`.
    pub fn delete_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
    ) -> Result<SketchEntityDeletion, CommandError<SketchError>> {
        self.session.delete_sketch_entities(sketch, entities)
    }
    /// Prepare the caller's reply before adopting geometry and its dimension-source cascade.
    /// Example: `commands.delete_sketch_entities_projected(sketch, &[line], |reply| encode(reply?))?;`.
    pub fn delete_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        project: impl FnOnce(SketchEntityDeleteProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .delete_sketch_entities_projected(sketch, entities, project)
    }
}
