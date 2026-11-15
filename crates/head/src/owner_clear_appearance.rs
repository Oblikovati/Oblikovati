// SPDX-License-Identifier: GPL-2.0-only
//! Clear current exact body and face overrides in reversible edits.
use super::*;
use oblikovati_model::assets::AssignmentReference;

#[derive(Clone, Debug)]
pub(super) struct SelectedFaceAppearance {
    pub face: browser_selection::ChosenModelFace,
    pub key: String,
    pub generation: u64,
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn capture_face_appearance_pick(
        &mut self,
        document: u64,
        face: Option<browser_selection::ChosenModelFace>,
    ) -> Option<SelectedFaceAppearance> {
        let face = face?;
        let key = self.pick_face_key(document, face)?;
        let generation = self.files.workspace().session(document).ok()?.generation();
        Some(SelectedFaceAppearance {
            face,
            key,
            generation,
        })
    }
    fn pick_face_key(
        &mut self,
        document: u64,
        face: browser_selection::ChosenModelFace,
    ) -> Option<String> {
        match self.part_scenes.selected_face_reference(
            &mut self.files,
            document,
            face.feature,
            face.role,
        ) {
            Ok(key) => Some(key),
            Err(cause) => {
                self.command_notice = Some(crate::DesktopCommandNotice::SelectionFailed { cause });
                None
            }
        }
    }
    pub(super) fn current_face_appearance_pick(
        &self,
        document: u64,
    ) -> Option<&SelectedFaceAppearance> {
        let pick = self.face_appearance_pick.as_ref()?;
        let session = self.files.workspace().session(document).ok()?;
        (self.chosen_model_face(document) == Some(pick.face)
            && session.generation() == pick.generation)
            .then_some(pick)
    }
    pub(super) fn clear_selected_face_appearance(
        &mut self,
        document: u64,
        key: &str,
    ) -> Result<bool, String> {
        let pick = self.validate_face_clear_selection(document, key)?;
        self.verify_current_face_key(document, key, &pick)?;
        self.clear_face_override(document, &AssignmentReference::new(key, vec![])?)
    }
    fn validate_face_clear_selection(
        &self,
        document: u64,
        key: &str,
    ) -> Result<SelectedFaceAppearance, String> {
        self.require_browser_document(document)?;
        let pick = self
            .current_face_appearance_pick(document)
            .ok_or_else(|| format!("face {key:?}: expected the current named face selection"))?;
        if pick.key != key {
            return Err(format!(
                "face {key:?}: expected the selected reference {:?}",
                pick.key
            ));
        }
        Ok(pick.clone())
    }
    fn verify_current_face_key(
        &mut self,
        document: u64,
        key: &str,
        pick: &SelectedFaceAppearance,
    ) -> Result<(), String> {
        let current = self.part_scenes.selected_face_reference(
            &mut self.files,
            document,
            pick.face.feature,
            pick.face.role,
        )?;
        if current != key || self.current_face_appearance_pick(document).is_none() {
            return Err(format!(
                "face {key:?}: expected the unchanged unique reference {current:?}"
            ));
        }
        Ok(())
    }
    fn clear_face_override(
        &mut self,
        document: u64,
        reference: &AssignmentReference,
    ) -> Result<bool, String> {
        if !self.face_has_override(document, reference)? {
            return Ok(false);
        }
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .run("Clear appearance", |snapshot, _| {
                snapshot
                    .assets
                    .assignments
                    .set_face_appearance(reference, "")
            })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn face_has_override(
        &self,
        document: u64,
        reference: &AssignmentReference,
    ) -> Result<bool, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        Ok(session
            .snapshot()
            .assets
            .assignments
            .face_appearances()
            .any(|(key, _)| key == reference))
    }
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
