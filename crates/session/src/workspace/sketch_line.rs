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
}
