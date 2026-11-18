// SPDX-License-Identifier: GPL-2.0-only
//! The application-owned window controls share the top bar with quick access.
use crate::{GuiFrameRequest, GuiShellAction, GuiWindowAction};
use imgui::{Condition, MouseButton, Ui};

const CONTROL_WIDTH: f32 = 48.;

#[derive(Default)]
pub(crate) struct FrameInteraction {
    drag_armed: bool,
    popup_press: crate::popup_press::PopupPress<usize>,
}

#[derive(Default)]
struct ControlResults {
    bounds: [[f32; 4]; 3],
    focused: Option<usize>,
}

pub(crate) fn draw(
    ui: &Ui,
    request: &GuiFrameRequest,
    bar_height: f32,
    actions: &mut Vec<GuiShellAction>,
    interaction: &mut FrameInteraction,
    focus: Option<usize>,
) -> (Option<GuiWindowAction>, crate::GuiWindowControls) {
    let width = request.extent[0] as f32 / request.layout_scale()[0];
    let mut action = None;
    let mut controls = ControlResults::default();
    crate::widget_abi::window_geometry(
        ui,
        [width - CONTROL_WIDTH * 3., 0.],
        [CONTROL_WIDTH * 3., bar_height],
        Condition::Always,
    );
    let _padding = ui.push_style_var(imgui::StyleVar::WindowPadding([0., 0.]));
    let _spacing = ui.push_style_var(imgui::StyleVar::ItemSpacing([0., 0.]));
    ui.window("##application-window-controls")
        .focused(focus.is_some())
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .collapsible(false)
        .scroll_bar(false)
        .scrollable(false)
        .build(|| {
            if focus == Some(0) {
                ui.set_keyboard_focus_here();
            }
            if control(
                ui,
                "###window-minimize",
                "Minimize",
                bar_height,
                0,
                &mut controls,
                interaction,
            ) {
                action = Some(GuiWindowAction::Minimize);
            }
            ui.same_line();
            let (symbol, title) = if request.commands.window_maximized {
                ("###window-maximize", "Restore")
            } else {
                ("###window-maximize", "Maximize")
            };
            if focus == Some(1) {
                ui.set_keyboard_focus_here();
            }
            if control(ui, symbol, title, bar_height, 1, &mut controls, interaction) {
                action = Some(GuiWindowAction::ToggleMaximize);
            }
            ui.same_line();
            if focus == Some(2) {
                ui.set_keyboard_focus_here();
            }
            if control(
                ui,
                "###window-close",
                "Close",
                bar_height,
                2,
                &mut controls,
                interaction,
            ) {
                actions.push(GuiShellAction::Exit);
            }
        });
    interaction.popup_press.finish_frame(ui);
    let scale = request.layout_scale();
    for bounds in &mut controls.bounds {
        for axis in 0..2 {
            bounds[axis] *= scale[axis];
            bounds[axis + 2] *= scale[axis];
        }
    }
    (
        action.or_else(|| bar_gesture(ui, width, bar_height, &mut interaction.drag_armed)),
        crate::GuiWindowControls {
            bounds: controls.bounds,
            focused: controls.focused,
        },
    )
}
fn control(
    ui: &Ui,
    label: &str,
    accessible: &str,
    bar_height: f32,
    index: usize,
    controls: &mut ControlResults,
    interaction: &mut FrameInteraction,
) -> bool {
    let clicked = crate::widget_abi::button(ui, label, [CONTROL_WIDTH, bar_height]);
    let through_popup = interaction.popup_press.released(ui, index);
    let clicked = clicked || through_popup;
    let clicked = clicked || crate::accessibility::focused_activation(ui);
    record_control(ui, index, &mut controls.bounds, &mut controls.focused);
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

fn record_control(ui: &Ui, index: usize, bounds: &mut [[f32; 4]; 3], focused: &mut Option<usize>) {
    let min = ui.item_rect_min();
    let max = ui.item_rect_max();
    bounds[index] = [min[0], min[1], max[0], max[1]];
    if ui.is_item_focused() {
        *focused = Some(index);
    }
}
