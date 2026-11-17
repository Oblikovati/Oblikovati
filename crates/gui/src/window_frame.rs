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
    drag_armed: &mut bool,
) -> Option<GuiWindowAction> {
    let width = request.extent[0] as f32 / request.layout_scale()[0];
    let mut action = None;
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
            if control(ui, "###window-minimize", "Minimize", bar_height) {
                action = Some(GuiWindowAction::Minimize);
            }
            ui.same_line();
            let (symbol, title) = if request.commands.window_maximized {
                ("###window-maximize", "Restore")
            } else {
                ("###window-maximize", "Maximize")
            };
            if control(ui, symbol, title, bar_height) {
                action = Some(GuiWindowAction::ToggleMaximize);
            }
            ui.same_line();
            if control(ui, "###window-close", "Close", bar_height) {
                actions.push(GuiShellAction::Exit);
            }
        });
    action.or_else(|| bar_gesture(ui, width, bar_height, drag_armed))
}
fn control(ui: &Ui, label: &str, accessible: &str, bar_height: f32) -> bool {
    let clicked = crate::widget_abi::button(ui, label, [CONTROL_WIDTH, bar_height]);
    draw_icon(ui, accessible);
    if ui.is_item_hovered() {
        let [x, y] = ui.item_rect_min();
        // The controls touch the viewport edge; leave room for the full tooltip (#4063).
        let text = ui.calc_text_size(accessible);
        crate::widget_abi::window_geometry(
            ui,
            [(x - text[0] - 16.).max(0.), y + bar_height],
            [text[0] + 16., text[1] + 12.],
            Condition::Always,
        );
        ui.tooltip(|| {
            ui.indent_by(4.);
            ui.text(accessible);
        });
    }
    clicked
}
fn draw_icon(ui: &Ui, kind: &str) {
    let min = ui.item_rect_min();
    let max = ui.item_rect_max();
    let center = [(min[0] + max[0]) / 2., (min[1] + max[1]) / 2.];
    let draw = ui.get_window_draw_list();
    let color = ui.style_color(imgui::StyleColor::Text);
    let point = |x: f32, y: f32| [center[0] + x, center[1] + y];
    match kind {
        "Minimize" => {
            draw.add_line(point(-6., 4.), point(6., 4.), color)
                .thickness(1.5)
                .build();
        }
        "Maximize" => {
            draw.add_rect(point(-6., -5.), point(6., 5.), color)
                .thickness(1.5)
                .build();
        }
        "Restore" => {
            draw.add_rect(point(-4., -6.), point(6., 3.), color)
                .thickness(1.5)
                .build();
            draw.add_rect(point(-6., -3.), point(4., 6.), color)
                .thickness(1.5)
                .build();
        }
        "Close" => {
            draw.add_line(point(-5., -5.), point(5., 5.), color)
                .thickness(1.5)
                .build();
            draw.add_line(point(5., -5.), point(-5., 5.), color)
                .thickness(1.5)
                .build();
        }
        _ => unreachable!("window control {kind}: expected a known control"),
    }
}
fn bar_gesture(
    ui: &Ui,
    width: f32,
    bar_height: f32,
    drag_armed: &mut bool,
) -> Option<GuiWindowAction> {
    let [x, y] = ui.io().mouse_pos;
    if ui.is_mouse_clicked(MouseButton::Left) {
        *drag_armed = y >= 0.
            && y < bar_height
            && x >= 0.
            && x < width - CONTROL_WIDTH * 3.
            && !ui.is_any_item_hovered()
            && !ui.is_any_item_active();
    }
    if !ui.is_mouse_down(MouseButton::Left) {
        *drag_armed = false;
    }
    if !*drag_armed {
        return None;
    }
    if ui.is_mouse_double_clicked(MouseButton::Left) {
        return Some(GuiWindowAction::ToggleMaximize);
    }
    ui.is_mouse_dragging(MouseButton::Left)
        .then_some(GuiWindowAction::Move)
}
