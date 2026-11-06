// SPDX-License-Identifier: GPL-2.0-only
//! The Projects dialog the template chooser opens: what is loaded, and which project is active.
use crate::{GuiCommandState, GuiProject};
use imgui::Ui;

const TITLE: &str = "Projects";
const NOT_AVAILABLE: &str = "This command is not available in this build.";

#[derive(Default)]
pub(crate) struct ProjectsDialog {
    opening: bool,
}
impl ProjectsDialog {
    pub(crate) fn request(&mut self) {
        self.opening = true;
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening
    }
    /// Draw inside the chooser, so it stacks above it; example: `dialog.panel(ui, &state)`.
    pub(crate) fn panel(&mut self, ui: &Ui, state: &GuiCommandState) {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(TITLE);
        }
        let Some(_dialog) = ui.begin_modal_popup(TITLE) else {
            return;
        };
        project_rows(ui, &state.projects);
        ui.separator();
        settings_tree(ui, state.projects.iter().find(|project| project.active));
        ui.separator();
        for label in ["New", "Browse", "Save", "Apply"] {
            muted_button(ui, label);
            ui.same_line();
        }
        if crate::widget_abi::button(ui, "Done", [80., 0.]) {
            ui.close_current_popup();
        }
    }
}
fn project_rows(ui: &Ui, projects: &[GuiProject]) {
    if projects.is_empty() {
        ui.text_disabled("No projects loaded");
    }
    for project in projects {
        ui.text(project_row_text(project));
    }
}
fn project_row_text(project: &GuiProject) -> String {
    let mark = if project.active { "(active) " } else { "" };
    format!("{mark}{}  {}", project.name, project.location)
}
fn settings_tree(ui: &Ui, active: Option<&GuiProject>) {
    let Some(project) = active else {
        return;
    };
    if let Some(_settings) = ui
        .tree_node_config("Project Settings")
        .default_open(true)
        .push()
    {
        for (label, value) in &project.settings {
            ui.bullet_text(format!("{label}: {value}"));
        }
    }
}
fn muted_button(ui: &Ui, label: &str) {
    {
        let _muted = ui.begin_disabled(true);
        crate::widget_abi::button(ui, label, [0.; 2]);
    }
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text(NOT_AVAILABLE);
    }
}
#[cfg(test)]
#[path = "../../../tests/application/document-creation/projects.rs"]
mod tests;
