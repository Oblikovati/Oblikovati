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

pub(crate) fn placed_point_press(
    ui: &Ui,
    state: &GuiCommandState,
    positions: &[Option<crate::dimension_labels::LabelPosition>],
) -> Option<GuiShellAction> {
    if !ui.is_mouse_double_clicked(imgui::MouseButton::Left)
        || state.loft_placed.is_none_or(|placed| placed.editing)
    {
        return None;
    }
    let (_, position) = state
        .dimension_labels
        .iter()
        .zip(positions)
        .find(|(label, _)| label.role == crate::GuiDimensionLabelRole::LoftPlacedSection)?;
    let position = position.as_ref()?;
    let [x, y] = ui.io().mouse_pos;
    let [px, py] = position.anchor;
    let in_caption = (x - position.center[0]).abs() <= position.size[0] * 0.5
        && (y - position.center[1]).abs() <= position.size[1] * 0.5;
    if (x - px).hypot(y - py) <= 8. || in_caption {
        Some(GuiShellAction::Loft(GuiLoftAction::EditPlaced(
            state.document?,
        )))
    } else {
        None
    }
}
