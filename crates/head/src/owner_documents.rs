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
