// SPDX-License-Identifier: GPL-2.0-only
//! Explain why a sketch frame cannot be edited inside sketch edit (#4107).
use crate::{GuiAuthoringContext, GuiCommandState};
use imgui::Ui;

pub(super) const LABEL: &str = "Edit Coordinate System";
const MESSAGE: &str = "Edit Coordinate System##sketch-edit-restriction";

pub(super) fn ready(label: &str, state: &GuiCommandState) -> bool {
    label == LABEL && state.authoring == GuiAuthoringContext::Sketch && !state.transaction_open
}
pub(super) fn open(ui: &Ui) {
    ui.open_popup(MESSAGE);
}
pub(super) fn dialog(ui: &Ui) {
    let Some(_popup) = ui.begin_modal_popup(MESSAGE) else {
        return;
    };
    ui.text("Finish the sketch to edit its coordinate system from the browser context menu.");
    ui.text("Editing a finished sketch's coordinate system is not available in this build.");
    if ui.button("OK") || ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
        ui.close_current_popup();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn command_is_available_only_during_unlocked_sketch_edit() {
        let mut state = GuiCommandState::default();
        assert!(!ready(LABEL, &state));
        state.authoring = GuiAuthoringContext::Sketch;
        assert!(ready(LABEL, &state));
        assert!(!ready("Other", &state));
        state.transaction_open = true;
        assert!(!ready(LABEL, &state));
    }
}
