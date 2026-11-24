// SPDX-License-Identifier: GPL-2.0-only
//! The Projects dialog the template chooser opens: what is loaded, and which project is active.
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiProject, GuiProjectGroup};
use imgui::{
    Condition, Key, SelectableFlags, TableColumnFlags, TableColumnSetup, TreeNodeFlags, Ui,
};

#[path = "new_project_wizard.rs"]
mod new_project_wizard;

const TITLE: &str = "Projects";
const SIZE: [f32; 2] = [760., 520.];
const NOT_AVAILABLE: &str = "This command is not available in this build.";

#[derive(Default)]
pub(crate) struct ProjectsDialog {
    opening: bool,
    open: crate::dockable_lifecycle::DockableState,
    dock_title: String,
    /// The row a click chose to inspect; choosing one does not activate it.
    inspected: Option<String>,
    /// The settings group the arrow keys open and close.
    selected_group: Option<String>,
    open_groups: Vec<String>,
    wizard: new_project_wizard::NewProjectWizard,
}
impl DockableWindow for ProjectsDialog {}
impl ProjectsDialog {
    pub(crate) fn register_dockable(
        &mut self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
        identity: &str,
    ) -> Result<(), String> {
        self.dock_title = format!("Projects###{identity}");
        registry.register::<Self>(&self.dock_title, &self.open)
    }
    pub(crate) fn request(&mut self) {
        self.opening = true;
        self.inspected = None;
    }
    /// Close with its parent; example: `projects.close()` when the chooser closes.
    pub(crate) fn close(&mut self) {
        self.open.set_open(false);
        self.opening = false;
        self.wizard.cancel();
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening || self.wizard.needs_frame()
    }
    /// Draw inside the chooser, so it stacks above it; example: `dialog.panel(ui, &state)`.
    pub(crate) fn panel(&mut self, ui: &Ui, state: &GuiCommandState) {
        if std::mem::take(&mut self.opening) {
            self.open.set_open(true);
        }
        if !self.open.is_open() {
            return;
        }
        let title = if self.dock_title.is_empty() {
            TITLE.to_owned()
        } else {
            self.dock_title.clone()
        };
        Self::center_dockable(ui, &title, SIZE);
        let mut open = true;
        let mut done = false;
        Self::dockable_window(ui, &title, &mut open).build(|| {
            self.project_rows(ui, &state.projects);
            ui.separator();
            self.settings_tree(ui, &state.projects);
            ui.separator();
            if crate::widget_abi::button(ui, "New", [0.; 2]) {
                self.wizard.request();
            }
            ui.same_line();
            for label in ["Browse", "Save", "Apply"] {
                muted_button(ui, label);
                ui.same_line();
            }
            done = crate::widget_abi::button(ui, "Done", [80., 0.])
                || !crate::widget_abi::any_popup_open(ui)
                    && ui.is_window_focused()
                    && ui.is_key_pressed_no_repeat(Key::Escape);
            self.wizard.panel(ui);
        });
        if !open || done {
            self.close();
        }
    }
    fn project_rows(&mut self, ui: &Ui, projects: &[GuiProject]) {
        if projects.is_empty() {
            ui.text_disabled("No projects loaded");
        }
        self.project_table(ui, projects);
        if let Some(provisional) = self.wizard.provisional_name() {
            ui.text_disabled(provisional);
        }
    }
    fn project_table(&mut self, ui: &Ui, projects: &[GuiProject]) {
        let Some(_table) = ui.begin_table("project-list", 2) else {
            return;
        };
        let mut name = TableColumnSetup::new("Name");
        name.flags = TableColumnFlags::WIDTH_FIXED;
        name.init_width_or_weight = ui.current_font_size() * 13.;
        ui.table_setup_column_with(name);
        ui.table_setup_column("Location");
        ui.table_headers_row();
        let shown = self.shown_project(projects).map(|row| row.location.clone());
        for project in projects {
            if project_row(ui, project, shown.as_deref()) {
                self.inspected = Some(project.location.clone());
            }
        }
    }
    /// The project the tree describes: the row a click chose, or else the active one.
    fn shown_project<'p>(&self, projects: &'p [GuiProject]) -> Option<&'p GuiProject> {
        let inspected = self.inspected.as_deref();
        projects
            .iter()
            .find(|row| Some(row.location.as_str()) == inspected)
            .or_else(|| projects.iter().find(|row| row.active))
    }
    fn settings_tree(&mut self, ui: &Ui, projects: &[GuiProject]) {
        let Some(project) = self.shown_project(projects).cloned() else {
            return;
        };
        let Some(_settings) = ui
            .tree_node_config("Project Settings")
            .default_open(true)
            .push()
        else {
            return;
        };
        ui.bullet();
        ui.text_wrapped(format!("Location: {}", project.location));
        for (label, value) in &project.settings {
            ui.bullet_text(format!("{label}: {value}"));
        }
        for group in &project.groups {
            self.settings_group(ui, group);
        }
    }
    fn settings_group(&mut self, ui: &Ui, group: &GuiProjectGroup) {
        let selected = self.selected_group.as_deref() == Some(group.label.as_str());
        self.follow_arrow_keys(ui, &group.label, selected);
        let open = self.open_groups.contains(&group.label);
        let mut flags = TreeNodeFlags::OPEN_ON_ARROW | TreeNodeFlags::SPAN_AVAIL_WIDTH;
        if selected {
            flags |= TreeNodeFlags::SELECTED;
        }
        let token = ui
            .tree_node_config(&group.label)
            .flags(flags)
            .opened(open, Condition::Always)
            .push();
        if ui.is_item_clicked() {
            self.selected_group = Some(group.label.clone());
        }
        if token.is_some() {
            for row in &group.rows {
                ui.bullet_text(row);
            }
        }
    }
    /// Right opens the selected group and Left closes it; no other group changes.
    fn follow_arrow_keys(&mut self, ui: &Ui, label: &str, selected: bool) {
        if !selected {
            return;
        }
        if ui.is_key_pressed_no_repeat(Key::RightArrow)
            && !self.open_groups.iter().any(|g| g == label)
        {
            self.open_groups.push(label.to_owned());
        }
        if ui.is_key_pressed_no_repeat(Key::LeftArrow) {
            self.open_groups.retain(|group| group != label);
        }
    }
}
fn project_row(ui: &Ui, project: &GuiProject, inspected: Option<&str>) -> bool {
    ui.table_next_row();
    ui.table_next_column();
    let _identity = ui.push_id(&project.location);
    let chose = choose_project_name(
        ui,
        &project_row_name(project),
        inspected == Some(project.location.as_str()),
    );
    ui.table_next_column();
    ui.text_wrapped(&project.location);
    chose
}

fn choose_project_name(ui: &Ui, label: &str, selected: bool) -> bool {
    let cursor = ui.cursor_pos();
    let width = ui.content_region_avail()[0].max(1.);
    let height = ui.calc_text_size_with_opts(label, false, width)[1];
    let chose = ui
        .selectable_config("##project-name")
        .selected(selected)
        .size([0., height])
        // Inside a dialog a plain selectable would close it.
        .flags(SelectableFlags::DONT_CLOSE_POPUPS | SelectableFlags::SPAN_ALL_COLUMNS)
        .build();
    ui.set_cursor_pos(cursor);
    ui.text_wrapped(label);
    chose
}
fn project_row_name(project: &GuiProject) -> String {
    let mark = if project.active { "(active) " } else { "" };
    format!("{mark}{}", project.name)
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
