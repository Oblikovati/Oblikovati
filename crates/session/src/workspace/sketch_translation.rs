// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchTransformProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityId, SketchError, SketchSelectionTransform},
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
        project: impl FnOnce(SketchTransformProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.transform_sketch_entities_projected(
            sketch,
            entities,
            SketchSelectionTransform::Move(vector),
            project,
        )
    }
    /// Prepare a rigid edit response before adoption; example: `session.transform_sketch_entities_projected(sketch, &entities, transform, |result| result)?;`.
    pub fn transform_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        transform: SketchSelectionTransform,
        project: impl FnOnce(SketchTransformProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .transform_sketch_entities_projected(sketch, entities, transform, project)
    }
}
