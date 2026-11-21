// SPDX-License-Identifier: GPL-2.0-only
//! New parts create their startup sketch through the normal session command.
use super::*;
use oblikovati_api::types::{DOCUMENT_PART, DocumentType};
use oblikovati_model::{
    application_options::PartPreferenceAccess, document_sketch::SketchPlacement,
    history::DocumentSketchCreation,
};

impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Share accepted Part settings; example: `host.with_part_preferences(preferences)`.
    pub fn with_part_preferences(mut self, preferences: &'a mut dyn PartPreferenceAccess) -> Self {
        self.context.get_mut().part_preferences = Some(preferences);
        self
    }
    /// Supply the current origin projection choice; example: `host.with_startup_projection(true)`.
    pub fn with_startup_projection(mut self, project: bool) -> Self {
        self.context.get_mut().startup_project_origin = project;
        self
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn startup_sketch_request(
        &self,
        kind: DocumentType,
    ) -> Option<DocumentSketchCreation> {
        if kind != DOCUMENT_PART {
            return None;
        }
        let plane = self
            .part_preferences
            .as_deref()?
            .part_options()
            .startup_sketch_plane?;
        let project = self
            .sketch_preferences
            .as_deref()
            .map_or(self.startup_project_origin, |store| {
                store.sketch_options().auto_project_origin
            });
        Some(DocumentSketchCreation {
            name: String::new(),
            placement: SketchPlacement::OriginPlane(plane),
            orientation: None,
            project_origin: project,
        })
    }
    pub(super) fn create_startup_sketch(
        &mut self,
        document: u64,
        previous: Option<u64>,
        request: &DocumentSketchCreation,
    ) -> Result<(), Error> {
        if let Err(cause) = self.startup_sketch_command(document, request) {
            let recovery = self.rollback_startup_document(document, previous);
            return Err(Error(format!(
                "new part {document} startup sketch: expected successful creation: {cause}; {recovery}"
            )));
        }
        Ok(())
    }
    fn startup_sketch_command(
        &mut self,
        document: u64,
        request: &DocumentSketchCreation,
    ) -> Result<(), String> {
        let mut commands = self
            .files
            .commands(document)
            .map_err(|error| error.to_string())?;
        let sketch = commands
            .create_automatic_sketch(request)
            .map_err(|error| error.to_string())?;
        commands
            .tag_sketch_creation(sketch)
            .map_err(|error| error.to_string())
    }
    fn rollback_startup_document(&mut self, document: u64, previous: Option<u64>) -> String {
        if let Err(error) = self.files.close_discarding_changes(document) {
            return format!("rollback of document {document} failed: {error}");
        }
        if let Some(previous) = previous
            && let Err(error) = self.files.activate(previous)
        {
            return format!("rollback activation of document {previous} failed: {error}");
        }
        "new document discarded and prior focus restored".into()
    }
}
