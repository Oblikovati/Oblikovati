// SPDX-License-Identifier: GPL-2.0-only
//! The private editor for a Loft's intermediate section.
use super::*;

const TITLE: &str = "Section Dimensions";

#[derive(Default)]
pub(super) struct SectionDimensions {
    active: bool,
    position_mm: f64,
    area_scale: f64,
}

impl SectionDimensions {
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        state: Option<GuiLoftPlacedView>,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let editing = state.filter(|view| view.editing);
        if let Some(view) = editing.filter(|_| !self.active) {
            self.position_mm = view.section.position * view.guide_length_mm;
            self.area_scale = view.section.area_scale;
            ui.open_popup(TITLE);
        }
        self.active = editing.is_some();
        let Some(_popup) = ui
            .modal_popup_config(TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        let Some(view) = editing else {
            ui.close_current_popup();
            return;
        };
        self.fields(ui);
        let section =
            SketchLoftPlacedSection::new(self.position_mm / view.guide_length_mm, self.area_scale)
                .ok();
        if crate::theme::property_accept(ui, section.is_some())
            && let Some(section) = section
        {
            actions.push(GuiShellAction::Loft(GuiLoftAction::ApplyPlaced {
                document,
                section,
            }));
            ui.close_current_popup();
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel", [58., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            actions.push(GuiShellAction::Loft(GuiLoftAction::CancelPlaced(document)));
            ui.close_current_popup();
        }
    }

    fn fields(&mut self, ui: &Ui) {
        ui.input_scalar("Position (mm)", &mut self.position_mm)
            .display_format("%.3f")
            .build();
        ui.input_scalar("Alternate Section Size", &mut self.area_scale)
            .display_format("%.3f")
            .build();
    }
}
