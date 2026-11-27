// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchRelationDeleteProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraint, SketchConstraintId, SketchError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Delete by stable identity through the owned session without changing workspace focus.
    /// Example: `commands.delete_sketch_geometric_constraint(sketch, constraint)?;`.
    pub fn delete_sketch_geometric_constraint(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
    ) -> Result<SketchConstraint, CommandError<SketchError>> {
        self.session
            .delete_sketch_geometric_constraint(sketch, constraint)
    }
    /// Prepare the canonical reply before the shared transaction adopts deletion.
    /// Example: `commands.delete_sketch_geometric_constraint_projected(sketch, constraint, |result| encode(result?))?;`.
    pub fn delete_sketch_geometric_constraint_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
        project: impl FnOnce(SketchRelationDeleteProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .delete_sketch_geometric_constraint_projected(sketch, constraint, project)
    }
}
