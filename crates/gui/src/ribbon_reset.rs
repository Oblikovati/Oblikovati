// SPDX-License-Identifier: GPL-2.0-only
//! Confirm and atomically restore the ribbon's saved defaults.
use super::*;
use imgui::Ui;

const RESET_TITLE: &str = "Reset Ribbon";
const RESET_SIZE: [f32; 2] = [440., 150.];
impl DocumentRibbon {
    /// Ask before restoring ribbon defaults; example: `ribbon.reset_confirmation(ui, state)`.
    pub(crate) fn reset_confirmation(&mut self, ui: &Ui, state: &GuiCommandState) {
        if self.display.take_reset_request() {
            self.reset_error = None;
            self.reset_open = true;
            ui.open_popup(RESET_TITLE);
        }
        if !self.reset_open {
            return;
        }
        let display = ui.io().display_size;
        let origin = std::array::from_fn(|axis| ((display[axis] - RESET_SIZE[axis]) * 0.5).max(0.));
        crate::widget_abi::fit_dialog(ui, RESET_TITLE, origin, RESET_SIZE);
        self.draw_reset_confirmation(ui, state);
    }
    fn draw_reset_confirmation(&mut self, ui: &Ui, state: &GuiCommandState) {
        let Some(_popup) = ui.begin_modal_popup(RESET_TITLE) else {
            return;
        };
        ui.text_wrapped(
            "All ribbon user interface customizations will be removed. The ribbon will return to its default layout. Continue?",
        );
        if let Some(error) = &self.reset_error {
            ui.text_wrapped(error);
        }
        if crate::widget_abi::button(ui, "Yes", [80., 0.]) {
            self.accept_reset(ui, state);
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "No", [80., 0.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.reset_open = false;
            ui.close_current_popup();
        }
    }
    fn accept_reset(&mut self, ui: &Ui, state: &GuiCommandState) {
        match self.reset_to_defaults(state) {
            Ok(()) => {
                self.reset_open = false;
                ui.close_current_popup();
            }
            Err(error) => self.reset_error = Some(error),
        }
    }
    fn reset_to_defaults(&mut self, state: &GuiCommandState) -> Result<(), String> {
        self.customization.reset()?;
        self.display = Default::default();
        self.panels = Default::default();
        self.tips = Default::default();
        let sketch = state.authoring == GuiAuthoringContext::Sketch;
        let tabs = document_tabs(
            &document_ribbon_key(state),
            sketch,
            active_sheet_metal(state),
        );
        self.select_tab(contextual_tab(&tabs, sketch));
        Ok(())
    }
}
