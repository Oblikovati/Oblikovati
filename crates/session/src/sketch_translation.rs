// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId, SketchError, SketchSelectionTransform},
};
/// Solved private geometry and the selected entities for a transform preview.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchTransformPreview {
    pub geometry: Sketch2D,
    pub members: Vec<SketchEntityId>,
}
/// Authored defining points or an input decline before adoption.
/// Example: `let reply: SketchTransformProjectionInput = Ok(points);`.
pub type SketchTransformProjectionInput = Result<Vec<SketchEntityId>, SketchError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Preview a transform with the same recompute as commit, without changing history.
    /// Example: `session.preview_sketch_transform(sketch, &[line], transform)?`.
    pub fn preview_sketch_transform(
        &self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        transform: SketchSelectionTransform,
    ) -> Result<SketchTransformPreview, CommandError<SketchError>> {
        self.preview_projected(
            |snapshot, _| {
                snapshot
                    .sketches
                    .transform_entities(sketch, entities, transform)
            },
            |snapshot, _| {
                Ok(SketchTransformPreview {
                    geometry: snapshot.sketches.get(sketch)?.geometry().clone(),
                    members: entities.to_vec(),
                })
            },
        )
    }
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
        self.run(transform.label(), |snapshot, _| {
            project(
                snapshot
                    .sketches
                    .transform_entities(sketch, entities, transform),
            )
        })
    }
}
