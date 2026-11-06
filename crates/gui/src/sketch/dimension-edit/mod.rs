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
    text: String,
    focus: bool,
    position: [f32; 2],
}
const TITLE: &str = "Edit Dimension";
const SIZE: [f32; 2] = [240., 70.];

impl GuiDimensionValueEditor {
    /// Open on a double-click over a label, then draw while open.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        request: &GuiFrameRequest,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.open_at_pointer(ui, request);
        if !request.commands.sketch_at_ready {
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
            let accepted = ui
                .input_text("##dimension-value", &mut self.text)
                .auto_select_all(true)
                .enter_returns_true(true)
                .build();
            if accepted {
                actions.push(GuiShellAction::EditDimensionValue(GuiDimensionValue {
                    sketch: edit.sketch,
                    constraint: edit.constraint,
                    expression: self.text.clone(),
                }));
                close = true;
            }
            close |= ui.is_key_pressed_no_repeat(Key::Escape);
        });
        if close {
            self.open = None;
        }
    }
    fn open_at_pointer(&mut self, ui: &Ui, request: &GuiFrameRequest) {
        if !request.commands.sketch_at_ready
            || !ui.is_mouse_double_clicked(imgui::MouseButton::Left)
            || ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel())
        {
            return;
        }
        let Some(edit) = label_under_pointer(ui, request, &request.commands) else {
            return;
        };
        self.text.clone_from(&edit.expression);
        self.open = Some(edit);
        self.focus = true;
        self.position = ui.io().mouse_pos;
    }
}
fn label_under_pointer(
    ui: &Ui,
    request: &GuiFrameRequest,
    state: &GuiCommandState,
) -> Option<GuiDimensionEdit> {
    let [x, y] = ui.io().mouse_pos;
    let scale = request.layout_scale();
    state.dimension_labels.iter().find_map(|label| {
        let size = ui.calc_text_size(&label.text);
        let [cx, cy] = [
            label.pixel[0] as f32 / scale[0],
            label.pixel[1] as f32 / scale[1],
        ];
        let hit = (x - cx).abs() <= size[0] / 2. + 2. && (y - cy).abs() <= size[1] / 2. + 2.;
        hit.then(|| label.edit.clone()).flatten()
    })
}
