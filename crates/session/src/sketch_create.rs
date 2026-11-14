// SPDX-License-Identifier: GPL-2.0-only
mod part;
use crate::{CommandError, DocumentSession, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentSketchCreation, DocumentSketchCreationError},
};
pub use part::PartSketchCreateError;
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
    /// Create with the next unused default name shared by desktop and automation.
    /// Example: `session.create_automatic_sketch(&request)?` for a new sketch.
    pub fn create_automatic_sketch(
        &mut self,
        request: &DocumentSketchCreation,
    ) -> Result<DocumentSketchId, CommandError<DocumentSketchCreationError>> {
        self.create_automatic_sketch_projected(request, |result| result)
    }
    /// Project the reply before adopting an automatically named sketch.
    /// Example: `session.create_automatic_sketch_projected(&request, |result| encode(result?))?`.
    pub fn create_automatic_sketch_projected<R, E>(
        &mut self,
        request: &DocumentSketchCreation,
        project: impl FnOnce(SketchCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        let candidate =
            crate::sketch_auto_names::next_name(self.next_sketch_name, &self.snapshot().sketches);
        let (name, next) = match candidate {
            Ok(candidate) => candidate,
            Err(error) => {
                return self.run("Create sketch", |_, _| {
                    project(Err(DocumentSketchCreationError::Identity(error)))
                });
            }
        };
        let mut named = request.clone();
        named.name = name;
        let mut created = false;
        let result = self.create_sketch_projected(&named, |input| {
            created = input.is_ok();
            project(input)
        });
        if created && result.is_ok() {
            self.next_sketch_name = next;
        }
        result
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
