// SPDX-License-Identifier: GPL-2.0-only
//! Clear current exact body and face overrides in reversible edits.
use super::*;
use oblikovati_model::assets::AssignmentReference;

#[derive(Clone, Debug)]
pub(super) struct SelectedFaceAppearance {
    pub face: browser_selection::ChosenModelFace,
    pub document: u64,
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
            document,
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
    pub(super) fn current_face_appearance_picks(&self, document: u64) -> &[SelectedFaceAppearance] {
        let Some(primary) = self.chosen_model_face(document) else {
            return &[];
        };
        let Ok(session) = self.files.workspace().session(document) else {
            return &[];
        };
        if self
            .face_appearance_picks
            .last()
            .is_none_or(|pick| pick.face != primary)
            || self
                .face_appearance_picks
                .iter()
                .any(|pick| pick.document != document || pick.generation != session.generation())
        {
            return &[];
        }
        &self.face_appearance_picks
    }
    pub(super) fn clear_selected_face_appearances(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<bool, String> {
        self.clear_selected_appearances(document, &[], keys)
    }
    pub(super) fn clear_selected_appearances(
        &mut self,
        document: u64,
        body_keys: &[String],
        face_keys: &[String],
    ) -> Result<bool, String> {
        if body_keys.is_empty() && face_keys.is_empty() {
            return Err("clear selection []: expected at least one body or face".into());
        }
        let bodies = self.validate_optional_body_selection(document, body_keys)?;
        let faces = self.validate_optional_face_selection(document, face_keys)?;
        self.validate_optional_body_selection(document, body_keys)?;
        if !self.selection_has_body_override(document, &bodies)?
            && !self.selection_has_face_override(document, &faces)?
        {
            return Ok(false);
        }
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .run("Clear appearance", |snapshot, _| {
                clear_body_overrides(snapshot, &bodies)?;
                for reference in &faces {
                    snapshot
                        .assets
                        .assignments
                        .set_face_appearance(reference, "")?;
                }
                Ok::<(), String>(())
            })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    fn validate_optional_face_selection(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<Vec<AssignmentReference>, String> {
        if keys.is_empty() && self.current_face_appearance_picks(document).is_empty() {
            return Ok(Vec::new());
        }
        self.validate_face_clear_selection(document, keys)
    }
    fn validate_optional_body_selection(
        &self,
        document: u64,
        keys: &[String],
    ) -> Result<Vec<AssignmentReference>, String> {
        if keys.is_empty() {
            self.require_browser_document(document)?;
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            if session.selected_browser_bodies().is_empty() {
                return Ok(Vec::new());
            }
        }
        self.validate_clear_body_selection(document, keys)
    }
    fn validate_face_clear_selection(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<Vec<AssignmentReference>, String> {
        self.require_browser_document(document)?;
        let picks = self.current_face_appearance_picks(document).to_vec();
        if keys.is_empty() || picks.iter().map(|pick| &pick.key).ne(keys.iter()) {
            return Err(format!(
                "faces {keys:?}: expected the current nonempty named face selection"
            ));
        }
        for pick in &picks {
            let current = self.part_scenes.selected_face_reference(
                &mut self.files,
                document,
                pick.face.feature,
                pick.face.role,
            )?;
            if current != pick.key {
                return Err(format!(
                    "face {:?}: expected unchanged unique reference {current:?}",
                    pick.key
                ));
            }
        }
        if self
            .current_face_appearance_picks(document)
            .iter()
            .map(|pick| &pick.key)
            .ne(keys.iter())
        {
            return Err(format!(
                "faces {keys:?}: expected an unchanged current selection"
            ));
        }
        keys.iter()
            .map(|key| AssignmentReference::new(key, vec![]))
            .collect()
    }
    fn selection_has_face_override(
        &self,
        document: u64,
        references: &[AssignmentReference],
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
            .any(|(key, _)| references.contains(key)))
    }
    pub(super) fn clear_selected_body_appearances(
        &mut self,
        document: u64,
        keys: &[String],
    ) -> Result<bool, String> {
        self.clear_selected_appearances(document, keys, &[])
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
