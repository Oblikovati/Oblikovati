// SPDX-License-Identifier: GPL-2.0-only
//! The component a populated drawing layout asks for before its drawing exists.
use crate::application::document_saving::folder_browser::{
    DialogEscape, FileTypeList, FolderBrowser, center_dialog, file_name_field,
};
use crate::application::file_system::NativeFolders;
use imgui::Ui;
use std::rc::Rc;

const TITLE: &str = "Select Component";
const NOT_AVAILABLE: &str = "Placing a drawing view is not available in this build.";

pub(crate) struct ComponentDialog {
    browser: FolderBrowser,
    name: String,
    browsing: bool,
    opening: bool,
    escape: DialogEscape,
}
impl Default for ComponentDialog {
    fn default() -> Self {
        Self {
            browser: FolderBrowser::new(Rc::new(NativeFolders)),
            name: String::new(),
            browsing: false,
            opening: false,
            escape: DialogEscape::default(),
        }
    }
}
impl ComponentDialog {
    pub(crate) fn request(&mut self) {
        self.name.clear();
        self.browsing = false;
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
        file_name_field(ui, &mut self.name);
        if crate::widget_abi::button(ui, "Browse...", [0.; 2]) {
            self.browsing = true;
            self.browser.refresh();
        }
        if self.browsing
            && let Some(name) = self
                .browser
                .controls(ui, "Look in", FileTypeList::any().filter(0))
        {
            self.name = name;
        }
        ui.separator();
        self.buttons(ui);
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
        if crate::widget_abi::button(ui, "Cancel", [80., 0.]) || self.escape.pressed(ui) {
            ui.close_current_popup();
        }
    }
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/component_dialog.rs"]
mod tests;
