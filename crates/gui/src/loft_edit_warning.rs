// SPDX-License-Identifier: GPL-2.0-only
//! A failed Closed Loop edit leaves the committed Loft intact.
use super::*;

const TITLE: &str = "Edit Loft Feature";

#[derive(Default)]
pub(super) struct LoftEditWarning {
    open: bool,
}

impl LoftDialog {
    pub(super) fn accept_request(&self, state: &GuiCommandState) -> Option<GuiLoftAction> {
        let (document, _) = self.document?;
        let ready = GuiLoftAction::Commit(document);
        if !ready.enabled(state)
            || self.manual_mapping
            || self.merge_tangent_faces
            || state.loft_edit_warning.is_some()
            || state.loft_placed.is_some_and(|placed| placed.editing)
            || (self.path_mode != GuiLoftPathMode::Rails && state.loft_center_line.is_none())
        {
            return None;
        }
        if self.closed_loop {
            return (self.editing
                && self.path_mode == GuiLoftPathMode::PlacedSections
                && state.loft_placed.is_some())
            .then_some(GuiLoftAction::AttemptClosedEdit(document));
        }
        Some(ready)
    }
}

impl LoftEditWarning {
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        cause: Option<&str>,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        if cause.is_some() && !self.open {
            ui.open_popup(TITLE);
        }
        self.open = cause.is_some();
        let Some(_popup) = ui
            .modal_popup_config(TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return false;
        };
        let Some(cause) = cause else {
            ui.close_current_popup();
            return false;
        };
        ui.text("The Loft feature edit failed.");
        if ui.collapsing_header("Details", imgui::TreeNodeFlags::empty()) {
            ui.text_wrapped(cause);
        }
        if crate::widget_abi::button(ui, "Edit", [58., 28.]) {
            actions.push(GuiShellAction::Loft(GuiLoftAction::DismissEditWarning(
                document,
            )));
            ui.close_current_popup();
        }
        ui.same_line();
        let cancel = crate::widget_abi::button(ui, "Cancel", [58., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if cancel {
            ui.close_current_popup();
        }
        cancel
    }
}
