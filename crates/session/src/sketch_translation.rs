// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchEntityId, SketchError},
};
/// Authored defining points or an input decline before adoption.
/// Example: `let reply: SketchTranslationProjectionInput = Ok(points);`.
pub type SketchTranslationProjectionInput = Result<Vec<SketchEntityId>, SketchError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Move selected defining points through history and current constraint/projection recompute.
    /// Example: `session.translate_sketch_entities(sketch, &[line], [2.0, 3.0])?;`.
    pub fn translate_sketch_entities(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<Vec<SketchEntityId>, CommandError<SketchError>> {
        self.translate_sketch_entities_projected(sketch, entities, vector, |result| result)
    }
    /// Prepare the response before adopting coordinates; reply errors discard the candidate.
    /// Example: `session.translate_sketch_entities_projected(sketch, &[line], [2.0, 3.0], |result| encode(result?))?;`.
    pub fn translate_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
        project: impl FnOnce(SketchTranslationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Move sketch entities", |snapshot, _| {
            project(
                snapshot
                    .sketches
                    .translate_entities(sketch, entities, vector),
            )
        })
    }
}
