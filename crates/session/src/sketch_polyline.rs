// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchPolylineCreation, SketchPolylineCreationReport, SketchPrimitiveCreationError},
};
/// Creation evidence or a typed decline before owner adoption.
/// Example: `let reply: SketchPolylineProjectionInput = Ok(report);`.
pub type SketchPolylineProjectionInput =
    Result<SketchPolylineCreationReport, SketchPrimitiveCreationError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Author geometry through normal history and automatic current-source recompute.
    /// Example: `session.create_sketch_polyline(sketch, &input)?;`.
    pub fn create_sketch_polyline(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPolylineCreation,
    ) -> Result<SketchPolylineCreationReport, CommandError<SketchPrimitiveCreationError>> {
        self.create_sketch_polyline_projected(sketch, input, |result| result)
    }
    /// Prepare the reply before adoption; response failures discard geometry but retain reservations.
    /// Example: `session.create_sketch_polyline_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_sketch_polyline_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPolylineCreation,
        project: impl FnOnce(SketchPolylineProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Create sketch polyline", |snapshot, reservations| {
            project(snapshot.create_sketch_polyline(sketch, input, reservations))
        })
    }
}
