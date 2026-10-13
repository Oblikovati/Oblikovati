// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchPolylineProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchPolylineCreation, SketchPolylineCreationReport, SketchPrimitiveCreationError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Create on this document without changing workspace focus.
    /// Example: `commands.create_sketch_polyline(sketch, &input)?;`.
    pub fn create_sketch_polyline(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPolylineCreation,
    ) -> Result<SketchPolylineCreationReport, CommandError<SketchPrimitiveCreationError>> {
        self.session.create_sketch_polyline(sketch, input)
    }
    /// Prepare a typed reply before the shared document command adopts geometry.
    /// Example: `commands.create_sketch_polyline_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_sketch_polyline_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPolylineCreation,
        project: impl FnOnce(SketchPolylineProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .create_sketch_polyline_projected(sketch, input, project)
    }
}
