// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchGeometricCreationProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchGeometricCreationError, SketchGeometricCreationReport},
    sketch::SketchGeometricConstraintCreation,
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Author a geometric relation on the owned document without changing workspace focus.
    /// Example: `commands.create_sketch_geometric_constraint(sketch, &input)?;`.
    pub fn create_sketch_geometric_constraint(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchGeometricConstraintCreation,
    ) -> Result<SketchGeometricCreationReport, CommandError<SketchGeometricCreationError>> {
        self.session
            .create_sketch_geometric_constraint(sketch, input)
    }
    /// Prepare the reply before the shared command adopts the relation and current solve state.
    /// Example: `commands.create_sketch_geometric_constraint_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_sketch_geometric_constraint_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchGeometricConstraintCreation,
        project: impl FnOnce(SketchGeometricCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .create_sketch_geometric_constraint_projected(sketch, input, project)
    }
}
