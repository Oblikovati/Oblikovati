// SPDX-License-Identifier: GPL-2.0-only
//! Clear current exact body overrides in one reversible edit.
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
    pub(super) fn clear_selected_body_appearances(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<bool, String> {
        let references = self.validate_clear_body_selection(document, keys)?;
        if !self.selection_has_body_override(document, &references)? {
            return Ok(false);
        }
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .run("Clear appearance", |snapshot, _| {
                clear_body_overrides(snapshot, &references)
            })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn selection_has_body_override(
        &self,
        document: u64,
        refs: &[AssignmentReference],
    ) -> Result<bool, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let assignments = &session.snapshot().assets.assignments;
        Ok(refs
            .iter()
            .any(|reference| assignments.body_appearance_id(reference).is_some()))
    }
    fn validate_clear_body_selection(
        &self,
        document: u64,
        keys: &[String],
    ) -> Result<Vec<AssignmentReference>, String> {
        self.require_current_body_selection(document, keys)?;
        let bodies = self.part_scenes.body_rows(&self.files, document)?;
        if keys
            .iter()
            .any(|key| !bodies.iter().any(|body| body.key == *key))
        {
            return Err(format!(
                "bodies {keys:?}: expected current exact body references"
            ));
        }
        keys.iter()
            .map(|key| AssignmentReference::new(key, vec![]))
            .collect()
    }
    fn require_current_body_selection(&self, document: u64, keys: &[String]) -> Result<(), String> {
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
        require_selected_keys(session, keys)
    }
}
fn clear_body_overrides(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    refs: &[AssignmentReference],
) -> Result<(), String> {
    for reference in refs {
        snapshot
            .assets
            .assignments
            .set_body_appearance(reference, "")?;
    }
    Ok(())
}

fn require_selected_keys<O: TransactionObserver>(
    session: &oblikovati_session::DocumentSession<O>,
    keys: &[String],
) -> Result<(), String> {
    let selected: Vec<_> = session
        .selected_browser_bodies()
        .iter()
        .map(|body| body.key.as_str())
        .collect();
    if keys.is_empty() || selected != keys.iter().map(String::as_str).collect::<Vec<_>>() {
        return Err(format!(
            "bodies {keys:?}: expected current nonempty selection {selected:?}"
        ));
    }
    Ok(())
}
