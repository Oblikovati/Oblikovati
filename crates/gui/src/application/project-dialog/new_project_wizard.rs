// SPDX-License-Identifier: GPL-2.0-only
//! The new-project wizard: a type, a name and folder, then libraries. A folder browser hangs off
//! its second page. Finishing is not offered yet: this build cannot write a project file.
use crate::application::document_saving::folder_browser::center_dialog;
use crate::application::file_system::{GuiFolders, NativeFolders};
use imgui::{Key, Ui};
use std::rc::Rc;

const TITLE: &str = "New Project";
const SIZE: [f32; 2] = [620., 300.];
const BROWSER_TITLE: &str = "Browse for Folder";
const BROWSER_SIZE: [f32; 2] = [460., 420.];
const FOOTER_HEIGHT: f32 = 64.;
const DEFAULT_NAME: &str = "New Project";
const PROJECT_EXTENSION: &str = "opj";

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum WizardPage {
    #[default]
    ProjectType,
    ProjectFile,
    Libraries,
}
pub(super) struct NewProjectWizard {
    folders: Rc<dyn GuiFolders>,
    opening: bool,
    open: bool,
    page: WizardPage,
    name: String,
    workspace: String,
    browser_opening: bool,
    /// The folder the browser has chosen, before OK hands it to the wizard.
    browsed: String,
}
impl Default for NewProjectWizard {
    fn default() -> Self {
        Self {
            folders: Rc::new(NativeFolders),
            opening: false,
            open: false,
            page: WizardPage::default(),
            name: String::new(),
            workspace: String::new(),
            browser_opening: false,
            browsed: String::new(),
        }
    }
}
impl NewProjectWizard {
    /// Start from the first page, with a name and folder to change; example: on New.
    pub(super) fn request(&mut self) {
        self.opening = true;
        self.open = true;
        self.page = WizardPage::ProjectType;
        self.name = DEFAULT_NAME.to_owned();
        self.workspace = self.folders.default_folder();
    }
    pub(super) fn needs_frame(&self) -> bool {
        self.opening || self.browser_opening
    }
    /// The entry the project list shows while the wizard is open.
    pub(super) fn provisional_name(&self) -> Option<&str> {
        self.open.then_some(self.name.as_str())
    }
    fn project_file(&self) -> String {
        let file = format!("{}.{PROJECT_EXTENSION}", self.name);
        std::path::Path::new(&self.workspace)
            .join(file)
            .display()
            .to_string()
    }
    /// Draw above the Projects dialog; example: `wizard.panel(ui)`.
    pub(super) fn panel(&mut self, ui: &Ui) {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(TITLE);
        }
        center_dialog(ui, SIZE);
        let Some(_wizard) = ui.begin_modal_popup(TITLE) else {
            return;
        };
        match self.page {
            WizardPage::ProjectType => {
                ui.selectable_config("New Single User Project")
                    .selected(true)
                    .flags(imgui::SelectableFlags::DONT_CLOSE_POPUPS)
                    .build();
            }
            WizardPage::ProjectFile => self.project_file_page(ui),
            WizardPage::Libraries => libraries_page(ui),
        }
        ui.separator();
        self.buttons(ui);
        self.folder_browser(ui);
    }
    fn project_file_page(&mut self, ui: &Ui) {
        ui.input_text("Name", &mut self.name).build();
        ui.text("Project (Workspace) Folder");
        // Before the path, which can run past the dialog's edge and wrap.
        if crate::widget_abi::button(ui, "...##browse-workspace", [0.; 2]) {
            self.browsed.clone_from(&self.workspace);
            self.browser_opening = true;
        }
        ui.same_line();
        ui.text_wrapped(&self.workspace);
        ui.text("Project File to be Created");
        ui.text_wrapped(self.project_file());
    }
    fn buttons(&mut self, ui: &Ui) {
        let back = match self.page {
            WizardPage::ProjectType => None,
            WizardPage::ProjectFile => Some(WizardPage::ProjectType),
            WizardPage::Libraries => Some(WizardPage::ProjectFile),
        };
        let next = match self.page {
            WizardPage::ProjectType => Some(WizardPage::ProjectFile),
            WizardPage::ProjectFile => Some(WizardPage::Libraries),
            WizardPage::Libraries => None,
        };
        self.page_button(ui, "Back", back);
        ui.same_line();
        self.page_button(ui, "Next", next);
        ui.same_line();
        {
            let _muted = ui.begin_disabled(true);
            crate::widget_abi::button(ui, "Finish", [0.; 2]);
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel##wizard", [0.; 2])
            || ui.is_key_pressed_no_repeat(Key::Escape)
        {
            self.open = false;
            ui.close_current_popup();
        }
    }
    fn page_button(&mut self, ui: &Ui, label: &str, target: Option<WizardPage>) {
        let _muted = ui.begin_disabled(target.is_none());
        if crate::widget_abi::button(ui, label, [0.; 2])
            && let Some(page) = target
        {
            self.page = page;
        }
    }
    fn folder_browser(&mut self, ui: &Ui) {
        if std::mem::take(&mut self.browser_opening) {
            ui.open_popup(BROWSER_TITLE);
        }
        center_dialog(ui, BROWSER_SIZE);
        let Some(_browser) = ui.begin_modal_popup(BROWSER_TITLE) else {
            return;
        };
        // The tree scrolls by itself, so a long folder path cannot push the buttons away.
        ui.child_window("##folder-tree")
            .size([0., -FOOTER_HEIGHT])
            .build(|| {
                for (label, folder) in self.folders.system_locations() {
                    self.folder_node(ui, &label, &folder);
                }
            });
        ui.separator();
        ui.text(format!("Folder: {}", folder_label(&self.browsed)));
        {
            let _muted = ui.begin_disabled(true);
            crate::widget_abi::button(ui, "Make New Folder", [0.; 2]);
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "OK##folder", [0.; 2]) {
            self.workspace.clone_from(&self.browsed);
            ui.close_current_popup();
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel##folder", [0.; 2]) {
            ui.close_current_popup();
        }
    }
    /// A branch shows its own folders only once a click has reached it.
    fn folder_node(&mut self, ui: &Ui, label: &str, folder: &str) {
        let selected = self.browsed == folder;
        let mut flags = imgui::TreeNodeFlags::SPAN_AVAIL_WIDTH;
        if selected {
            flags |= imgui::TreeNodeFlags::SELECTED;
        }
        let chosen_path = self.browsed.starts_with(folder);
        let token = ui
            .tree_node_config(label)
            .flags(flags)
            .opened(chosen_path, imgui::Condition::Always)
            .push();
        if ui.is_item_clicked() {
            self.browsed = folder.to_owned();
        }
        if token.is_none() {
            return;
        }
        let children = self.folders.entries(folder).unwrap_or_default();
        for child in children.into_iter().filter(|entry| entry.is_folder) {
            let path = std::path::Path::new(folder).join(&child.name);
            self.folder_node(ui, &child.name, &path.display().to_string());
        }
    }
}
fn libraries_page(ui: &Ui) {
    ui.text("Select Libraries");
    ui.group(|| {
        ui.text("All Projects");
        ui.text_disabled("(none)");
    });
    ui.same_line();
    ui.group(|| {
        let _muted = ui.begin_disabled(true);
        crate::widget_abi::button(ui, ">>", [0.; 2]);
        crate::widget_abi::button(ui, "<<", [0.; 2]);
    });
    ui.same_line();
    ui.group(|| {
        ui.text("New Project");
        ui.text_disabled("(none)");
    });
}
/// A folder reads as its last name, as a row in the tree does.
fn folder_label(folder: &str) -> &str {
    std::path::Path::new(folder)
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or(folder)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn the_project_file_is_named_for_the_project_inside_its_folder() {
        let mut wizard = NewProjectWizard::default();
        wizard.request();
        wizard.name = "Gearbox".into();
        wizard.workspace = "/work/Documents".into();
        assert_eq!(wizard.project_file(), "/work/Documents/Gearbox.opj");
        assert_eq!(folder_label(&wizard.workspace), "Documents");
    }
    #[test]
    fn a_new_wizard_starts_on_the_type_page_with_a_provisional_entry() {
        let mut wizard = NewProjectWizard::default();
        assert_eq!(wizard.provisional_name(), None);
        wizard.request();
        assert_eq!(wizard.page, WizardPage::ProjectType);
        assert_eq!(wizard.provisional_name(), Some(DEFAULT_NAME));
    }
}
