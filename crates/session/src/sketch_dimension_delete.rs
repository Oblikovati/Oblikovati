// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchError},
};
/// Resulting DOF or a named decline before candidate adoption.
/// Example: `let reply: SketchDimensionDeleteProjectionInput = Ok(4);`.
pub type SketchDimensionDeleteProjectionInput = Result<usize, SketchError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Delete an owned dimension and recompute through the normal history boundary.
    /// Example: `session.delete_sketch_dimension(sketch, constraint)?;`.
    pub fn delete_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
    ) -> Result<usize, CommandError<SketchError>> {
        self.delete_sketch_dimension_projected(sketch, constraint, |reply| reply)
    }
    /// Prepare the reply before adoption; a rejected reply preserves dimension and parameter.
    /// Example: `session.delete_sketch_dimension_projected(sketch, constraint, |reply| encode(reply?))?;`.
    pub fn delete_sketch_dimension_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
        project: impl FnOnce(SketchDimensionDeleteProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        let options = self.sketch_options;
        self.run("Delete sketch dimension", |snapshot, _| {
            project(snapshot.delete_sketch_dimension(sketch, constraint, options))
        })
    }
}
