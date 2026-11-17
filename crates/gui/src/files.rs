// SPDX-License-Identifier: GPL-2.0-only
use crate::application::document_opening::OpenDialog;
use crate::application::document_saving::{
    close_prompt::{CloseOutcome, ClosePrompt},
    first_saves::FirstSaves,
    folder_browser::DialogEnd,
    save_as::SaveAsDialog,
};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
#[path = "files/new_menu.rs"]
mod new_menu;

const NOT_AVAILABLE: &str = "This command is not available in this build.";
const SAVE_AS_COMMANDS: [(&str, &str); 3] = [
    (
        "Save Copy As",
        "Save a copy under a new name; the original file remains open.",
    ),
    (
        "Save Copy As Template",
        "Save a copy to the template folder.",
    ),
    (
        "Pack and Go",
        "Gather this file and what it references into one folder.",
    ),
];
const FILE_MENU_WIDTH: f32 = 440.;
#[derive(Default)]
pub(crate) struct GuiDocumentFiles {
    templates: crate::application::document_creation::TemplateChooser,
    projects: crate::application::project_dialog::ProjectsDialog,
    recent_menu: RecentMenu,
    open: OpenDialog,
    save_as: SaveAsDialog,
    close_prompt: ClosePrompt,
    first_saves: FirstSaves,
    menu_requested: bool,
    options_requested: bool,
    properties_requested: bool,
    /// Home covers the active document, so commands about that document have nothing to act on.
    pub home_in_front: bool,
}
/// How the File menu orders and shows its recent files.
#[derive(Default)]
struct RecentMenu {
    by_date: bool,
    with_locations: bool,
}
impl RecentMenu {
    fn rows(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        self.controls(ui);
        if state.recent_documents.is_empty() {
            ui.text_disabled("No recent documents");
        }
        let mut rows: Vec<&crate::GuiRecentDocument> = state.recent_documents.iter().collect();
        if self.by_date {
            rows.sort_by_key(|row| std::cmp::Reverse(row.modified));
        }
        for row in rows.into_iter().take(RECENT_MENU_ROWS) {
            self.row(ui, row, actions);
        }
    }
    fn controls(&mut self, ui: &Ui) {
        let ordering = if self.by_date {
            "By Access Date"
        } else {
            "By Ordered List"
        };
        if ui.button(ordering) {
            self.by_date = !self.by_date;
        }
        ui.same_line();
        let display = if self.with_locations {
            "Name and Location"
        } else {
            "Name"
        };
        if ui.button(format!("{display}###recent-display")) {
            self.with_locations = !self.with_locations;
        }
    }
    fn row(&self, ui: &Ui, row: &crate::GuiRecentDocument, actions: &mut Vec<GuiShellAction>) {
        use crate::application::home_recent_documents::recent_rows::kind_mark;
        ui.text(kind_mark(row.kind));
        ui.same_line_with_pos(RECENT_NAME_X);
        if ui.button(format!("{}###recent-menu-{}", row.name, row.path)) {
            actions.push(GuiShellAction::OpenDocument(
                oblikovati_api::wire::OpenDocumentArgs {
                    full_document_name: row.path.clone(),
                    ..Default::default()
                },
            ));
        }
        ui.same_line_with_pos(RECENT_PIN_X);
        let pin = if row.pinned { "Unpin" } else { "Pin" };
        if ui.button(format!("{pin}###recent-menu-pin-{}", row.path)) {
            actions.push(GuiShellAction::Recent(crate::GuiRecentAction::SetPinned {
                path: row.path.clone(),
                pinned: !row.pinned,
            }));
        }
        if self.with_locations {
            ui.text_disabled(&row.location);
        }
    }
}
const RECENT_MENU_ROWS: usize = 8;
const RECENT_NAME_X: f32 = 22.;
const RECENT_PIN_X: f32 = 190.;
impl GuiDocumentFiles {
    pub(crate) fn request_application_close(&mut self) {
        self.close_prompt.request_application_exit();
    }
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
        show_key_tip: bool,
    ) {
        if std::mem::take(&mut self.menu_requested) {
            ui.open_popup("File");
        }
        ui.menu("File", || {
            // Matching widths keep the first popup frame and its column clips aligned.
            ui.columns(2, "file-menu-layout", false);
            self.file_commands(ui, state, actions);
            ui.next_column();
            ui.text_disabled("Recent documents");
            self.recent_menu.rows(ui, state, actions);
            ui.columns(1, "file-menu-layout-end", false);
            ui.separator();
            self.options_item(ui);
            if ui.menu_item("Exit") {
                actions.push(GuiShellAction::Exit);
            }
            // Columns take their width from the menu, so the menu needs its own.
            crate::widget_abi::dummy(ui, [FILE_MENU_WIDTH, 0.]);
        });
        if show_key_tip {
            crate::ribbon_key_tips::badge(ui, "F");
        }
    }
    fn file_commands(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.new_menu_item(ui, state);
        self.open_menu_item(ui, state);
        file_save_item(ui, state, actions);
        self.save_as_menu_item(ui, state);
        file_close_item(ui, state, actions);
        unavailable_file_item(ui, "Export", NOT_AVAILABLE);
        unavailable_file_item(ui, "Share", NOT_AVAILABLE);
        unavailable_file_item(ui, "Manage", NOT_AVAILABLE);
        self.document_properties_item(ui, state);
        unavailable_file_item(ui, "Print", NOT_AVAILABLE);
    }
    fn document_properties_item(&mut self, ui: &Ui, state: &GuiCommandState) {
        self.properties_requested |= ui
            .menu_item_config("Document Properties")
            .enabled(state.document.is_some() && !self.home_in_front)
            .build();
    }
    fn options_item(&mut self, ui: &Ui) {
        self.options_requested |= ui.menu_item("Options...");
    }
    /// Request the File menu from a keyboard shortcut; example: `files.request_menu()`.
    pub(crate) fn request_menu(&mut self) {
        self.menu_requested = true;
    }
    /// Open the new-document chooser; example: from the Quick Access Toolbar.
    pub(crate) fn request_new(&mut self) {
        self.templates.request();
    }
    /// Open project settings; example: `files.request_projects()` from quick access.
    pub(crate) fn request_projects(&mut self) {
        self.projects.request();
    }
    /// Open the Open dialog; example: from the Quick Access Toolbar.
    pub(crate) fn request_open(&mut self) {
        self.open.request();
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
    pub(crate) fn receive_isometric_preview(
        &mut self,
        document: u64,
        image: crate::GuiPreviewImage,
    ) {
        self.save_as.receive_isometric_preview(document, image);
    }
    pub(crate) fn reject_isometric_preview(&mut self, document: u64, cause: String) {
        self.save_as.reject_isometric_preview(document, cause);
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
    /// Save As opens beside the menu with its related commands; only Save As itself is available.
    fn save_as_menu_item(&mut self, ui: &Ui, state: &GuiCommandState) {
        let enabled = state.document.is_some() && !state.transaction_open && !self.home_in_front;
        ui.menu_with_enabled("Save As", enabled, || {
            let chosen = ui.menu_item("Save As...");
            ui.text_disabled("Save under a new name and keep working on the new file.");
            if let Some(document) = state.document.filter(|_| chosen) {
                self.save_as
                    .request(document, active_name(state), state.document_type, false);
            }
            for (label, description) in SAVE_AS_COMMANDS {
                unavailable_file_item(ui, label, NOT_AVAILABLE);
                ui.text_disabled(description);
            }
        });
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
        icons: &crate::icons::GuiRibbonIcons,
    ) {
        self.templates.panel(ui, state, actions, icons);
        self.open.panel(ui, state, actions);
        self.projects.panel(ui, state);
    }
    pub(crate) fn answer_close_prompt(&mut self, choice: crate::GuiClosePromptChoice) {
        self.close_prompt.answer_accessibly(choice);
    }
    pub(crate) fn close_prompt_semantics(&self) -> Option<crate::GuiClosePrompt> {
        self.close_prompt.semantics()
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
        let exit_saves = self.close_prompt.advance_application_exit(
            state,
            actions,
            self.save_as.is_open() || self.first_saves.is_waiting(),
        );
        self.first_saves.extend(exit_saves);
        let routed = self.close_prompt.intercept(state, actions);
        self.apply_close_outcome(routed, actions);
        let answered = self.close_prompt.panel(ui, state);
        self.apply_close_outcome(answered, actions);
        self.next_first_save(state, actions);
        match self.save_as.panel(ui, state, actions) {
            Some(DialogEnd::Cancelled) => {
                self.first_saves.cancel();
                self.close_prompt.cancel_application_exit();
            }
            Some(DialogEnd::Accepted) => self.close_prompt.resume_application_exit(),
            None => {}
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
            || self.templates.needs_frame()
            || self.projects.needs_frame()
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
