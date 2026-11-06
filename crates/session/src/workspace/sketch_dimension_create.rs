// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchDimensionCreationProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        SketchDimensionCreation, SketchDimensionCreationError, SketchDimensionCreationReport,
    },
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Author a dimension on the owned document without changing workspace focus.
    /// Example: `commands.create_sketch_dimension(sketch, &input)?;`.
    pub fn create_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
    ) -> Result<SketchDimensionCreationReport, CommandError<SketchDimensionCreationError>> {
        self.session.create_sketch_dimension(sketch, input)
    }
    /// Prepare the reply before the shared command adopts the relation and current solve state.
    /// Example: `commands.create_sketch_dimension_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_sketch_dimension_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchDimensionCreation,
        project: impl FnOnce(SketchDimensionCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .create_sketch_dimension_projected(sketch, input, project)
    }
}
