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
        let width = ui.calc_text_size("Select a body with an appearance override to continue.")[0];
        let size = [width + ui.frame_height() * 2., ui.frame_height() * 6.];
        Self::dockable_window(ui, "Clear Appearance", &mut open)
            .size(size, imgui::Condition::FirstUseEver)
            .build(|| done = confirm_clear(ui, state, action, actions));
        if !open || done {
            self.document = None;
        }
    }
    fn action_for(state: &GuiCommandState, document: u64) -> Option<GuiShellAction> {
        Some(GuiShellAction::ClearSelectedBodyAppearance {
            document,
            key: state.selected_body_key.clone()?,
        })
    }
}
fn confirm_clear(
    ui: &Ui,
    state: &GuiCommandState,
    action: Option<GuiShellAction>,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    ui.text("Clear the selected body's appearance override?");
    let valid = action.as_ref().is_some_and(|value| value.enabled(state));
    if !valid {
        ui.text_wrapped("Select a body with an appearance override to continue.");
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
    fn clear_target_tracks_current_body_selection() {
        let mut state = GuiCommandState::default();
        assert!(ClearAppearance::action_for(&state, 7).is_none());
        state.selected_body_key = Some("first".into());
        let first = ClearAppearance::action_for(&state, 7).unwrap();
        state.selected_body_key = Some("second".into());
        let second = ClearAppearance::action_for(&state, 7).unwrap();
        assert_eq!(
            first,
            GuiShellAction::ClearSelectedBodyAppearance {
                document: 7,
                key: "first".into(),
            }
        );
        assert_eq!(
            second,
            GuiShellAction::ClearSelectedBodyAppearance {
                document: 7,
                key: "second".into(),
            }
        );
    }
}
