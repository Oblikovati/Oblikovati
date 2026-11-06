// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction};
pub(crate) fn append_keyboard_delete(
    ui: &imgui::Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    let action = GuiShellAction::DeleteSketchSelection;
    if !ui.is_key_pressed_no_repeat(imgui::Key::Delete)
        || ui.io().want_text_input
        || ui.is_any_item_active()
        || !action.enabled(state)
        || actions.contains(&action)
    {
        return;
    }
    actions.push(action);
}
