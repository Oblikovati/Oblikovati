// SPDX-License-Identifier: GPL-2.0-only
//! The value editor a double-click opens on a dimension annotation.
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{Key, Ui};

/// What a label's editor needs; example: `d34` and its stored expression `4 mm`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDimensionEdit {
    pub sketch: u64,
    pub constraint: u64,
    pub name: String,
    pub expression: String,
}
/// One accepted value; the owner re-solves the sketch through history.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDimensionValue {
    pub sketch: u64,
    pub constraint: u64,
    pub expression: String,
}
/// The open editor; closing it by any route drops what was typed.
#[derive(Default)]
pub(crate) struct GuiDimensionValueEditor {
    open: Option<GuiDimensionEdit>,
    submitted: Option<GuiDimensionEdit>,
    closed_this_frame: bool,
    /// The pointer is on a label that opens an editor, so its clicks are not a viewport pick.
    pub over_label: bool,
    text: String,
    focus: bool,
    warning_requested: bool,
    warning_active: bool,
    position: [f32; 2],
}
const TITLE: &str = "Edit Dimension";
const SIZE: [f32; 2] = [240., 70.];
const INVALID_TEXT: [f32; 4] = [0.95, 0.2, 0.2, 1.];

impl GuiDimensionValueEditor {
    pub(crate) fn reject(&mut self, value: &GuiDimensionValue) {
        let Some(edit) = self.submitted.take() else {
            return;
        };
        if edit.sketch != value.sketch || edit.constraint != value.constraint {
            return;
        }
        self.text.clone_from(&value.expression);
        self.open = Some(edit);
        self.warning_requested = true;
        self.warning_active = true;
    }
    /// The editor owned this frame's keys; the Escape that closed it was not the command's.
    pub(crate) fn had_keyboard(&self) -> bool {
        self.open.is_some() || self.closed_this_frame
    }
    /// Open on a double-click over a label, then draw while open.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.closed_this_frame = false;
        self.open_at_pointer(ui, request);
        let state = &request.commands;
        if !state.sketch_at_ready && !state.dimension_tool_active {
            self.open = None;
        }
        let Some(edit) = self.open.clone() else {
            return;
        };
        let mut close = false;
        crate::widget_abi::fit_dialog(ui, TITLE, self.position, SIZE);
        ui.window(TITLE).collapsible(false).build(|| {
            ui.text(&edit.name);
            ui.same_line();
            if std::mem::take(&mut self.focus) {
                ui.set_keyboard_focus_here();
            }
            let invalid = !expression_syntax_valid(&self.text);
            let red = invalid.then(|| ui.push_style_color(imgui::StyleColor::Text, INVALID_TEXT));
            let accepted = ui
                .input_text("##dimension-value", &mut self.text)
                .auto_select_all(true)
                .enter_returns_true(true)
                .build();
            drop(red);
            if accepted {
                self.submitted = Some(edit.clone());
                actions.push(GuiShellAction::EditDimensionValue(GuiDimensionValue {
                    sketch: edit.sketch,
                    constraint: edit.constraint,
                    expression: self.text.clone(),
                }));
                close = true;
            }
            close |= !self.warning_active && ui.is_key_pressed_no_repeat(Key::Escape);
            close |= self.warning(ui);
        });
        if close {
            self.open = None;
            self.closed_this_frame = true;
            // Keyboard focus was this editor's; the sketch's own keys work again without a click.
            crate::widget_abi::release_keyboard_focus(ui);
        }
    }
    fn warning(&mut self, ui: &Ui) -> bool {
        const WARNING: &str = "Edit Dimension Value";
        if std::mem::take(&mut self.warning_requested) {
            ui.open_popup(WARNING);
        }
        let Some(_warning) = ui.begin_modal_popup(WARNING) else {
            return false;
        };
        ui.text("Failed to edit dimension value.");
        ui.text("The expression cannot be evaluated.");
        if ui.button("OK") || ui.is_key_pressed_no_repeat(Key::Enter) {
            ui.close_current_popup();
            self.warning_active = false;
            self.focus = true;
        }
        ui.same_line();
        if ui.button("Cancel") || ui.is_key_pressed_no_repeat(Key::Escape) {
            ui.close_current_popup();
            self.warning_active = false;
            return true;
        }
        false
    }
    fn open_at_pointer(&mut self, ui: &Ui, request: &GuiFrameRequest) {
        self.over_label = false;
        let state = &request.commands;
        if !(state.sketch_at_ready || state.dimension_tool_active)
            || ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel())
        {
            return;
        }
        let Some(edit) = label_under_pointer(ui, request, &request.commands) else {
            return;
        };
        self.over_label = true;
        // At rest a label opens on a double-click; the running command takes a single one.
        let opens = if state.dimension_tool_active {
            ui.is_mouse_clicked(imgui::MouseButton::Left)
        } else {
            ui.is_mouse_double_clicked(imgui::MouseButton::Left)
        };
        if !opens {
            return;
        }
        self.text.clone_from(&edit.expression);
        self.open = Some(edit);
        self.focus = true;
        self.position = ui.io().mouse_pos;
    }
}
fn expression_syntax_valid(source: &str) -> bool {
    let units = oblikovati_model::units::UnitsOfMeasure::default();
    oblikovati_model::expression::Expression::parse(source, &units).is_ok()
}

fn label_under_pointer(
    ui: &Ui,
    request: &GuiFrameRequest,
    state: &GuiCommandState,
) -> Option<GuiDimensionEdit> {
    let [x, y] = ui.io().mouse_pos;
    let scale = request.layout_scale();
    // Labels can overlap on a small shape; the one nearest the pointer is the one meant.
    let hits = state.dimension_labels.iter().filter_map(|label| {
        let size = ui.calc_text_size(&label.text);
        let [cx, cy] = [
            label.pixel[0] as f32 / scale[0],
            label.pixel[1] as f32 / scale[1],
        ];
        let hit = (x - cx).abs() <= size[0] / 2. + 2. && (y - cy).abs() <= size[1] / 2. + 2.;
        let edit = label.edit.clone()?;
        if hit {
            return Some(((x - cx).hypot(y - cy), edit));
        }
        let [from, to] = label.line?;
        let logical = |point: [i32; 2]| [point[0] as f32 / scale[0], point[1] as f32 / scale[1]];
        let distance = line_distance([x, y], logical(from), logical(to));
        (distance <= 5.).then_some((distance, edit))
    });
    hits.min_by(|first, second| first.0.total_cmp(&second.0))
        .map(|(_, edit)| edit)
}
fn line_distance(point: [f32; 2], from: [f32; 2], to: [f32; 2]) -> f32 {
    let direction = [to[0] - from[0], to[1] - from[1]];
    let length_squared = direction[0] * direction[0] + direction[1] * direction[1];
    if length_squared == 0. {
        return (point[0] - from[0]).hypot(point[1] - from[1]);
    }
    let along = (((point[0] - from[0]) * direction[0] + (point[1] - from[1]) * direction[1])
        / length_squared)
        .clamp(0., 1.);
    (point[0] - from[0] - along * direction[0]).hypot(point[1] - from[1] - along * direction[1])
}

#[cfg(test)]
mod tests {
    use super::expression_syntax_valid;

    #[test]
    fn syntax_feedback_accepts_references_and_marks_incomplete_formulas() {
        for expression in ["2*6", "width * 2", "2 mm * 6", "radius"] {
            assert!(expression_syntax_valid(expression), "{expression:?}");
        }
        for expression in ["", "2*", "("] {
            assert!(!expression_syntax_valid(expression), "{expression:?}");
        }
    }
    #[test]
    fn dimension_line_hit_distance_is_limited_to_the_segment() {
        assert_eq!(super::line_distance([5., 3.], [0., 0.], [10., 0.]), 3.);
        assert_eq!(super::line_distance([13., 4.], [0., 0.], [10., 0.]), 5.);
        assert_eq!(super::line_distance([3., 4.], [0., 0.], [0., 0.]), 5.);
    }
}
