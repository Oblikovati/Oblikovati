// SPDX-License-Identifier: GPL-2.0-only
//! The Line command's pointer fields and its active context menu.
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{StyleColor, Ui, WindowFlags};

/// One numeric field beside the pointer; example: the length field reading `10.271 mm`.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiSketchHudField {
    pub text: String,
    /// A typed value holds the field: a padlock says the pointer no longer moves it.
    pub locked: bool,
    pub active: bool,
    pub invalid: bool,
    pub selected: bool,
}
/// The length and angle fields of a pending segment, at the owner's window pixel.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiSketchHud {
    pub pixel: [i32; 2],
    pub fields: Vec<GuiSketchHudField>,
}
/// What the running Line command offers now; example: `close_enabled` after two segments.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct GuiLineCommandMenu {
    pub close_enabled: bool,
    pub restart_enabled: bool,
    pub auto_project: bool,
}
/// One choice from the running Line command's menu.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiLineCommand {
    Cancel,
    Ok,
    Close,
    Restart,
    ToggleAutoProject,
}

const HUD_OFFSET: [f32; 2] = [18., 14.];
const FIELD_WIDTH: f32 = 86.;
const PADLOCK_WIDTH: f32 = 7.;
const INVALID_TEXT: [f32; 4] = [0.95, 0.2, 0.2, 1.];
const MENU_ID: &str = "##line-command-menu";

/// Draw the pending segment's fields beside the pointer; they follow it and take no input.
/// True while the pointer rests on the fields: a click there is theirs, not the sketch's.
pub(crate) fn hud_panel(ui: &Ui, request: &GuiFrameRequest) -> bool {
    let Some(hud) = &request.commands.sketch_hud else {
        return false;
    };
    let scale = request.layout_scale();
    let position =
        std::array::from_fn(|axis| hud.pixel[axis] as f32 / scale[axis] + HUD_OFFSET[axis]);
    ui.window("##sketch-hud")
        .position(position, imgui::Condition::Always)
        .flags(hud_flags())
        .build(|| {
            for (index, field) in hud.fields.iter().enumerate() {
                if index > 0 {
                    ui.same_line();
                }
                hud_field(ui, field);
            }
            let [x, y] = ui.io().mouse_pos;
            let [left, top] = ui.window_pos();
            let [width, height] = ui.window_size();
            (x >= left && x <= left + width) && (y >= top && y <= top + height)
        })
        .unwrap_or(false)
}
fn hud_flags() -> WindowFlags {
    WindowFlags::NO_DECORATION
        | WindowFlags::ALWAYS_AUTO_RESIZE
        | WindowFlags::NO_MOVE
        | WindowFlags::NO_INPUTS
        | WindowFlags::NO_SAVED_SETTINGS
        | WindowFlags::NO_FOCUS_ON_APPEARING
        | WindowFlags::NO_NAV
}
fn hud_field(ui: &Ui, field: &GuiSketchHudField) {
    let [x, y] = ui.cursor_screen_pos();
    let height = ui.text_line_height();
    let border = if field.active {
        ui.style_color(StyleColor::ButtonActive)
    } else {
        ui.style_color(StyleColor::Border)
    };
    let draw = ui.get_window_draw_list();
    if field.selected {
        let highlight = ui.style_color(StyleColor::TextSelectedBg);
        draw.add_rect([x, y], [x + FIELD_WIDTH, y + height], highlight)
            .filled(true)
            .build();
    }
    draw.add_rect(
        [x - 2., y - 1.],
        [x + FIELD_WIDTH + 2., y + height + 1.],
        border,
    )
    .build();
    if field.locked {
        let color = ui.style_color(StyleColor::Text);
        draw_padlock(&draw, color, [x + FIELD_WIDTH - PADLOCK_WIDTH, y], height);
    }
    drop(draw);
    if field.invalid {
        ui.text_colored(INVALID_TEXT, &field.text);
    } else {
        ui.text(&field.text);
    }
    ui.same_line_with_pos(x - ui.window_pos()[0] + FIELD_WIDTH + 8.);
    ui.dummy([0., height]);
}
/// A body with a shackle above it, sized from the text line it sits in.
fn draw_padlock(draw: &imgui::DrawListMut, color: [f32; 4], [x, y]: [f32; 2], height: f32) {
    let body_top = y + height * 0.45;
    draw.add_rect([x, body_top], [x + PADLOCK_WIDTH, y + height - 1.], color)
        .filled(true)
        .build();
    let shackle = [x + PADLOCK_WIDTH / 2., body_top];
    draw.add_circle(shackle, PADLOCK_WIDTH / 3., color)
        .num_segments(12)
        .build();
}
/// Open on the pointer's release over the workspace: the press itself belongs to no panel.
pub(crate) fn command_menu(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(menu) = state.line_command else {
        return;
    };
    let over_panel = ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel());
    if !over_panel && ui.is_mouse_released(imgui::MouseButton::Right) {
        ui.open_popup(MENU_ID);
    }
    let Some(_popup) = ui.begin_popup(MENU_ID) else {
        return;
    };
    command_rows(ui, menu, actions);
}
fn command_rows(ui: &Ui, menu: GuiLineCommandMenu, actions: &mut Vec<GuiShellAction>) {
    use GuiLineCommand::{Cancel, Close, Ok, Restart, ToggleAutoProject};
    let mut choose = |command| actions.push(GuiShellAction::LineCommand(command));
    if ui.menu_item("Cancel (ESC)") {
        choose(Cancel);
    }
    if ui.menu_item("OK") {
        choose(Ok);
    }
    ui.separator();
    if ui
        .menu_item_config("Close")
        .enabled(menu.close_enabled)
        .build()
    {
        choose(Close);
    }
    if ui
        .menu_item_config("Restart")
        .enabled(menu.restart_enabled)
        .build()
    {
        choose(Restart);
    }
    ui.separator();
    ui.menu_item_config("Line").selected(true).build();
    if ui
        .menu_item_config("AutoProject")
        .selected(menu.auto_project)
        .build()
    {
        choose(ToggleAutoProject);
    }
}
