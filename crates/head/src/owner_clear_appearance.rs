// SPDX-License-Identifier: GPL-2.0-only
//! Remove a selected body's own appearance without changing its selection or material.
use super::*;
use oblikovati_model::assets::AssignmentReference;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn clear_selected_body_appearance(
        &mut self,
        document: u64,
        key: &str,
    ) -> Result<bool, String> {
        if self.files.workspace().active_document() != Some(document) {
            return Err(format!(
                "document {document}: expected the active document for Clear"
            ));
        }
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if !selected_has_override(session, key)? {
            return Ok(false);
        }
        self.assign_body_appearance(document, key, "")
    }
}

fn selected_has_override<O: TransactionObserver>(
    session: &oblikovati_session::DocumentSession<O>,
    key: &str,
) -> Result<bool, String> {
    let selected = session.selected_browser_body().map(|(key, _)| key);
    if selected != Some(key) {
        return Err(format!(
            "body {key:?}: expected the current selected body {selected:?}"
        ));
    }
    let reference = AssignmentReference::new(key, vec![])?;
    Ok(session
        .snapshot()
        .assets
        .assignments
        .body_appearance_id(&reference)
        .is_some())
}
