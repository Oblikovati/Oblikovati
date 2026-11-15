// SPDX-License-Identifier: GPL-2.0-only
//! View commands keep their menus and window actions in icon-only layouts.
use super::*;

pub(super) fn windows(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let icons = owners.icons;
    let (position, size) = icons.appearance.cell(0);
    icons.cursor(ui, origin, position);
    owners.workspace.user_interface_menu(ui, size, Some(icons));
    let (position, size) = icons.appearance.cell(1);
    icons.cursor(ui, origin, position);
    owners
        .workspace
        .switcher
        .menu(ui, size, state, actions, Some(icons));
    let (position, size) = icons.appearance.cell(2);
    icons.cursor(ui, origin, position);
    icons.command_button(
        ui,
        "New",
        GuiShellAction::NewDocumentWindow,
        state,
        size,
        actions,
    );
}

pub(super) fn theme(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    editor: &mut crate::theme_editor::GuiThemeEditor,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    use crate::ribbon_command_layout::RibbonAppearance;
    if let Some(theme) = &state.theme {
        theme.controls(ui, actions);
    }
    let column = 176. + ui.calc_text_size("Colour scheme")[0];
    let second = if icons.appearance == RibbonAppearance::Compact {
        [column, 0.]
    } else {
        [0., icons.appearance.cell(1).0[1]]
    };
    icons.cursor(ui, origin, second);
    if let Some(scheme) = &state.color_scheme {
        scheme.controls(ui, actions);
    }
    if state.theme.is_none() {
        return false;
    }
    let edit_x = if icons.appearance == RibbonAppearance::Compact {
        2. * column
    } else {
        column
    };
    icons.cursor(ui, origin, [edit_x, 0.]);
    crate::theme_editor::launcher_icon(ui, editor, icons, icons.appearance.cell(0).1)
}
