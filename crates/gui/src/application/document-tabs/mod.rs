// SPDX-License-Identifier: GPL-2.0-only
//! Reaching open documents without the tab strip, and the commands on a tab.
#[path = "properties.rs"]
pub(crate) mod properties;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::{ActivateDocumentArgs, CloseDocumentArgs, DocumentInfo};

const SELECT_WINDOW: &str = "Select Window";
const SWITCH_MENU: &str = "##switch-windows";

fn visible(state: &GuiCommandState) -> impl Iterator<Item = &DocumentInfo> {
    state.documents.iter().filter(|row| row.visible)
}
fn activate(id: u64) -> GuiShellAction {
    GuiShellAction::ActivateDocument(ActivateDocumentArgs { id })
}
fn close(id: u64) -> GuiShellAction {
    GuiShellAction::CloseDocument(CloseDocumentArgs { id, force: false })
}
/// The View tab's Switch menu and its extended list.
#[derive(Debug, Default)]
pub(crate) struct WindowSwitcher {
    opening: bool,
    selected: Option<u64>,
}
impl WindowSwitcher {
    /// Draw the Switch button; example: `switcher.menu(ui, [108., 26.], state, actions)`.
    pub(crate) fn menu(
        &mut self,
        ui: &Ui,
        size: [f32; 2],
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if crate::widget_abi::button(ui, "Switch", size) {
            ui.open_popup(SWITCH_MENU);
        }
        let Some(_menu) = ui.begin_popup(SWITCH_MENU) else {
            return;
        };
        for row in visible(state) {
            let chosen = ui
                .menu_item_config(&row.name)
                .selected(state.document == Some(row.id))
                .build();
            if chosen {
                actions.push(activate(row.id));
            }
        }
        ui.separator();
        if ui.menu_item("More Windows...") {
            self.opening = true;
            self.selected = state.document;
        }
    }
    /// A frame must follow so the extended list opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening
    }
    /// Draw the extended list; example: `switcher.panel(ui, state, actions)`.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(SELECT_WINDOW);
        }
        let Some(_dialog) = ui.begin_modal_popup(SELECT_WINDOW) else {
            return;
        };
        self.document_list(ui, state);
        if crate::widget_abi::button(ui, "OK", [80., 0.]) {
            actions.extend(self.selected.map(activate));
            ui.close_current_popup();
        }
        ui.same_line();
        let cancelled = crate::widget_abi::button(ui, "Cancel", [80., 0.])
            || ui.is_key_pressed(imgui::Key::Escape);
        if cancelled {
            ui.close_current_popup();
        }
    }
    fn document_list(&mut self, ui: &Ui, state: &GuiCommandState) {
        ui.child_window("##open-documents")
            .size([360., 180.])
            .border(true)
            .build(|| {
                for row in visible(state) {
                    let chosen = self.selected == Some(row.id);
                    if crate::widget_abi::dialog_selectable(ui, &row.name, chosen) {
                        self.selected = Some(row.id);
                    }
                }
            });
    }
}
/// The commands of one tab; `Close Tabs` keeps the tab it was opened on.
#[derive(Default)]
pub(crate) struct TabMenu {
    pub properties: properties::DocumentProperties,
}
impl TabMenu {
    /// Offer the tab's commands on a right click; call right after the tab item.
    pub(crate) fn popup(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        row: &DocumentInfo,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let identity = format!("##tab-menu-{}", row.id);
        if ui.is_item_clicked_with_button(imgui::MouseButton::Right) {
            ui.open_popup(&identity);
        }
        if let Some(_menu) = ui.begin_popup(&identity) {
            self.items(ui, state, row, actions);
        }
    }
    fn items(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        row: &DocumentInfo,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if ui.menu_item_config("Close").shortcut("Ctrl+W").build() {
            actions.push(close(row.id));
        }
        if ui.menu_item("Close Tabs") {
            actions.extend(tab_closes(state, Some(row.id)));
        }
        if ui.menu_item("Close All") {
            actions.extend(tab_closes(state, None));
        }
        if ui.menu_item_config("Save").shortcut("Ctrl+S").build() {
            actions.extend((state.document != Some(row.id)).then(|| activate(row.id)));
            actions.push(GuiShellAction::Save);
        }
        if ui.menu_item("Properties") {
            self.properties.request(state, row.id);
        }
    }
}
fn tab_closes(state: &GuiCommandState, keep: Option<u64>) -> Vec<GuiShellAction> {
    visible(state)
        .filter(|row| Some(row.id) != keep)
        .map(|row| close(row.id))
        .collect()
}
/// Keyboard routes to the same commands as the tab strip; a dialog or menu above keeps the keys.
/// Example: `document_keys(ui, switch_requested, state, actions)` once per frame.
pub(crate) fn document_keys(
    ui: &Ui,
    switch_requested: bool,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    if crate::widget_abi::any_popup_open(ui) {
        return;
    }
    if ui.io().key_ctrl && ui.is_key_pressed(imgui::Key::W) && !state.transaction_open {
        actions.extend(state.document.map(close));
    }
    if switch_requested {
        actions.extend(next_document(state).filter(|action| action.dispatchable(state)));
    }
}
/// The document after the active one in tab order, wrapping past the last.
fn next_document(state: &GuiCommandState) -> Option<GuiShellAction> {
    let rows: Vec<u64> = visible(state).map(|row| row.id).collect();
    let active = rows.iter().position(|id| Some(*id) == state.document)?;
    let next = rows[(active + 1) % rows.len()];
    (next != rows[active]).then(|| activate(next))
}
#[cfg(test)]
#[path = "../../../tests/application/document-tabs/commands.rs"]
mod tests;
