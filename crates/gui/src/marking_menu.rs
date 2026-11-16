// SPDX-License-Identifier: GPL-2.0-only
//! Shared radial layout for configurable canvas menus.
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::MarkingMenuView;

pub(crate) type ResolveCommand = fn(&str) -> Option<(&'static str, GuiShellAction)>;

pub(crate) fn draw(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    resolve: ResolveCommand,
) {
    for row in [[7, 0, 1], [6, 8, 2], [5, 4, 3]] {
        for (index, slot) in row.into_iter().enumerate() {
            if index > 0 {
                ui.same_line();
            }
            draw_slot(ui, menu, slot, state, actions, resolve);
        }
    }
    ui.separator();
    draw_overflow(ui, menu, state, actions, resolve);
}
pub(crate) fn draw_classic(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    resolve: ResolveCommand,
) {
    for item in menu.quadrants.iter().flatten() {
        if let Some((label, command)) = resolve(&item.command_id)
            && ui
                .menu_item_config(label)
                .enabled(command.enabled(state))
                .build()
        {
            actions.push(command);
        }
    }
    draw_overflow(ui, menu, state, actions, resolve);
    ui.separator();
}
fn draw_overflow(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    resolve: ResolveCommand,
) {
    for id in menu.overflow.iter().flatten() {
        let Some((label, command)) = resolve(id) else {
            continue;
        };
        if ui
            .menu_item_config(label)
            .enabled(command.enabled(state))
            .build()
        {
            actions.push(command);
        }
    }
}
fn draw_slot(
    ui: &Ui,
    menu: &MarkingMenuView,
    slot: u8,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    resolve: ResolveCommand,
) {
    let item = menu
        .quadrants
        .iter()
        .flatten()
        .find(|item| item.quadrant.0 == slot);
    let Some((label, action)) = item.and_then(|item| resolve(&item.command_id)) else {
        ui.dummy([100., 24.]);
        return;
    };
    let disabled = ui.begin_disabled(!action.enabled(state));
    let pressed = ui.button_with_size(format!("{label}##marking-{slot}"), [100., 24.]);
    drop(disabled);
    if pressed {
        actions.push(action);
        ui.close_current_popup();
    }
}
