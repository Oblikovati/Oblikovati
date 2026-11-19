// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchLineProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchLineCreation, SketchLineCreationError, SketchLineCreationReport},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Create an inferred line on the owned document without changing workspace focus.
    /// Example: `commands.create_inferred_sketch_line(sketch, &input)?;`.
    pub fn create_inferred_sketch_line(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineCreation,
    ) -> Result<SketchLineCreationReport, CommandError<SketchLineCreationError>> {
        self.session.create_inferred_sketch_line(sketch, input)
    }
    /// Prepare a typed response before the shared command adopts geometry and relations.
    /// Example: `commands.create_inferred_sketch_line_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_inferred_sketch_line_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchLineCreation,
        project: impl FnOnce(SketchLineProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .create_inferred_sketch_line_projected(sketch, input, project)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Commit a converged line chain on this document without changing focus.
    /// Example: `commands.create_inferred_sketch_line_chain(sketch, &input)?;`.
    pub fn create_inferred_sketch_line_chain(
        &mut self,
        sketch: DocumentSketchId,
        input: &oblikovati_model::history::SketchLineChainCreation,
    ) -> Result<
        oblikovati_model::history::SketchLineChainCreationReport,
        CommandError<SketchLineCreationError>,
    > {
        self.session
            .create_inferred_sketch_line_chain(sketch, input)
    }
    /// Place one point on this document; example: `commands.create_sketch_point(id, &at)?`.
    pub fn create_sketch_point(
        &mut self,
        sketch: DocumentSketchId,
        placement: &oblikovati_model::history::SketchPointPlacement,
    ) -> Result<oblikovati_model::sketch::SketchEntityId, CommandError<SketchLineCreationError>>
    {
        self.session.create_sketch_point(sketch, placement)
    }
    /// Create one tangent arc on this document without changing focus.
    /// Example: `commands.create_tangent_arc(sketch, &input)?;`.
    pub fn create_tangent_arc(
        &mut self,
        sketch: DocumentSketchId,
        input: &oblikovati_model::history::SketchTangentArcCreation,
    ) -> Result<
        oblikovati_model::history::SketchTangentArcReport,
        CommandError<SketchLineCreationError>,
    > {
        self.session.create_tangent_arc(sketch, input)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Set session-only inference candidates; example: `commands.set_sketch_inference_scope(sketch, mode, &entities)?`.
    pub fn set_sketch_inference_scope(
        &mut self,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        mode: oblikovati_api::types::SketchInferenceScope,
        entities: &[oblikovati_model::sketch::SketchEntityId],
    ) -> Result<(), oblikovati_model::sketch::SketchError> {
        self.session
            .set_sketch_inference_scope(sketch, mode, entities)
    }
}
