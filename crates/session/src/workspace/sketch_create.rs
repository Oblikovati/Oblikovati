// SPDX-License-Identifier: GPL-2.0-only
use super::SessionCommands;
use crate::{CommandError, SketchCreationProjectionInput, TransactionObserver};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentSketchCreation, DocumentSketchCreationError},
};
impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Create on this document without changing workspace focus.
    /// Example: `commands.create_sketch(&request)?`.
    pub fn create_sketch(
        &mut self,
        request: &DocumentSketchCreation,
    ) -> Result<DocumentSketchId, CommandError<DocumentSketchCreationError>> {
        self.session.create_sketch(request)
    }
    /// Create with the next document-local default name.
    /// Example: `commands.create_automatic_sketch(&request)?` for a new sketch.
    pub fn create_automatic_sketch(
        &mut self,
        request: &DocumentSketchCreation,
    ) -> Result<DocumentSketchId, CommandError<DocumentSketchCreationError>> {
        self.session.create_automatic_sketch(request)
    }
    /// Project an automatically named sketch before adopting it.
    /// Example: `commands.create_automatic_sketch_projected(&request, |result| encode(result?))?`.
    pub fn create_automatic_sketch_projected<R, E>(
        &mut self,
        request: &DocumentSketchCreation,
        project: impl FnOnce(SketchCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session
            .create_automatic_sketch_projected(request, project)
    }
    /// Build a reply before adopting geometry, through the shared owner command.
    /// Example: `commands.create_sketch_projected(&request, |result| encode(result?))?`.
    pub fn create_sketch_projected<R, E>(
        &mut self,
        request: &DocumentSketchCreation,
        project: impl FnOnce(SketchCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session.create_sketch_projected(request, project)
    }
}
