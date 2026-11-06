// SPDX-License-Identifier: GPL-2.0-only
use crate::application::document_saving::{
    close_prompt::{CloseOutcome, ClosePrompt},
    first_saves::FirstSaves,
    folder_browser::DialogEnd,
    open_dialog::OpenDialog,
    save_as::SaveAsDialog,
};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

const NOT_AVAILABLE: &str = "This command is not available in this build.";
#[derive(Default)]
pub(crate) struct GuiDocumentFiles {
    templates: crate::application::document_creation::TemplateChooser,
    open: OpenDialog,
    save_as: SaveAsDialog,
    close_prompt: ClosePrompt,
    first_saves: FirstSaves,
    options_requested: bool,
    properties_requested: bool,
}
impl GuiDocumentFiles {
    pub(crate) fn request_save_path(
        &mut self,
        document: u64,
        name: &str,
        kind: Option<oblikovati_api::types::DocumentType>,
    ) -> Result<(), String> {
        if document == 0 {
            return Err(format!(
                "save dialog document {document}: expected a nonzero document identity"
            ));
        }
        self.save_as.request(document, name, kind, false);
        Ok(())
    }
    pub(crate) fn menu(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        ui.menu("File", || {
            if ui
                .menu_item_config("New")
                .enabled(!state.transaction_open)
                .build()
            {
                self.templates.request();
            }
            self.open_menu_item(ui, state);
            file_save_item(ui, state, actions);
            self.save_as_menu_item(ui, state);
            file_close_item(ui, state, actions);
            unavailable_file_item(ui, "Export", NOT_AVAILABLE);
            unavailable_file_item(ui, "Share", NOT_AVAILABLE);
            unavailable_file_item(ui, "Manage", NOT_AVAILABLE);
            self.document_properties_item(ui, state);
            unavailable_file_item(ui, "Print", NOT_AVAILABLE);
            ui.separator();
            ui.text_disabled("Recent documents");
            ui.text_disabled("No recent documents");
            ui.separator();
            self.options_item(ui);
            if ui.menu_item("Exit") {
                actions.push(GuiShellAction::Exit);
            }
        });
    }
    fn document_properties_item(&mut self, ui: &Ui, state: &GuiCommandState) {
        self.properties_requested |= ui
            .menu_item_config("Document Properties")
            .enabled(state.document.is_some())
            .build();
    }
    fn options_item(&mut self, ui: &Ui) {
        self.options_requested |= ui.menu_item("Options...");
    }
    /// Consume the File menu's request; example: `files.take_options_request()`.
    pub(crate) fn take_options_request(&mut self) -> bool {
        std::mem::take(&mut self.options_requested)
    }
    /// Consume the File menu's request; example: `files.take_properties_request()`.
    pub(crate) fn take_properties_request(&mut self) -> bool {
        std::mem::take(&mut self.properties_requested)
    }
    /// Show a captured window picture in Save As.
    pub(crate) fn receive_window_preview(&mut self, image: crate::GuiPreviewImage) {
        self.save_as.receive_window_preview(image);
    }
    fn open_menu_item(&mut self, ui: &Ui, state: &GuiCommandState) {
        if ui
            .menu_item_config("Open...")
            .enabled(!state.transaction_open)
            .build()
        {
            self.open.request();
        }
    }
    fn save_as_menu_item(&mut self, ui: &Ui, state: &GuiCommandState) {
        let chosen = ui
            .menu_item_config("Save As...")
            .enabled(state.document.is_some() && !state.transaction_open)
            .build();
        if let Some(document) = state.document.filter(|_| chosen) {
            self.save_as
                .request(document, active_name(state), state.document_type, false);
        }
    }
    pub(crate) fn validate(&self) -> Result<(), String> {
        self.open.validate()?;
        self.save_as.validate()
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.templates.panel(ui, state, actions);
        self.open.panel(ui, state, actions);
    }
    /// Route this frame's closes, then draw the prompts and Save As. Runs after every control has
    /// drawn, and answered closes join `actions` after the routing so no answer is asked twice.
    /// Example: `files.closing_panel(ui, &state, &mut actions)` at the end of the frame.
    pub(crate) fn closing_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let routed = self.close_prompt.intercept(state, actions);
        self.apply_close_outcome(routed, actions);
        let answered = self.close_prompt.panel(ui, state);
        self.apply_close_outcome(answered, actions);
        self.next_first_save(state, actions);
        if self.save_as.panel(ui, state, actions) == Some(DialogEnd::Cancelled) {
            self.first_saves.cancel();
        }
    }
    fn apply_close_outcome(&mut self, outcome: CloseOutcome, actions: &mut Vec<GuiShellAction>) {
        actions.extend(outcome.actions);
        self.first_saves.extend(outcome.first_saves);
    }
    fn next_first_save(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        if self.save_as.is_open() {
            return;
        }
        if let Some(document) = self.first_saves.next(state, actions) {
            self.save_as.request(
                document,
                document_name(state, document),
                state.document_type,
                true,
            );
        }
    }
    /// A dialog must open or a held close must continue; GUI frames otherwise wait for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.close_prompt.needs_frame()
            || self.first_saves.is_waiting()
            || self.save_as.is_opening()
            || self.open.is_opening()
    }
}
fn document_name(state: &GuiCommandState, document: u64) -> &str {
    state
        .documents
        .iter()
        .find(|row| row.id == document)
        .map_or("Part", |row| row.name.as_str())
}
fn active_name(state: &GuiCommandState) -> &str {
    state
        .document
        .map_or("Part", |document| document_name(state, document))
}
fn unavailable_file_item(ui: &Ui, label: &str, reason: &str) {
    ui.menu_item_config(label).enabled(false).build();
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(reason);
    }
}
fn file_save_item(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let can_save = state.document.is_some() && !state.transaction_open;
    if ui.menu_item_config("Save").enabled(can_save).build() {
        actions.push(GuiShellAction::Save);
    }
}
fn file_close_item(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let chosen = ui
        .menu_item_config("Close")
        .enabled(state.document.is_some() && !state.transaction_open)
        .build();
    if let Some(id) = state.document.filter(|_| chosen) {
        actions.push(GuiShellAction::CloseDocument(
            oblikovati_api::wire::CloseDocumentArgs { id, force: false },
        ));
    }
}
#[cfg(test)]
#[path = "../tests/files/unit.rs"]
mod tests;
