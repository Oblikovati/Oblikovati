// SPDX-License-Identifier: GPL-2.0-only
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentSketchCreation, DocumentSketchCreationError},
};
/// Creation success or a typed decline before adoption; example: `let reply: SketchCreationProjectionInput = Ok(sketch);`.
pub type SketchCreationProjectionInput = Result<DocumentSketchId, DocumentSketchCreationError>;
impl<O: TransactionObserver> DocumentSession<O> {
    /// Create through the normal history and automatic solve boundary.
    /// Example: `let sketch = session.create_sketch(&request)?;`.
    pub fn create_sketch(
        &mut self,
        request: &DocumentSketchCreation,
    ) -> Result<DocumentSketchId, CommandError<DocumentSketchCreationError>> {
        self.create_sketch_projected(request, |result| result)
    }
    /// Prepare the creation reply before adoption; failed serialization discards the candidate.
    /// Example: `session.create_sketch_projected(&request, |result| encode(result?))?`.
    pub fn create_sketch_projected<R, E>(
        &mut self,
        request: &DocumentSketchCreation,
        project: impl FnOnce(SketchCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.run("Create sketch", |snapshot, reservations| {
            project(snapshot.create_document_sketch(request, reservations))
        })
    }
}
