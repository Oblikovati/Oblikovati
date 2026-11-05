// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::SketchDimensionEdit,
    sketch::{SketchConstraintId, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Edit dimension state through the owned document without changing workspace focus.
    /// Example: `commands.edit_sketch_dimension(sketch, dimension, &edit)?;`.
    pub fn edit_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        edit: &SketchDimensionEdit,
    ) -> Result<(), CommandError<SketchError>> {
        self.session.edit_sketch_dimension(sketch, dimension, edit)
    }
    /// Prepare the canonical reply before adoption of the dimension edit.
    /// Example: `commands.edit_sketch_dimension_projected(sketch, dimension, &edit, |reply| encode(reply?))?;`.
    pub fn edit_sketch_dimension_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        edit: &SketchDimensionEdit,
        project: impl FnOnce(Result<(), SketchError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .edit_sketch_dimension_projected(sketch, dimension, edit, project)
    }
}
