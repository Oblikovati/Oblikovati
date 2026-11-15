// SPDX-License-Identifier: GPL-2.0-only
//! Stage a body's appearance clear until the user accepts or cancels it.
use crate::{GuiCommandState, GuiShellAction, dockable_window::DockableWindow};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct ClearAppearance {
    target: Option<(u64, String)>,
}
impl DockableWindow for ClearAppearance {}
impl ClearAppearance {
    pub(crate) fn open(&mut self, document: u64, key: String) {
        self.target = Some((document, key));
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(action) = self.action_for(state) else {
            return;
        };
        let mut open = true;
        let mut done = false;
        let width = ui
            .calc_text_size("Select the original body with an appearance override to continue.")[0];
        let size = [width + ui.frame_height() * 2., ui.frame_height() * 6.];
        Self::dockable_window(ui, "Clear Appearance", &mut open)
            .size(size, imgui::Condition::FirstUseEver)
            .build(|| done = confirm_clear(ui, state, action, actions));
        if !open || done {
            self.target = None;
        }
    }
    fn action_for(&mut self, state: &GuiCommandState) -> Option<GuiShellAction> {
        let (document, key) = self.target.clone()?;
        if state.document != Some(document) {
            self.target = None;
            return None;
        }
        Some(GuiShellAction::ClearSelectedBodyAppearance { document, key })
    }
}
fn confirm_clear(
    ui: &Ui,
    state: &GuiCommandState,
    action: GuiShellAction,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    ui.text("Clear the selected body's appearance override?");
    let valid = action.enabled(state);
    if !valid {
        ui.text_wrapped("Select the original body with an appearance override to continue.");
    }
    let disabled = ui.begin_disabled(!valid);
    let accepted = ui.button("OK");
    drop(disabled);
    ui.same_line();
    let cancelled = ui.button("Cancel");
    if accepted {
        actions.push(action);
    }
    accepted || cancelled
}
