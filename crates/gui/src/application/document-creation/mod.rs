// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui};

/// Built-in part templates specify display units without changing internal geometry units.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiPartTemplate {
    #[default]
    Metric,
    Inch,
}
impl GuiPartTemplate {
    /// Return the canonical length unit; example: `GuiPartTemplate::Metric.length_unit() == "mm"`.
    pub fn length_unit(self) -> &'static str {
        match self {
            Self::Metric => "mm",
            Self::Inch => "in",
        }
    }
    fn folder(self) -> &'static str {
        match self {
            Self::Metric => "Metric",
            Self::Inch => "English",
        }
    }
    fn title(self) -> &'static str {
        match self {
            Self::Metric => "Standard metric part",
            Self::Inch => "Standard inch part",
        }
    }
}
#[derive(Default)]
pub(crate) struct TemplateChooser {
    requested: bool,
    folder: GuiPartTemplate,
    selected: Option<GuiPartTemplate>,
}
impl TemplateChooser {
    pub(crate) fn request(&mut self) {
        self.requested = true;
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if ui.io().key_ctrl && ui.is_key_pressed(imgui::Key::N) && !state.transaction_open {
            self.request();
        }
        if std::mem::take(&mut self.requested) {
            self.selected = None;
            ui.open_popup("New document");
            dialog_geometry(ui);
        }
        let Some(_dialog) = ui.begin_modal_popup("New document") else {
            return;
        };
        self.contents(ui);
        self.buttons(ui, state, actions);
    }
    fn contents(&mut self, ui: &Ui) {
        ui.columns(3, "template-layout", true);
        ui.text("Template folders");
        self.folders(ui);
        ui.next_column();
        ui.text("Part");
        if crate::widget_abi::dialog_selectable(
            ui,
            self.folder.title(),
            self.selected == Some(self.folder),
        ) {
            self.selected = Some(self.folder);
        }
        ui.next_column();
        self.details(ui);
        ui.columns(1, "template-layout-end", false);
    }
    fn folders(&mut self, ui: &Ui) {
        for folder in [GuiPartTemplate::Metric, GuiPartTemplate::Inch] {
            if crate::widget_abi::dialog_selectable(ui, folder.folder(), self.folder == folder) {
                self.folder = folder;
            }
        }
    }
    fn details(&self, ui: &Ui) {
        ui.text("Template details");
        let Some(selected) = self.selected else {
            ui.text_wrapped("Select a template.");
            return;
        };
        ui.text_wrapped(selected.title());
        ui.text("Document type: Part");
        ui.text(match selected {
            GuiPartTemplate::Metric => "Length: millimeters (mm)",
            GuiPartTemplate::Inch => "Length: inches (in)",
        });
    }
    fn buttons(&self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        ui.separator();
        let disabled = ui.begin_disabled(self.selected.is_none() || state.transaction_open);
        if crate::widget_abi::button(ui, "Create", [0.; 2])
            && let Some(template) = self.selected
        {
            actions.push(GuiShellAction::CreatePart(template));
            ui.close_current_popup();
        }
        drop(disabled);
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel", [0.; 2]) || ui.is_key_pressed(imgui::Key::Escape)
        {
            ui.close_current_popup();
        }
    }
}
fn dialog_geometry(ui: &Ui) {
    let size = [
        720_f32.min(ui.io().display_size[0] - 24.),
        220_f32.min(ui.io().display_size[1] - 24.),
    ];
    let position = std::array::from_fn(|axis| (ui.io().display_size[axis] - size[axis]) * 0.5);
    crate::widget_abi::window_geometry(ui, position, size, Condition::Appearing);
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/templates.rs"]
mod tests;
