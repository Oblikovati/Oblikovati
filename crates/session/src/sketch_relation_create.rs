// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchGeometricCreationError, SketchGeometricCreationReport},
    sketch::SketchGeometricConstraintCreation,
};
/// Creation evidence or a typed decline before the transaction adopts the candidate.
/// Example: `let reply: SketchGeometricCreationProjectionInput = Ok(report);`.
pub type SketchGeometricCreationProjectionInput =
    Result<SketchGeometricCreationReport, SketchGeometricCreationError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Author and evaluate a geometric relation through the normal history boundary.
    /// Example: `session.create_sketch_geometric_constraint(sketch, &input)?;`.
    pub fn create_sketch_geometric_constraint(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchGeometricConstraintCreation,
    ) -> Result<SketchGeometricCreationReport, CommandError<SketchGeometricCreationError>> {
        self.create_sketch_geometric_constraint_projected(sketch, input, |result| result)
    }
    /// Prepare the response before publication; rejected responses retain attempted IDs only.
    /// Example: `session.create_sketch_geometric_constraint_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_sketch_geometric_constraint_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchGeometricConstraintCreation,
        project: impl FnOnce(SketchGeometricCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        let options = self.sketch_options;
        self.run("Create sketch constraint", |snapshot, ids| {
            project(snapshot.create_sketch_geometric_constraint(sketch, input, options, ids))
        })
    }
}
