// SPDX-License-Identifier: GPL-2.0-only
//! Track the background model owners of opened drawing documents.
use super::*;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Remember a resolved drawing model; example: `files.attach_drawing_reference(drawing, name, model, true)`.
    pub fn attach_drawing_reference(
        &mut self,
        drawing: u64,
        name: String,
        model: u64,
        background: bool,
    ) {
        self.drawing_references.insert(drawing, (name, model));
        self.drawing_reference_errors.remove(&drawing);
        if background {
            self.drawing_background_models.insert(model);
        }
    }
    /// Find a drawing's resolved model only while its authored reference still matches.
    /// Example: `files.drawing_reference(drawing, name)` before projection.
    pub fn drawing_reference(&self, drawing: u64, name: &str) -> Option<u64> {
        self.drawing_references
            .get(&drawing)
            .filter(|(reference, model)| reference == name && self.retained.contains_key(model))
            .map(|(_, model)| *model)
    }
    /// Check whether a model is retained by a drawing; example: `files.has_drawing_dependents(model)`.
    pub fn has_drawing_dependents(&self, model: u64) -> bool {
        self.drawing_references
            .iter()
            .any(|(drawing, (reference, source))| {
                *source == model
                    && self
                        .workspace
                        .session(*drawing)
                        .ok()
                        .and_then(|session| session.snapshot().drawing.as_ref())
                        .is_some_and(|drawing| drawing.model_reference() == reference)
            })
    }
    /// Record a failed model load; example: `files.reject_drawing_reference(drawing, name, cause)`.
    pub fn reject_drawing_reference(&mut self, drawing: u64, name: String, cause: String) {
        self.drawing_reference_errors.insert(drawing, (name, cause));
    }
    /// Read the last failure for the same reference; example: `files.drawing_reference_error(id, name)`.
    pub fn drawing_reference_error(&self, drawing: u64, name: &str) -> Option<&str> {
        self.drawing_reference_errors
            .get(&drawing)
            .filter(|(reference, _)| reference == name)
            .map(|(_, cause)| cause.as_str())
    }
    pub(super) fn release_drawing_reference(&mut self, drawing: u64) {
        self.drawing_reference_errors.remove(&drawing);
        let Some((_, model)) = self.drawing_references.remove(&drawing) else {
            return;
        };
        let shared = self.has_drawing_dependents(model);
        if shared || !self.drawing_background_models.contains(&model) {
            return;
        }
        let can_release = self
            .document_info(model)
            .is_ok_and(|info| !info.visible && !info.dirty);
        if can_release && self.workspace.close_discarding_changes(model).is_ok() {
            self.drawing_background_models.remove(&model);
            self.release_file(model);
        }
    }
}
