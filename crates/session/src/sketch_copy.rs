// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchCopyError, SketchCopyOperation, SketchCopyReport, SketchEntityId},
};
/// A complete geometry copy or a typed decline before publication.
/// Example: `let reply: SketchCopyProjectionInput = Ok(report);`.
pub type SketchCopyProjectionInput = Result<SketchCopyReport, SketchCopyError>;
/// A complete batch of copies or a typed decline before publication.
/// Example: `let reply: SketchCopyBatchProjectionInput = Ok(reports);`.
pub type SketchCopyBatchProjectionInput = Result<Vec<SketchCopyReport>, SketchCopyError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Preview linked translations without changing live history or identities.
    /// Example: `session.preview_associative_translations(sketch, &[circle], &[[10.0, 0.0]])?;`.
    pub fn preview_associative_translations(
        &self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        offsets: &[[f64; 2]],
    ) -> Result<crate::SketchTransformPreview, CommandError<SketchCopyError>> {
        self.preview_projected(
            |snapshot, reservations| {
                snapshot.copy_associative_translations(sketch, entities, offsets, reservations)
            },
            |snapshot, reports| {
                Ok(crate::SketchTransformPreview {
                    geometry: snapshot
                        .sketches
                        .get(sketch)
                        .map_err(SketchCopyError::Sketch)?
                        .geometry()
                        .clone(),
                    members: reports
                        .into_iter()
                        .flat_map(|report| report.copies)
                        .collect(),
                })
            },
        )
    }
    /// Publish linked translations as one undoable command after preparing its reply.
    /// Example: `session.copy_associative_translations_projected(sketch, &[circle], &[[10.0, 0.0]], |result| result)?;`.
    pub fn copy_associative_translations_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        offsets: &[[f64; 2]],
        project: impl FnOnce(SketchCopyBatchProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Copy linked sketch instances", |snapshot, reservations| {
            project(snapshot.copy_associative_translations(sketch, entities, offsets, reservations))
        })
    }
    /// Preview a batch without changing live geometry, history, or identity reservations.
    /// Example: `session.preview_selected_sketch_copies(sketch, &[line], &[operation])?;`.
    pub fn preview_selected_sketch_copies(
        &self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        operations: &[SketchCopyOperation],
    ) -> Result<crate::SketchTransformPreview, CommandError<SketchCopyError>> {
        self.preview_projected(
            |snapshot, reservations| {
                snapshot.copy_selected_sketch_entities_batch(
                    sketch,
                    entities,
                    operations,
                    reservations,
                )
            },
            |snapshot, reports| {
                Ok(crate::SketchTransformPreview {
                    geometry: snapshot
                        .sketches
                        .get(sketch)
                        .map_err(SketchCopyError::Sketch)?
                        .geometry()
                        .clone(),
                    members: reports
                        .into_iter()
                        .flat_map(|report| report.copies)
                        .collect(),
                })
            },
        )
    }
    /// Publish all copies in one history transaction after preparing the response.
    /// Example: `session.copy_selected_sketch_entities_batch_projected(sketch, &[line], &[operation], |result| result)?;`.
    pub fn copy_selected_sketch_entities_batch_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        operations: &[SketchCopyOperation],
        project: impl FnOnce(SketchCopyBatchProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Copy sketch instances", |snapshot, reservations| {
            project(snapshot.copy_selected_sketch_entities_batch(
                sketch,
                entities,
                operations,
                reservations,
            ))
        })
    }
    /// Preview a solved copy without changing the live sketch or reserving its identities.
    /// Example: `session.preview_sketch_copy(sketch, &[line], [2.0, 0.0])?;`.
    pub fn preview_sketch_copy(
        &self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<crate::SketchTransformPreview, CommandError<SketchCopyError>> {
        self.preview_selected_sketch_copy(sketch, entities, SketchCopyOperation::Translate(vector))
    }
    /// Preview any supported copy transform without changing history or reserving live identities.
    /// Example: `session.preview_selected_sketch_copy(sketch, &[line], SketchCopyOperation::Rotate { center: [0.0; 2], radians: 1.0 })?;`.
    pub fn preview_selected_sketch_copy(
        &self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        operation: SketchCopyOperation,
    ) -> Result<crate::SketchTransformPreview, CommandError<SketchCopyError>> {
        self.preview_projected(
            |snapshot, reservations| {
                snapshot.copy_selected_sketch_entities(sketch, entities, operation, reservations)
            },
            |snapshot, report| {
                Ok(crate::SketchTransformPreview {
                    geometry: snapshot
                        .sketches
                        .get(sketch)
                        .map_err(SketchCopyError::Sketch)?
                        .geometry()
                        .clone(),
                    members: report.copies,
                })
            },
        )
    }
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
        self.copy_selected_sketch_entities_projected(
            sketch,
            entities,
            SketchCopyOperation::Translate(vector),
            project,
        )
    }
    /// Prepare a transformed copy response before publication; example: `session.copy_selected_sketch_entities_projected(sketch, &[arc], SketchCopyOperation::Mirror(line), |result| result)?;`.
    pub fn copy_selected_sketch_entities_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        entities: &[SketchEntityId],
        operation: SketchCopyOperation,
        project: impl FnOnce(SketchCopyProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run(operation.label(), |snapshot, reservations| {
            project(snapshot.copy_selected_sketch_entities(
                sketch,
                entities,
                operation,
                reservations,
            ))
        })
    }
}
