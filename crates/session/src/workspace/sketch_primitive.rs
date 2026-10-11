// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchPrimitiveProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchPrimitiveCreationError, SketchPrimitiveCreationReport},
    sketch::SketchPrimitiveCreation,
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Create on this document without changing workspace focus.
    /// Example: `commands.create_sketch_primitive(sketch, &input)?;`.
    pub fn create_sketch_primitive(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPrimitiveCreation,
    ) -> Result<SketchPrimitiveCreationReport, CommandError<SketchPrimitiveCreationError>> {
        self.session.create_sketch_primitive(sketch, input)
    }
    /// Prepare a typed reply before the shared document command adopts geometry.
    /// Example: `commands.create_sketch_primitive_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_sketch_primitive_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPrimitiveCreation,
        project: impl FnOnce(SketchPrimitiveProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .create_sketch_primitive_projected(sketch, input, project)
    }
}
