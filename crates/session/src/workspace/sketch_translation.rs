// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchTranslationProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityId, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Move geometry on this document without changing focus.
    /// Example: `commands.translate_sketch_entities(sketch, &[line], [2.0, 3.0])?;`.
    pub fn translate_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<Vec<SketchEntityId>, CommandError<SketchError>> {
        self.session
            .translate_sketch_entities(sketch, entities, vector)
    }
    /// Prepare the canonical response before adopting the shared command's candidate.
    /// Example: `commands.translate_sketch_entities_projected(sketch, &[line], [2.0, 3.0], |result| encode(result?))?;`.
    pub fn translate_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
        project: impl FnOnce(SketchTranslationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .translate_sketch_entities_projected(sketch, entities, vector, project)
    }
}
