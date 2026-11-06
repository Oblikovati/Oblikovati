// SPDX-License-Identifier: GPL-2.0-only
//! The component a populated drawing layout asks for before its drawing exists.
use crate::application::document_saving::folder_browser::{
    DialogEnd, DialogEscape, FileTypeList, FolderBrowser, center_dialog, dialog_buttons,
    file_name_field, full_path,
};
use crate::application::file_system::NativeFolders;
use imgui::Ui;
use std::rc::Rc;

const TITLE: &str = "Select Component";
const FILE_CHOOSER: &str = "Select Component File";
const NOT_AVAILABLE: &str = "Placing a drawing view is not available in this build.";

pub(crate) struct ComponentDialog {
    browser: FolderBrowser,
    /// The component's document: an open document's name, or a chosen file's path.
    name: String,
    /// The nested chooser's own field, which a Cancel there discards.
    file_name: String,
    file_escape: DialogEscape,
    opening: bool,
    escape: DialogEscape,
}
impl Default for ComponentDialog {
    fn default() -> Self {
        Self {
            browser: FolderBrowser::new(Rc::new(NativeFolders)),
            name: String::new(),
            file_name: String::new(),
            file_escape: DialogEscape::default(),
            opening: false,
            escape: DialogEscape::default(),
        }
    }
}
impl ComponentDialog {
    /// Propose the document to show; example: `dialog.request("Part1")` for the active part.
    pub(crate) fn request(&mut self, name: &str) {
        self.name = name.to_string();
        self.opening = true;
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening
    }
    /// Draw the dialog; example: `dialog.panel(ui)` once the chooser has closed.
    pub(crate) fn panel(&mut self, ui: &Ui) {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(TITLE);
        }
        center_dialog(ui, [560., 380.]);
        let Some(_dialog) = ui.begin_modal_popup(TITLE) else {
            return;
        };
        ui.text("Document name");
        ui.same_line();
        ui.set_next_item_width(-96.);
        ui.input_text("##component-document", &mut self.name)
            .build();
        ui.same_line();
        if crate::widget_abi::button(ui, "Browse...", [0.; 2]) {
            self.browser.refresh();
            self.file_name.clear();
            ui.open_popup(FILE_CHOOSER);
        }
        self.file_chooser(ui);
        ui.separator();
        self.buttons(ui);
    }
    /// A nested chooser: Open replaces the document name, and Cancel leaves it as it was.
    fn file_chooser(&mut self, ui: &Ui) {
        center_dialog(ui, [680., 400.]);
        let Some(_chooser) = ui.begin_modal_popup(FILE_CHOOSER) else {
            return;
        };
        if let Some(name) = self
            .browser
            .controls(ui, "Look in", FileTypeList::any().filter(0))
        {
            self.file_name = name;
        }
        let entered = file_name_field(ui, &mut self.file_name);
        let can_open = !self.file_name.is_empty();
        let end = dialog_buttons(ui, "Open", can_open, entered, &mut self.file_escape);
        if end == Some(DialogEnd::Accepted) {
            self.name = full_path(&self.browser.folder, &self.file_name);
        }
        if end.is_some() {
            ui.close_current_popup();
        }
    }
    /// Placing the view needs a drawing engine, so OK stays muted and says why.
    fn buttons(&mut self, ui: &Ui) {
        {
            let _muted = ui.begin_disabled(true);
            crate::widget_abi::button(ui, "OK", [80., 0.]);
        }
        if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
            ui.tooltip_text(NOT_AVAILABLE);
        }
        ui.same_line();
        let escaped = self.escape.pressed(ui);
        if crate::widget_abi::button(ui, "Cancel", [80., 0.]) || escaped {
            ui.close_current_popup();
        }
    }
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/component_dialog.rs"]
mod tests;
