// SPDX-License-Identifier: GPL-2.0-only
//! Stage the current body appearance clear until the user accepts or cancels it.
use crate::{GuiCommandState, GuiShellAction, dockable_window::DockableWindow};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct ClearAppearance {
    document: Option<u64>,
}
impl DockableWindow for ClearAppearance {}
impl ClearAppearance {
    pub(crate) fn open(&mut self, document: u64) {
        self.document = Some(document);
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = self.document else {
            return;
        };
        if state.document != Some(document) {
            self.document = None;
            return;
        }
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
            self.document = None;
        }
    }
    fn action_for(state: &GuiCommandState, document: u64) -> Option<GuiShellAction> {
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
fn confirm_clear(
    ui: &Ui,
    state: &GuiCommandState,
    action: Option<GuiShellAction>,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    ui.text("Clear appearance overrides for the selected bodies or faces?");
    if state.selected_face_appearances.len() > 1 {
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
