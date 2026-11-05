// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{SketchPrimitiveCreationError, SketchPrimitiveCreationReport},
    sketch::SketchPrimitiveCreation,
};
/// Creation evidence or a typed decline before owner adoption.
/// Example: `let reply: SketchPrimitiveProjectionInput = Ok(report);`.
pub type SketchPrimitiveProjectionInput =
    Result<SketchPrimitiveCreationReport, SketchPrimitiveCreationError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Author geometry through normal history and automatic current-source recompute.
    /// Example: `session.create_sketch_primitive(sketch, &input)?;`.
    pub fn create_sketch_primitive(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPrimitiveCreation,
    ) -> Result<SketchPrimitiveCreationReport, CommandError<SketchPrimitiveCreationError>> {
        self.create_sketch_primitive_projected(sketch, input, |result| result)
    }
    /// Prepare the reply before adoption; response failures discard geometry but retain reservations.
    /// Example: `session.create_sketch_primitive_projected(sketch, &input, |result| encode(result?))?;`.
    pub fn create_sketch_primitive_projected<R, E>(
        &mut self,
        sketch: DocumentSketchId,
        input: &SketchPrimitiveCreation,
        project: impl FnOnce(SketchPrimitiveProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Create sketch primitive", |snapshot, reservations| {
            project(snapshot.create_sketch_primitive(sketch, input, reservations))
        })
    }
}
