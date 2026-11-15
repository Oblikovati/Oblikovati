// SPDX-License-Identifier: GPL-2.0-only
//! More than one window on a document: each has its own view and its own sketch edit.
use super::*;
use oblikovati_gui::GuiDocumentWindow;

use crate::FIRST_EXTRA_WINDOW;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// The window in front; a document with one window is its own window.
    pub(super) fn active_view_slot(&self) -> u64 {
        let Some(document) = self.files.workspace().active_document() else {
            return 0;
        };
        self.active_window
            .filter(|id| {
                self.window_of(*id)
                    .is_some_and(|window| window.document == document)
            })
            .unwrap_or_else(|| self.first_window_of(document))
    }
    fn window_of(&self, id: u64) -> Option<GuiDocumentWindow> {
        self.document_windows
            .iter()
            .copied()
            .find(|window| window.id == id)
    }
    fn first_window_of(&self, document: u64) -> u64 {
        self.document_windows
            .iter()
            .find(|window| window.document == document)
            .map_or(document, |window| window.id)
    }
    /// Open another window on the active document and bring it to the front.
    pub(super) fn new_document_window(&mut self) -> Result<bool, String> {
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or("New window: expected an active document")?;
        if !self
            .document_windows
            .iter()
            .any(|window| window.document == document)
        {
            self.document_windows.push(GuiDocumentWindow {
                id: document,
                document,
                number: 1,
            });
        }
        self.park_active_sketch_edit()?;
        let id = self.next_window_id();
        let number = self.next_window_number(document);
        self.document_windows.push(GuiDocumentWindow {
            id,
            document,
            number,
        });
        self.active_window = Some(id);
        Ok(true)
    }
    fn next_window_id(&mut self) -> u64 {
        self.window_serial += 1;
        FIRST_EXTRA_WINDOW + self.window_serial
    }
    fn next_window_number(&self, document: u64) -> u32 {
        self.document_windows
            .iter()
            .filter(|window| window.document == document)
            .map(|window| window.number)
            .max()
            .map_or(1, |highest| highest + 1)
    }
    /// Bring one window forward: the edit in the window left behind waits for its return.
    pub(super) fn activate_document_window(&mut self, id: u64) -> Result<bool, String> {
        let window = self
            .window_of(id)
            .ok_or_else(|| format!("window {id}: expected an open document window"))?;
        if self.active_view_slot() == id {
            return Ok(false);
        }
        self.require_gui_document_switch(window.document)?;
        self.park_active_sketch_edit()?;
        if self.files.workspace().active_document() != Some(window.document) {
            oblikovati_api::client::Client::new(self.document_host())
                .documents()
                .activate(&oblikovati_api::wire::ActivateDocumentArgs {
                    id: window.document,
                })
                .map_err(|cause| format!("Activate window {id}: {cause}"))?;
        }
        self.active_window = Some(id);
        self.resume_parked_sketch_edit(id, window.document)?;
        Ok(true)
    }
    /// Close one of several windows: the document and its unsaved work stay open.
    pub(super) fn close_document_window(&mut self, id: u64) -> Result<bool, String> {
        let window = self
            .window_of(id)
            .ok_or_else(|| format!("window {id}: expected an open document window"))?;
        let count = self
            .document_windows
            .iter()
            .filter(|known| known.document == window.document)
            .count();
        if count < 2 {
            return Err(format!(
                "window {id} of document {}: expected another window on the document",
                window.document
            ));
        }
        if self.active_view_slot() == id {
            // Its edit has no window left; it was a view, so nothing in the recipe changes.
            self.park_active_sketch_edit()?;
            self.parked_sketch_edits.remove(&id);
        }
        self.document_windows.retain(|known| known.id != id);
        self.settle_after_window_close(window.document)
    }
    /// The next window comes forward; a lone window leaves the numbered list again.
    fn settle_after_window_close(&mut self, document: u64) -> Result<bool, String> {
        let remaining: Vec<u64> = self
            .document_windows
            .iter()
            .filter(|known| known.document == document)
            .map(|known| known.id)
            .collect();
        let next = remaining[0];
        self.active_window = None;
        if remaining.len() == 1 {
            if next != document {
                self.window_identity_revisions
                    .insert(document, self.window_serial);
            }
            self.document_windows
                .retain(|known| known.document != document);
            if let Some(sketch) = self.parked_sketch_edits.remove(&next) {
                self.parked_sketch_edits.insert(document, sketch);
            }
            self.resume_parked_sketch_edit(document, document)?;
            return Ok(true);
        }
        self.active_window = Some(next);
        self.resume_parked_sketch_edit(next, document)?;
        Ok(true)
    }
    /// A closed document takes all its windows with it.
    pub(super) fn forget_document_windows(&mut self, document: u64) {
        self.window_identity_revisions.remove(&document);
        let ids: Vec<u64> = self
            .document_windows
            .iter()
            .filter(|window| window.document == document)
            .map(|window| window.id)
            .collect();
        for id in ids {
            self.parked_sketch_edits.remove(&id);
        }
        self.document_windows
            .retain(|window| window.document != document);
        self.parked_sketch_edits.remove(&document);
    }
}
