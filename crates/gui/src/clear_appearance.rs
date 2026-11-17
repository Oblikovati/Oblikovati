// SPDX-License-Identifier: GPL-2.0-only
//! Stage the current body appearance clear until the user accepts or cancels it.
use crate::{GuiCommandState, GuiShellAction, dockable_window::DockableWindow};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct ClearAppearance {
    document: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
}
impl DockableWindow for ClearAppearance {}
impl ClearAppearance {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Clear Appearance", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.document = document;
        self.dock_state.set_open(document.is_some());
    }
    pub(crate) fn open(&mut self, document: u64) {
        self.set_document(Some(document));
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState) {
        if self.document.is_some() && self.document != state.document {
            self.set_document(None);
        }
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state);
        let Some(document) = self.document else {
            return;
        };
        let action = Self::action_for(state, document);
        let mut open = true;
        let mut done = false;
        let width =
            ui.calc_text_size("Select bodies or faces with an appearance override to continue.")[0];
        let size = [width + ui.frame_height() * 2., ui.frame_height() * 6.];
        Self::dockable_window(ui, "Clear Appearance", &mut open)
            .size(size, imgui::Condition::FirstUseEver)
            .build(|| done = confirm_clear(ui, state, action, actions));
        if !open || done {
            self.set_document(None);
        }
    }
    fn action_for(state: &GuiCommandState, document: u64) -> Option<GuiShellAction> {
        if !state.selected_face_appearances.is_empty() && !state.selected_body_keys.is_empty() {
            return Some(GuiShellAction::ClearSelectedMixedAppearance {
                document,
                body_keys: state.selected_body_keys.clone(),
                face_keys: state
                    .selected_face_appearances
                    .iter()
                    .map(|(key, _)| key.clone())
                    .collect(),
            });
        }
        if !state.selected_face_appearances.is_empty() {
            return Some(GuiShellAction::ClearSelectedFaceAppearance {
                document,
                keys: state
                    .selected_face_appearances
                    .iter()
                    .map(|(key, _)| key.clone())
                    .collect(),
            });
        }
        Some(GuiShellAction::ClearSelectedBodyAppearance {
            document,
            keys: (!state.selected_body_keys.is_empty())
                .then(|| state.selected_body_keys.clone())?,
        })
    }
}
/// Check an exact current clear selection; example: `clear_action_enabled(&action, &state)`.
pub(crate) fn clear_action_enabled(action: &GuiShellAction, state: &GuiCommandState) -> bool {
    let Some((document, bodies, faces)) = clear_action_targets(action) else {
        return false;
    };
    if state.document != Some(document)
        || state.authoring != crate::GuiAuthoringContext::Part
        || state.transaction_open
        || !clear_selection_matches(state, bodies, faces)
    {
        return false;
    }
    let Some(body_assigned) = body_assignment_status(state, bodies) else {
        return false;
    };
    body_assigned
        || (!faces.is_empty()
            && state
                .selected_face_appearances
                .iter()
                .any(|(_, assigned)| *assigned))
}

fn clear_action_targets(action: &GuiShellAction) -> Option<(u64, &[String], &[String])> {
    match action {
        GuiShellAction::ClearSelectedFaceAppearance { document, keys } if !keys.is_empty() => {
            Some((*document, &[], keys))
        }
        GuiShellAction::ClearSelectedBodyAppearance { document, keys } if !keys.is_empty() => {
            Some((*document, keys, &[]))
        }
        GuiShellAction::ClearSelectedMixedAppearance {
            document,
            body_keys,
            face_keys,
        } if !body_keys.is_empty() && !face_keys.is_empty() => {
            Some((*document, body_keys, face_keys))
        }
        _ => None,
    }
}

fn clear_selection_matches(state: &GuiCommandState, bodies: &[String], faces: &[String]) -> bool {
    state.selected_body_keys == bodies
        && state
            .selected_face_appearances
            .iter()
            .map(|(key, _)| key)
            .eq(faces.iter())
}

fn body_assignment_status(state: &GuiCommandState, keys: &[String]) -> Option<bool> {
    if keys.is_empty() {
        return Some(false);
    }
    let browser = state.browser.as_ref()?;
    if !keys
        .iter()
        .all(|key| browser.bodies.iter().any(|body| body.key == *key))
    {
        return None;
    }
    Some(keys.iter().any(|key| {
        browser
            .bodies
            .iter()
            .any(|body| body.key == *key && !body.appearance.is_empty())
    }))
}

fn confirm_clear(
    ui: &Ui,
    state: &GuiCommandState,
    action: Option<GuiShellAction>,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    ui.text("Clear appearance overrides for the selected bodies or faces?");
    if !state.selected_body_keys.is_empty() && !state.selected_face_appearances.is_empty() {
        ui.text(format!(
            "Selected bodies: {}; faces: {}",
            state.selected_body_keys.len(),
            state.selected_face_appearances.len()
        ));
    } else if state.selected_face_appearances.len() > 1 {
        ui.text(format!(
            "{} faces selected",
            state.selected_face_appearances.len()
        ));
    }
    let valid = action.as_ref().is_some_and(|value| value.enabled(state));
    if !valid {
        ui.text_wrapped("Select bodies or faces with an appearance override to continue.");
    }
    let disabled = ui.begin_disabled(!valid);
    let accepted = ui.button("OK");
    drop(disabled);
    ui.same_line();
    let cancelled = ui.button("Cancel");
    if accepted && let Some(action) = action {
        actions.push(action);
    }
    accepted || cancelled
}

#[cfg(test)]
mod tests {
    #[test]
    fn document_change_closes_before_docking() {
        let mut editor = ClearAppearance::default();
        editor.open(1);
        editor.sync_before_docking(&GuiCommandState::default());
        assert!(editor.document.is_none());
    }

    use super::*;

    #[test]
    fn clear_target_uses_current_face_and_requires_its_override() {
        let mut state = GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            selected_face_appearances: vec![("face-key".into(), true)],
            ..Default::default()
        };
        let action = ClearAppearance::action_for(&state, 7).unwrap();
        assert_eq!(
            action,
            GuiShellAction::ClearSelectedFaceAppearance {
                document: 7,
                keys: vec!["face-key".into()],
            }
        );
        assert!(action.enabled(&state));
        state.selected_face_appearances = vec![("face-key".into(), false)];
        assert!(!action.enabled(&state));
        state.selected_face_appearances = vec![("other".into(), true)];
        assert!(!action.enabled(&state));
    }

    #[test]
    fn clear_target_checks_the_complete_face_selection() {
        let mut state = GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            selected_face_appearances: vec![("first".into(), true), ("second".into(), false)],
            ..Default::default()
        };
        let action = ClearAppearance::action_for(&state, 7).unwrap();
        assert!(action.enabled(&state));
        state.selected_face_appearances.swap(0, 1);
        assert!(!action.enabled(&state));
        state.selected_face_appearances[0].1 = false;
        state.selected_face_appearances[1].1 = false;
        assert!(
            !ClearAppearance::action_for(&state, 7)
                .unwrap()
                .enabled(&state)
        );
    }

    #[test]
    fn mixed_clear_requires_current_known_body_and_face() {
        let body = crate::GuiBrowserBody {
            name: "Solid1".into(),
            feature: oblikovati_model::feature::FeatureId(1),
            key: "body".into(),
            index: 0,
            visible: true,
            faces: 6,
            appearance: String::new(),
            physical: None,
        };
        let mut state = GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            selected_body_keys: vec!["body".into()],
            selected_face_appearances: vec![("face".into(), true)],
            browser: Some(crate::GuiModelBrowser {
                bodies: vec![body],
                ..Default::default()
            }),
            ..Default::default()
        };
        let action = ClearAppearance::action_for(&state, 7).unwrap();
        assert!(matches!(
            action,
            GuiShellAction::ClearSelectedMixedAppearance { .. }
        ));
        assert!(action.enabled(&state));
        let partial = GuiShellAction::ClearSelectedFaceAppearance {
            document: 7,
            keys: vec!["face".into()],
        };
        assert!(!partial.enabled(&state));
        state.browser.as_mut().unwrap().bodies.clear();
        assert!(!action.enabled(&state));
        state.selected_face_appearances.clear();
        assert!(!action.enabled(&state));
    }

    #[test]
    fn clear_target_tracks_current_body_selection() {
        let mut state = GuiCommandState::default();
        assert!(ClearAppearance::action_for(&state, 7).is_none());
        state.selected_body_keys = vec!["first".into()];
        let first = ClearAppearance::action_for(&state, 7).unwrap();
        state.selected_body_keys = vec!["second".into()];
        let second = ClearAppearance::action_for(&state, 7).unwrap();
        assert_eq!(
            first,
            GuiShellAction::ClearSelectedBodyAppearance {
                document: 7,
                keys: vec!["first".into()],
            }
        );
        assert_eq!(
            second,
            GuiShellAction::ClearSelectedBodyAppearance {
                document: 7,
                keys: vec!["second".into()],
            }
        );
    }
}
