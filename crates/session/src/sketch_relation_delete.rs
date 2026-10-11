// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraint, SketchConstraintId, SketchError},
};

/// Deleted relation or a typed decline before the transaction adopts the candidate.
/// Example: `let reply: SketchRelationDeleteProjectionInput = Ok(removed);`.
pub type SketchRelationDeleteProjectionInput = Result<SketchConstraint, SketchError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Delete an authored geometric relation and recompute with current sources.
    /// Example: `session.delete_sketch_geometric_constraint(sketch, constraint)?;`.
    pub fn delete_sketch_geometric_constraint(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
    ) -> Result<SketchConstraint, CommandError<SketchError>> {
        self.delete_sketch_geometric_constraint_projected(sketch, constraint, |result| result)
    }
    /// Prepare a response before adoption; rejected responses preserve the relation and its ID.
    /// Example: `session.delete_sketch_geometric_constraint_projected(sketch, constraint, |result| encode(result?))?;`.
    pub fn delete_sketch_geometric_constraint_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        constraint: SketchConstraintId,
        project: impl FnOnce(SketchRelationDeleteProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Delete sketch constraint", |snapshot, _| {
            project(snapshot.delete_sketch_geometric_constraint(sketch, constraint))
        })
    }
}
