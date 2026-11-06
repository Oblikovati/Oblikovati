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
