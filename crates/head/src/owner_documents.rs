// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Client,
    wire::{ActivateDocumentArgs, DocumentInfo},
};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Copy canonical file metadata; example: `owner.gui_document_rows()?` for one GUI snapshot.
    pub(super) fn gui_document_rows(&self) -> Result<Vec<DocumentInfo>, String> {
        self.files
            .workspace()
            .documents()
            .iter()
            .copied()
            .map(|id| {
                self.files
                    .document_info(id)
                    .map_err(|cause| cause.to_string())
            })
            .collect()
    }
    /// Name a new document so two open tabs never read the same; example: `Part1`, then `Part2`.
    /// Each count only grows in a session: closing `Part2` does not reuse its name.
    pub(super) fn next_new_document_name(&mut self, prefix: &'static str) -> String {
        let open = self.gui_document_rows().unwrap_or_default();
        let count = self.new_document_counts.entry(prefix).or_default();
        loop {
            *count += 1;
            let name = format!("{prefix}{count}");
            if open.iter().all(|row| row.name != name) {
                return name;
            }
        }
    }
    /// Report each document's file and whether closing it would lose work.
    /// Example: `owner.gui_document_storage()?` for one GUI snapshot.
    pub(super) fn gui_document_storage(
        &self,
    ) -> Result<std::collections::BTreeMap<u64, oblikovati_gui::GuiDocumentStorage>, String> {
        let workspace = self.files.workspace();
        workspace
            .documents()
            .iter()
            .copied()
            .map(|id| {
                let path = self
                    .files
                    .path(id)
                    .ok()
                    .map(|path| path.display().to_string());
                // A bound document's identity counts; a new part's unused identity does not.
                let lost = if path.is_some() {
                    workspace.needs_save(id)
                } else {
                    workspace.recipe_changed(id)
                };
                let unsaved_edits = lost.map_err(|cause| cause.to_string())?;
                Ok((
                    id,
                    oblikovati_gui::GuiDocumentStorage {
                        path,
                        unsaved_edits,
                    },
                ))
            })
            .collect()
    }
    /// Guard and apply the existing document activation; example: `owner.activate_gui_document(&request)?`.
    pub(super) fn activate_gui_document(
        &mut self,
        request: &ActivateDocumentArgs,
    ) -> Result<bool, String> {
        self.require_gui_document_switch(request.id)?;
        if self.files.workspace().active_document() == Some(request.id) {
            return Ok(false);
        }
        Client::new(self.document_host())
            .documents()
            .activate(request)
            .map_err(|cause| format!("Activate document {}: {cause}", request.id))?;
        Ok(true)
    }
    fn require_gui_document_switch(&self, target: u64) -> Result<(), String> {
        let row = self
            .files
            .document_info(target)
            .map_err(|cause| cause.to_string())?;
        if !row.visible || self.active_sketch_edit().is_some() {
            return Err(format!(
                "Activate document {target}, visible={}: expected a visible document with no active sketch edit",
                row.visible
            ));
        }
        for document in self
            .files
            .workspace()
            .active_document()
            .into_iter()
            .chain([target])
        {
            self.require_switch_outside_transaction(target, document)?;
        }
        Ok(())
    }
    fn require_switch_outside_transaction(&self, target: u64, document: u64) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "Activate document {target}, open transaction in {document}: expected both documents outside transactions"
            ));
        }
        Ok(())
    }
}
