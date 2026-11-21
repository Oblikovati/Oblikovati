// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiSelectionMode, GuiShellAction};
use imgui::Ui;

const MENU: &str = "##quick-access-selection";

pub(super) fn button(
    ui: &Ui,
    icons: &crate::icons::GuiRibbonIcons,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    open: &mut bool,
) {
    let disabled =
        ui.begin_disabled(!GuiShellAction::SetSelectionMode(GuiSelectionMode::All).enabled(state));
    let requested = std::mem::take(open);
    if icons.compact_button(ui, "Select", crate::icons::IconGlyph::Select) || requested {
        ui.open_popup(MENU);
    }
    drop(disabled);
    if let Some(_menu) = ui.begin_popup(MENU) {
        choices(ui, state, actions);
        ui.dummy([0., 2.]);
    }
}

fn choices(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    if ui.menu_item("Selection Filter…") {
        actions.push(GuiShellAction::OpenSelectionFilter);
    }
    ui.separator();
    for (label, mode) in [
        ("All Types", GuiSelectionMode::All),
        ("Faces", GuiSelectionMode::Faces),
        ("Components", GuiSelectionMode::Components),
    ] {
        if ui
            .menu_item_config(label)
            .selected(state.selection_mode == mode)
            .build()
        {
            actions.push(GuiShellAction::SetSelectionMode(mode));
        }
    }
}
