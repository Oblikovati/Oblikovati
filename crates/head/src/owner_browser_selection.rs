// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_session::selection::SelectionClickMode;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn select_browser_sketch(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        toggle: bool,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let mode = SelectionClickMode::from_modifiers(toggle, false);
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_sketch_selection(Some(sketch), mode, self.placement.is_some())
            .map(|count| count.is_some())
            .map_err(|cause| cause.to_string())
    }
    fn require_browser_document(&self, document: u64) -> Result<(), String> {
        if self.files.workspace().active_document() != Some(document) {
            return Err(format!(
                "browser document {document}: expected active document {:?}",
                self.files.workspace().active_document()
            ));
        }
        Ok(())
    }
    pub(super) fn selected_browser_sketches(&self, document: u64) -> Vec<DocumentSketchId> {
        self.files
            .workspace()
            .session(document)
            .map(|session| session.selected_browser_sketches())
            .unwrap_or_default()
    }
}
