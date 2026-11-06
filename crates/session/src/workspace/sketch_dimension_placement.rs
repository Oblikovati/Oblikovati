// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Move an annotation through the owned document session without changing focus.
    /// Example: `commands.move_sketch_dimension(sketch, dimension, [2.0,3.0])?;`.
    pub fn move_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        position: [f64; 2],
    ) -> Result<(), CommandError<SketchError>> {
        self.session
            .move_sketch_dimension(sketch, dimension, position)
    }
    /// Prepare the response before placement publication.
    /// Example: `commands.move_sketch_dimension_projected(sketch, dimension, point, |reply| encode(reply?))?;`.
    pub fn move_sketch_dimension_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        position: [f64; 2],
        project: impl FnOnce(Result<(), SketchError>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .move_sketch_dimension_projected(sketch, dimension, position, project)
    }
}
