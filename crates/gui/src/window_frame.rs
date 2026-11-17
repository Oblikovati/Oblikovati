// SPDX-License-Identifier: GPL-2.0-only
//! The application-owned window controls share the top bar with quick access.
use crate::{GuiFrameRequest, GuiShellAction, GuiWindowAction};
use imgui::{Condition, MouseButton, Ui};

const CONTROL_WIDTH: f32 = 48.;

pub(crate) fn draw(
    ui: &Ui,
    request: &GuiFrameRequest,
    bar_height: f32,
    actions: &mut Vec<GuiShellAction>,
) -> (Option<GuiWindowAction>, crate::GuiWindowControls) {
    let width = request.extent[0] as f32 / request.layout_scale()[0];
    let mut action = None;
    let mut bounds = [[0.; 4]; 3];
    let mut focused = None;
    crate::widget_abi::window_geometry(
        ui,
        [width - CONTROL_WIDTH * 3., 0.],
        [CONTROL_WIDTH * 3., bar_height],
        Condition::Always,
    );
    let _padding = ui.push_style_var(imgui::StyleVar::WindowPadding([0., 0.]));
    let _spacing = ui.push_style_var(imgui::StyleVar::ItemSpacing([0., 0.]));
    ui.window("##application-window-controls")
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .scroll_bar(false)
        .scrollable(false)
        .build(|| {
            if control(
                ui,
                "_###window-minimize",
                "Minimize",
                bar_height,
                0,
                &mut bounds,
                &mut focused,
            ) {
                action = Some(GuiWindowAction::Minimize);
            }
            ui.same_line();
            let (symbol, title) = if request.commands.window_maximized {
                ("[][]###window-maximize", "Restore")
            } else {
                ("[]###window-maximize", "Maximize")
            };
            if control(ui, symbol, title, bar_height, 1, &mut bounds, &mut focused) {
                action = Some(GuiWindowAction::ToggleMaximize);
            }
            ui.same_line();
            if control(
                ui,
                "X###window-close",
                "Close",
                bar_height,
                2,
                &mut bounds,
                &mut focused,
            ) {
                actions.push(GuiShellAction::Exit);
            }
        });
    let scale = request.layout_scale();
    for bounds in &mut bounds {
        for axis in 0..2 {
            bounds[axis] *= scale[axis];
            bounds[axis + 2] *= scale[axis];
        }
    }
    (
        action.or_else(|| bar_gesture(ui, width, bar_height)),
        crate::GuiWindowControls { bounds, focused },
    )
}
fn control(
    ui: &Ui,
    label: &str,
    accessible: &str,
    bar_height: f32,
    index: usize,
    bounds: &mut [[f32; 4]; 3],
    focused: &mut Option<usize>,
) -> bool {
    let clicked = crate::widget_abi::button(ui, label, [CONTROL_WIDTH, bar_height]);
    record_control(ui, index, bounds, focused);
    if ui.is_item_hovered() {
        ui.tooltip_text(accessible);
    }
    clicked
}
fn bar_gesture(ui: &Ui, width: f32, bar_height: f32) -> Option<GuiWindowAction> {
    let [x, y] = ui.io().mouse_pos;
    if y < 0.
        || y >= bar_height
        || x < 0.
        || x >= width - CONTROL_WIDTH * 3.
        || ui.is_any_item_hovered()
        || ui.is_any_item_active()
    {
        return None;
    }
    if ui.is_mouse_double_clicked(MouseButton::Left) {
        return Some(GuiWindowAction::ToggleMaximize);
    }
    if ui.is_mouse_dragging(MouseButton::Left) {
        return Some(GuiWindowAction::Move);
    }
    None
}

fn record_control(ui: &Ui, index: usize, bounds: &mut [[f32; 4]; 3], focused: &mut Option<usize>) {
    let min = ui.item_rect_min();
    let max = ui.item_rect_max();
    bounds[index] = [min[0], min[1], max[0], max[1]];
    if ui.is_item_focused() {
        *focused = Some(index);
    }
}
