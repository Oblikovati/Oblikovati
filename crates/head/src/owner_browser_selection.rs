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
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let picks = self.browser_selection.entry(document).or_default();
        let mode = SelectionClickMode::from_modifiers(toggle, false);
        apply_checked_browser_sketch(session, sketch, mode, self.placement.is_some(), picks)
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
    pub(super) fn prune_browser_selection(&mut self) {
        let documents = self.files.workspace().documents();
        self.browser_selection
            .retain(|document, _| documents.contains(document));
    }
    pub(super) fn selected_browser_sketches(&self, document: u64) -> Vec<DocumentSketchId> {
        let Some(picks) = self.browser_selection.get(&document) else {
            return Vec::new();
        };
        let Ok(session) = self.files.workspace().session(document) else {
            return Vec::new();
        };
        picks
            .items()
            .iter()
            .filter(|pick| session.browser_sketch_pick(pick.sketch()).as_ref() == Ok(*pick))
            .map(|pick| pick.sketch())
            .collect()
    }
}

fn apply_checked_browser_sketch<O: TransactionObserver>(
    session: &oblikovati_session::DocumentSession<O>,
    sketch: DocumentSketchId,
    mode: SelectionClickMode,
    tool_active: bool,
    picks: &mut oblikovati_session::selection::SelectionSet<
        oblikovati_session::selection::BrowserSketchPick,
    >,
) -> Result<bool, String> {
    let pick = session
        .browser_sketch_pick(sketch)
        .map_err(|cause| cause.to_string())?;
    session
        .apply_browser_sketch_click(Some(pick), mode, tool_active, picks)
        .map(|count| count.is_some())
        .map_err(|cause| cause.to_string())
}
