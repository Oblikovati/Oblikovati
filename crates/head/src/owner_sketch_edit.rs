// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadSketchEdit;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART, DocumentType};
use oblikovati_model::{document_sketch::DocumentSketchId, history::DocumentSketchCreation};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Enter an explicitly selected sketch without editing its recipe; example: `owner.begin_sketch_edit(document, sketch)?`.
    pub fn begin_sketch_edit(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        self.require_sketch_edit_entry(document)?;
        self.watch_sketch_edit(document, sketch)?;
        Ok(())
    }
    /// Create one history step and select the new sketch; example: `owner.create_sketch_for_edit(document, &request)?`.
    pub fn create_sketch_for_edit(
        &mut self,
        document: u64,
        request: &DocumentSketchCreation,
    ) -> Result<DocumentSketchId, String> {
        self.require_sketch_edit_entry(document)?;
        let mut request = request.clone();
        request.project_origin &= self.sketch_document_kind(document)? == DOCUMENT_PART;
        let sketch = self
            .files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .create_sketch(&request)
            .map_err(|cause| cause.to_string())?;
        self.watch_sketch_edit(document, sketch)?;
        Ok(sketch)
    }
    /// Leave the edit and discard unfinished placement; committed commands keep their own undo entries.
    /// Example: `owner.finish_sketch_edit()?` before selecting another sketch.
    pub fn finish_sketch_edit(&mut self) -> Result<(), String> {
        let edit = self.active_sketch_edit().ok_or_else(|| {
            format!(
                "sketch edit {:?}: expected a live active sketch",
                self.sketch_edit
            )
        })?;
        self.require_edit_document(edit.document)?;
        self.placement = None;
        self.sketch_edit = None;
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: "Sketch finished.".into(),
        });
        Ok(())
    }
    /// Re-resolve identity after edits and history restore; example: `owner.active_sketch_edit()` for command enablement.
    pub fn active_sketch_edit(&self) -> Option<HeadSketchEdit> {
        let (document, watch, focus) = self.sketch_edit.as_ref()?;
        if !watch.is_live() || !self.files.workspace().retains_focus(focus) {
            return None;
        }
        let document = *document;
        let sketch = watch.sketch();
        if self.files.workspace().active_document() != Some(document) {
            return None;
        }
        self.resolve_edit(document, sketch).ok()
    }
    fn watch_sketch_edit(&mut self, document: u64, sketch: DocumentSketchId) -> Result<(), String> {
        let watch = self
            .files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .watch_sketch_continuity(sketch)
            .map_err(|cause| cause.to_string())?;
        self.sketch_edit = Some((document, watch, self.files.workspace().focus_identity()));
        Ok(())
    }
    pub(super) fn resolve_edit(
        &self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<HeadSketchEdit, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let plane = session
            .snapshot()
            .resolve_sketch_plane(sketch)
            .map_err(|cause| cause.to_string())?;
        Ok(HeadSketchEdit {
            document,
            sketch,
            plane,
        })
    }
    pub(super) fn require_edit_document(&self, document: u64) -> Result<(), String> {
        if self.files.workspace().active_document() != Some(document) {
            return Err(format!(
                "sketch edit document {document}: expected active document {:?}",
                self.files.workspace().active_document()
            ));
        }
        self.sketch_document_kind(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "sketch edit document {document}: expected no open transaction"
            ));
        }
        Ok(())
    }
    fn sketch_document_kind(&self, document: u64) -> Result<DocumentType, String> {
        let kind = self
            .files
            .document_type(document)
            .map_err(|cause| cause.to_string())?;
        if !matches!(kind, DOCUMENT_PART | DOCUMENT_ASSEMBLY) {
            return Err(format!(
                "sketch edit document {document}, kind {kind:?}: expected a part or assembly"
            ));
        }
        Ok(kind)
    }
    fn require_sketch_edit_entry(&mut self, document: u64) -> Result<(), String> {
        self.synchronize_sketch_edit();
        self.require_edit_document(document)?;
        if let Some(edit) = &self.sketch_edit {
            return Err(format!(
                "sketch edit {edit:?}: expected the current edit to be finished first"
            ));
        }
        Ok(())
    }
    pub(super) fn synchronize_sketch_edit(&mut self) -> bool {
        if self.sketch_edit.is_none() || self.active_sketch_edit().is_some() {
            return false;
        }
        self.sketch_edit = None;
        self.placement = None;
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: "Sketch edit ended because its document or plane is unavailable.".into(),
        });
        true
    }
    pub(super) fn select_placement_sketch(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        self.synchronize_sketch_edit();
        if self
            .active_sketch_edit()
            .is_some_and(|edit| (edit.document, edit.sketch) == (document, sketch))
        {
            return self.require_edit_document(document);
        }
        self.begin_sketch_edit(document, sketch)
    }
}
