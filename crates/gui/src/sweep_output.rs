// SPDX-License-Identifier: GPL-2.0-only
//! Sweep result controls and their current document names.
use super::*;

/// The solid output authored by a Sweep.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiSweepOutput {
    #[default]
    NewSolid,
    Join,
    Cut,
}

pub(super) const EDIT_WARNING: &str = "Edit Sweep Feature";

pub(super) fn cut_warning<'a>(
    preview: Option<&'a GuiSweepPreviewState>,
    request: &GuiSweepRequest,
) -> Option<&'a str> {
    let preview = preview.filter(|preview| preview.request == *request)?;
    match &preview.result {
        GuiSweepPreview::UnchangedCut(cause) => Some(cause),
        _ => None,
    }
}

impl SweepDialog {
    pub(super) fn draw_cut_warning(ui: &Ui, cause: Option<&str>) -> bool {
        let Some(_popup) = ui
            .modal_popup_config(EDIT_WARNING)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return false;
        };
        let Some(cause) = cause else {
            ui.close_current_popup();
            return false;
        };
        ui.text("The Sweep edit could not be accepted.");
        if ui.collapsing_header("Details", imgui::TreeNodeFlags::empty()) {
            ui.text_wrapped(cause);
        }
        let cancel = ui.button("Cancel") || ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if cancel {
            ui.close_current_popup();
        }
        cancel
    }

    pub(super) fn draw_operation(&mut self, ui: &Ui, state: &GuiCommandState) {
        let Some(browser) = &state.browser else {
            return;
        };
        let existing = browser
            .bodies
            .iter()
            .find(|body| Some(body.feature) == self.feature);
        let saved_boolean = self
            .feature
            .and_then(|id| browser.sweep_edits.get(&id))
            .is_some_and(|seed| seed.output != GuiSweepOutput::NewSolid);
        ui.radio_button("New Solid", &mut self.output, GuiSweepOutput::NewSolid);
        ui.same_line_with_pos(135.);
        let boolean_available = self.output != GuiSweepOutput::NewSolid
            || browser
                .bodies
                .iter()
                .filter(|body| Some(body.feature) != self.feature)
                .count()
                == 1
            || saved_boolean;
        let disabled = ui.begin_disabled(!boolean_available);
        ui.radio_button("Join", &mut self.output, GuiSweepOutput::Join);
        ui.same_line_with_pos(210.);
        ui.radio_button("Cut", &mut self.output, GuiSweepOutput::Cut);
        drop(disabled);
        if self.output == GuiSweepOutput::NewSolid {
            crate::theme::property_label(ui, "Body Name");
            if let Some(body) = existing.filter(|_| !saved_boolean) {
                ui.text(&body.name);
            } else {
                ui.text(format!("Solid{}", browser.bodies.len() + 1));
            }
        }
    }
}
