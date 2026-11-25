// SPDX-License-Identifier: GPL-2.0-only
//! The component a populated drawing layout asks for before its drawing exists.
use crate::application::document_saving::folder_browser::{
    BROWSER_COLUMN_WIDTH, DialogEnd, DialogEscape, FileTypeList, FolderBrowser, center_dialog,
    dialog_buttons, file_name_field, file_type_field, full_path,
};
use crate::application::file_system::NativeFolders;
use crate::{
    GuiCommandState, GuiDrawingLayoutAction, GuiLayoutComponent, GuiProjectTemplate, GuiShellAction,
};
use imgui::Ui;
use std::rc::Rc;

const TITLE: &str = "Select Component";
const FILE_CHOOSER: &str = "Select Component File";
const WARNING: &str = "Component Needs Updating";
const NOT_AVAILABLE: &str = "Placing a drawing view is not available in this build.";

pub(crate) struct ComponentDialog {
    browser: FolderBrowser,
    file_type: usize,
    /// The component's document: an open document's name, or a chosen file's path.
    name: String,
    /// The nested chooser's own field, which a Cancel there discards.
    file_name: String,
    file_escape: DialogEscape,
    opening: bool,
    escape: DialogEscape,
    template: Option<GuiProjectTemplate>,
    proposed: Option<(u64, String)>,
    warning_open: bool,
    suppress_warning: bool,
}
impl Default for ComponentDialog {
    fn default() -> Self {
        Self {
            browser: FolderBrowser::new(Rc::new(NativeFolders)),
            file_type: 0,
            name: String::new(),
            file_name: String::new(),
            file_escape: DialogEscape::default(),
            opening: false,
            escape: DialogEscape::default(),
            template: None,
            proposed: None,
            warning_open: false,
            suppress_warning: false,
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
    pub(super) fn request_layout(
        &mut self,
        name: &str,
        id: Option<u64>,
        template: Option<GuiProjectTemplate>,
    ) {
        self.request(name);
        self.template = template;
        self.proposed = id.map(|id| (id, name.to_owned()));
    }

    pub(super) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.component_panel(ui, actions);
        self.warning_panel(ui, state, actions);
    }

    fn component_panel(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
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
        self.buttons(ui, actions);
    }
    /// A nested chooser: Open replaces the document name, and Cancel leaves it as it was.
    fn file_chooser(&mut self, ui: &Ui) {
        center_dialog(ui, [860., 400.]);
        let Some(_chooser) = ui.begin_modal_popup(FILE_CHOOSER) else {
            return;
        };
        let types = FileTypeList::any();
        ui.columns(2, "component-layout", true);
        ui.set_column_width(0, BROWSER_COLUMN_WIDTH);
        if let Some(name) = self.browser.controls(ui, "Look in", types.filter(0)) {
            self.file_name = name;
        }
        ui.next_column();
        ui.child_window("##component-preview")
            .size([0., 110.])
            .border(true)
            .build(|| ui.text_disabled("Preview"));
        ui.columns(1, "component-layout-end", false);
        let entered = file_name_field(ui, &mut self.file_name);
        file_type_field(ui, &types, &mut self.file_type);
        let can_open = !self.file_name.is_empty();
        let end = dialog_buttons(ui, "Open", can_open, entered, &mut self.file_escape);
        if end == Some(DialogEnd::Accepted) {
            self.name = full_path(&self.browser.folder, &self.file_name);
        }
        if end.is_some() {
            ui.close_current_popup();
        }
    }
    fn buttons(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let disabled = ui.begin_disabled(self.template.is_none() || self.name.trim().is_empty());
        let confirm = crate::widget_abi::button(ui, "OK", [80., 0.]);
        drop(disabled);
        if self.template.is_none()
            && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
        {
            ui.tooltip_text(NOT_AVAILABLE);
        }
        if confirm && let Some(template) = self.template.take() {
            let component = match &self.proposed {
                Some((id, name)) if *name == self.name => GuiLayoutComponent::Open(*id),
                _ => GuiLayoutComponent::Named(self.name.trim().to_owned()),
            };
            actions.push(GuiShellAction::DrawingLayout(
                GuiDrawingLayoutAction::Confirm {
                    template: Box::new(template),
                    component,
                },
            ));
            ui.close_current_popup();
        } else if ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            actions.push(GuiShellAction::DrawingLayout(
                GuiDrawingLayoutAction::Cancel,
            ));
            ui.close_current_popup();
        }
        ui.same_line();
        let escaped = self.escape.pressed(ui);
        if crate::widget_abi::button(ui, "Cancel", [80., 0.]) || escaped {
            actions.push(GuiShellAction::DrawingLayout(
                GuiDrawingLayoutAction::Cancel,
            ));
            ui.close_current_popup();
        }
    }

    fn warning_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(component) = state.layout_warning.as_ref() else {
            self.warning_open = false;
            return;
        };
        if !self.warning_open {
            self.warning_open = true;
            self.suppress_warning = false;
            ui.open_popup(WARNING);
        }
        center_dialog(ui, [510., 160.]);
        let Some(_warning) = ui.begin_modal_popup(WARNING) else {
            return;
        };
        ui.text_wrapped(format!("{component} needs updating."));
        ui.text_wrapped("The drawing reflects the model's non-updated state.");
        ui.checkbox("Do not show again this session", &mut self.suppress_warning);
        if crate::widget_abi::button(ui, "OK", [80., 0.]) {
            actions.push(GuiShellAction::DrawingLayout(
                GuiDrawingLayoutAction::Acknowledge {
                    suppress: self.suppress_warning,
                },
            ));
            ui.close_current_popup();
        }
    }
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/component_dialog.rs"]
mod tests;
