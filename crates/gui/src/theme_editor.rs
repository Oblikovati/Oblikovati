// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiShellAction, GuiThemeState,
    theme_editor_rows::{THEME_COLOR_ROWS, ThemeColorRow},
};
use imgui::{Condition, Ui};
const THEME_EDITOR_POPUP: &str = "Theme editor";

pub(crate) fn launcher(ui: &Ui) {
    ui.same_line();
    if crate::widget_abi::button(ui, "Edit Theme", [100., 19.]) {
        ui.open_popup(THEME_EDITOR_POPUP);
        popup_geometry(ui);
    }
}
pub(crate) fn panel(ui: &Ui, theme: Option<&GuiThemeState>, actions: &mut Vec<GuiShellAction>) {
    let Some(_popup) = ui.begin_modal_popup(THEME_EDITOR_POPUP) else {
        return;
    };
    let Some(theme) = theme else {
        ui.close_current_popup();
        return;
    };
    ui.text(format!("Theme: {}", theme.active_name()));
    editor_actions(ui, theme, actions);
    ui.text("Edits preview immediately. Save keeps them after restart.");
    if !theme.editable() {
        ui.text("Built-in theme is read-only.");
    }
    color_rows(ui, theme, actions);
}
fn popup_geometry(ui: &Ui) {
    let [width, height] = ui.io().display_size;
    let size = [
        (width - 32.).clamp(1., 560.),
        (height - 32.).clamp(1., 620.),
    ];
    let position = [(width - size[0]) * 0.5, (height - size[1]) * 0.5];
    crate::widget_abi::window_geometry(ui, position, size, Condition::Appearing);
}
fn editor_actions(ui: &Ui, theme: &GuiThemeState, actions: &mut Vec<GuiShellAction>) {
    let disabled = ui.begin_disabled(!theme.save_available());
    if crate::widget_abi::button(ui, "Save theme", [100., 23.]) {
        actions.push(GuiShellAction::SaveTheme {
            name: theme.active_name().into(),
        });
    }
    drop(disabled);
    ui.same_line();
    if crate::widget_abi::button(ui, "Close", [70., 23.]) {
        ui.close_current_popup();
    }
}
fn color_rows(ui: &Ui, theme: &GuiThemeState, actions: &mut Vec<GuiShellAction>) {
    let _disabled = ui.begin_disabled(!theme.editable());
    let mut group = "";
    for row in THEME_COLOR_ROWS {
        if row.group != group {
            ui.separator();
            ui.text(row.group);
            group = row.group;
        }
        color_row(ui, theme, row, actions);
    }
}
fn color_row(
    ui: &Ui,
    theme: &GuiThemeState,
    row: &ThemeColorRow,
    actions: &mut Vec<GuiShellAction>,
) {
    let value = &theme.palette()[&row.token];
    let mut channels = [value.r, value.g, value.b, value.a];
    let label = format!("{}##{}", row.label, row.token.0);
    if ui.color_edit4(label, &mut channels) {
        let [r, g, b, a] = channels;
        actions.push(GuiShellAction::EditThemeColor {
            name: theme.active_name().into(),
            token: row.token.clone(),
            value: oblikovati_api::types::Rgba { r, g, b, a },
        });
    }
}
