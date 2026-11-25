// SPDX-License-Identifier: GPL-2.0-only
//! The new-project wizard: a type, a name and folder, then libraries. A folder browser hangs off
//! its second page. Finish submits the configured workspace to the desktop owner.
use crate::application::document_saving::folder_browser::center_dialog;
use crate::application::file_system::{GuiFolders, NativeFolders};
use crate::{GuiCommandState, GuiProjectAction, GuiShellAction};
use imgui::{Key, Ui};
use std::path::Path;

const TITLE: &str = "New Project";
const SIZE: [f32; 2] = [620., 300.];
const BROWSER_TITLE: &str = "Browse for Folder";
const BROWSER_SIZE: [f32; 2] = [460., 420.];
const FOOTER_HEIGHT: f32 = 64.;
const DEFAULT_NAME: &str = "New Project";

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum WizardPage {
    #[default]
    ProjectType,
    ProjectFile,
    Libraries,
}
pub(super) struct NewProjectWizard {
    folders: NativeFolders,
    opening: bool,
    open: bool,
    page: WizardPage,
    name: String,
    workspace: String,
    browser_opening: bool,
    /// The folder the browser has chosen, before OK hands it to the wizard.
    browsed: String,
    new_folder: Option<NewFolder>,
    creation_request: Option<u64>,
    creation_error: Option<String>,
}
#[derive(Default)]
struct NewFolder {
    name: String,
    error: Option<String>,
    focus: bool,
}
impl Default for NewProjectWizard {
    fn default() -> Self {
        Self {
            folders: NativeFolders,
            opening: false,
            open: false,
            page: WizardPage::default(),
            name: String::new(),
            workspace: String::new(),
            browser_opening: false,
            browsed: String::new(),
            new_folder: None,
            creation_request: None,
            creation_error: None,
        }
    }
}
impl NewProjectWizard {
    /// Discard a pending wizard; example: `wizard.cancel()` when its parent closes.
    pub(super) fn cancel(&mut self) {
        self.open = false;
        self.opening = false;
        self.browser_opening = false;
        self.new_folder = None;
        self.creation_request = None;
        self.creation_error = None;
    }
    /// Start from the first page, with a name and folder to change; example: on New.
    pub(super) fn request(&mut self) {
        self.opening = true;
        self.open = true;
        self.page = WizardPage::ProjectType;
        self.name = DEFAULT_NAME.to_owned();
        self.workspace = self.folders.default_folder();
    }
    pub(super) fn needs_frame(&self) -> bool {
        self.opening
            || self.browser_opening
            || self.new_folder.as_ref().is_some_and(|draft| draft.focus)
    }
    /// The entry the project list shows while the wizard is open.
    pub(super) fn provisional_name(&self) -> Option<&str> {
        self.open.then_some(self.name.as_str())
    }
    fn project_file(&self) -> String {
        let file = format!(
            "{}{}",
            self.name.trim(),
            oblikovati_api::types::PROJECT_FILE_EXTENSION
        );
        std::path::Path::new(&self.workspace)
            .join(file)
            .display()
            .to_string()
    }
    /// Draw above Projects, for example `wizard.panel(ui, state, &mut actions)`.
    pub(super) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) -> Option<String> {
        if std::mem::take(&mut self.opening) {
            ui.open_popup(TITLE);
        }
        center_dialog(ui, SIZE);
        let _wizard = ui.begin_modal_popup(TITLE)?;
        if let Some(location) = self.complete_creation(state) {
            ui.close_current_popup();
            return Some(location);
        }
        self.wizard_page(ui);
        self.buttons(ui, actions);
        self.folder_browser(ui);
        None
    }
    fn complete_creation(&mut self, state: &GuiCommandState) -> Option<String> {
        let (request, result) = state.project_file_result.as_ref()?;
        if self.creation_request != Some(*request) {
            return None;
        }
        self.creation_request = None;
        match result {
            Ok(location) => {
                self.cancel();
                Some(location.clone())
            }
            Err(error) => {
                self.creation_error = Some(error.clone());
                None
            }
        }
    }
    fn wizard_page(&mut self, ui: &Ui) {
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
        if let Some(error) = &self.creation_error {
            ui.text_wrapped(error);
        }
        ui.separator();
    }
    fn project_file_page(&mut self, ui: &Ui) {
        if ui.input_text("Name", &mut self.name).build() {
            self.creation_error = None;
        }
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
    fn buttons(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
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
        self.finish_button(ui, actions);
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel##wizard", [0.; 2])
            || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape)
        {
            self.cancel();
            ui.close_current_popup();
        }
    }
    fn finish_button(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let enabled = self.page == WizardPage::Libraries
            && self.creation_request.is_none()
            && !self.name.trim().is_empty();
        let _muted = ui.begin_disabled(!enabled);
        if crate::widget_abi::button(ui, "Finish", [0.; 2]) {
            let request = super::next_project_request();
            self.creation_request = Some(request);
            self.creation_error = None;
            actions.push(GuiShellAction::ProjectFile(GuiProjectAction::Create {
                request,
                workspace: self.workspace.clone().into(),
                name: self.name.clone(),
            }));
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
        let footer = if self.new_folder.is_some() {
            FOOTER_HEIGHT * 2.
        } else {
            FOOTER_HEIGHT
        };
        // The tree scrolls by itself, so a long folder path cannot push the buttons away.
        ui.child_window("##folder-tree")
            .size([0., -footer])
            .build(|| {
                for (label, folder) in self.folders.system_locations() {
                    self.folder_node(ui, &label, &folder);
                }
            });
        ui.separator();
        ui.text(format!("Folder: {}", folder_label(&self.browsed)));
        if self.new_folder.is_some() {
            self.new_folder_controls(ui);
        } else {
            self.folder_buttons(ui);
        }
    }
    fn folder_buttons(&mut self, ui: &Ui) {
        if crate::widget_abi::button(ui, "Make New Folder", [0.; 2]) {
            self.new_folder = Some(NewFolder {
                focus: true,
                ..Default::default()
            });
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "OK##folder", [0.; 2]) {
            self.workspace.clone_from(&self.browsed);
            self.creation_error = None;
            ui.close_current_popup();
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel##folder", [0.; 2])
            || ui.is_key_pressed_no_repeat(Key::Escape)
        {
            ui.close_current_popup();
        }
    }
    fn new_folder_controls(&mut self, ui: &Ui) {
        let Some(draft) = &mut self.new_folder else {
            return;
        };
        if std::mem::take(&mut draft.focus) {
            ui.set_keyboard_focus_here();
        }
        if ui.input_text("Folder name", &mut draft.name).build() {
            draft.error = None;
        }
        if let Some(error) = &draft.error {
            ui.text_wrapped(error);
        }
        if crate::widget_abi::button(ui, "Create", [0.; 2]) {
            match self.folders.create_child(&self.browsed, &draft.name) {
                Ok(path) => {
                    self.browsed = path;
                    self.new_folder = None;
                }
                Err(error) => draft.error = Some(error),
            }
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel New Folder", [0.; 2])
            || ui.is_key_pressed_no_repeat(Key::Escape)
        {
            self.new_folder = None;
        }
    }
    /// A branch shows its own folders only once a click has reached it.
    fn folder_node(&mut self, ui: &Ui, label: &str, folder: &str) {
        let selected = self.browsed == folder;
        let mut flags = imgui::TreeNodeFlags::SPAN_AVAIL_WIDTH;
        if selected {
            flags |= imgui::TreeNodeFlags::SELECTED;
        }
        let chosen_path = Path::new(&self.browsed).starts_with(folder);
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
        use std::path::Path;

        let mut wizard = NewProjectWizard::default();
        wizard.request();
        wizard.name = "Gearbox".into();
        let workspace = std::env::temp_dir().join("Documents");
        wizard.workspace = workspace
            .to_str()
            .expect("expected a UTF-8 workspace")
            .into();
        let file = wizard.project_file();
        let file = Path::new(&file);
        assert_eq!(file.parent(), Some(workspace.as_path()));
        assert_eq!(file.file_name(), Some("Gearbox.opj".as_ref()));
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

    #[test]
    fn finish_waits_for_its_own_result_and_allows_retry_after_failure() {
        let mut wizard = NewProjectWizard::default();
        wizard.request();
        wizard.creation_request = Some(9);
        let mut state = GuiCommandState {
            project_file_result: Some((8, Ok("/other.opj".into()))),
            ..Default::default()
        };
        assert_eq!(wizard.complete_creation(&state), None);
        assert_eq!(wizard.creation_request, Some(9));
        state.project_file_result = Some((9, Err("already exists".into())));
        assert_eq!(wizard.complete_creation(&state), None);
        assert!(wizard.open);
        assert_eq!(wizard.creation_request, None);
        assert_eq!(wizard.creation_error.as_deref(), Some("already exists"));
        wizard.creation_request = Some(10);
        state.project_file_result = Some((10, Ok("/new.opj".into())));
        assert_eq!(
            wizard.complete_creation(&state).as_deref(),
            Some("/new.opj")
        );
        assert!(!wizard.open);
    }

    #[test]
    fn a_new_folder_uses_one_name_and_keeps_existing_entries() {
        let scratch = tempfile::tempdir().unwrap();
        let parent = scratch.path().to_str().unwrap();
        assert!(NativeFolders.create_child(parent, "../escape").is_err());
        assert!(
            NativeFolders
                .create_child(parent, "nested\\escape")
                .is_err()
        );
        assert!(NativeFolders.create_child(parent, "  ").is_err());
        let target = scratch.path().join("Workspace");
        assert_eq!(
            Path::new(&NativeFolders.create_child(parent, " Workspace ").unwrap()),
            target
        );
        std::fs::write(target.join("keep"), b"existing project files").unwrap();
        assert!(NativeFolders.create_child(parent, "Workspace").is_err());
        assert_eq!(
            std::fs::read(target.join("keep")).unwrap(),
            b"existing project files"
        );
        assert_eq!(std::fs::read_dir(scratch.path()).unwrap().count(), 1);
    }
}
