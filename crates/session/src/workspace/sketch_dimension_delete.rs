// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchDimensionDeleteProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Delete a dimension through its document session without changing workspace focus.
    /// Example: `commands.delete_sketch_dimension(sketch, constraint)?;`.
    pub fn delete_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
    ) -> Result<usize, CommandError<SketchError>> {
        self.session.delete_sketch_dimension(sketch, constraint)
    }
    /// Prepare the canonical reply before the transaction adopts the cascade.
    /// Example: `commands.delete_sketch_dimension_projected(sketch, constraint, |reply| encode(reply?))?;`.
    pub fn delete_sketch_dimension_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
        project: impl FnOnce(SketchDimensionDeleteProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .delete_sketch_dimension_projected(sketch, constraint, project)
    }
}
