// SPDX-License-Identifier: GPL-2.0-only
//! The list of files this application opened or saved, for Home and the File menu.
use super::*;
use oblikovati_gui::{GuiRecentAction, GuiRecentDocument};
use oblikovati_model::application_recent::{RecentDocumentAccess, note_recent_document};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Attach the stored list; example: `owner.with_recent_documents(Box::new(file))`.
    pub fn with_recent_documents(mut self, store: Box<dyn RecentDocumentAccess>) -> Self {
        self.recent_documents = store.recent_documents();
        self.recent_store = Some(store);
        self.refresh_recent_rows();
        self
    }
    /// A failed write leaves the list in effect for this session and reports the cause.
    pub(super) fn note_recent_file(&mut self, path: &str) {
        let now = crate::clock::system_time_now()
            .duration_since(std::time::UNIX_EPOCH)
            .map_or(0, |since| since.as_secs());
        note_recent_document(&mut self.recent_documents, path, now);
        self.publish_recent_list();
    }
    fn publish_recent_list(&mut self) {
        self.refresh_recent_rows();
        if let Some(store) = self.recent_store.as_mut()
            && let Err(cause) = store.set_recent_documents(self.recent_documents.clone())
        {
            self.command_notice = Some(DesktopCommandNotice::AuthoringFailed { cause });
        }
    }
    /// Change the stored list from Home; example: remove a row, or pin one.
    pub fn recent_from_gui(&mut self, action: &GuiRecentAction) -> Result<bool, String> {
        match action {
            GuiRecentAction::Remove(path) => {
                self.recent_documents.retain(|known| known.path != *path);
            }
            GuiRecentAction::SetPinned { path, pinned } => {
                for known in &mut self.recent_documents {
                    if known.path == *path {
                        known.pinned = *pinned;
                    }
                }
            }
        }
        self.publish_recent_list();
        Ok(true)
    }
    pub(super) fn gui_recent_rows(&self) -> Vec<GuiRecentDocument> {
        self.recent_rows.clone()
    }
    fn refresh_recent_rows(&mut self) {
        let now = crate::clock::system_time_now();
        self.recent_rows = self
            .recent_documents
            .iter()
            .map(|known| {
                let mut row =
                    GuiRecentDocument::of_file(&known.path, known.last_opened, known.pinned, now);
                row.design_views = oblikovati_persistence::package::design_view_names_of_file(
                    std::path::Path::new(&known.path),
                );
                row
            })
            .collect();
    }
    /// Start a scenario from a known list; the stored file keeps other sessions' entries.
    #[cfg(all(test, target_os = "linux"))]
    pub(crate) fn forget_recent_documents(&mut self) {
        self.recent_documents.clear();
        self.recent_rows.clear();
    }
}
