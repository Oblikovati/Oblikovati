// SPDX-License-Identifier: GPL-2.0-only
//! A closed file's properties: the Details pane, and the separate window its link opens.
use super::file_choices::FileKind;
use crate::application::document_saving::folder_browser::FolderBrowser;
use crate::application::document_saving::save_as::draw_preview_image;
use crate::application::file_system::{FileFacts, GuiFolders, timestamp_text};
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use std::{collections::BTreeMap, path::Path, rc::Rc};

const TITLE: &str = "File Properties";
const SIZE: [f32; 2] = [560., 420.];
const SUMMARY_FIELDS: [&str; 5] = ["Title", "Subject", "Author", "Category", "Comments"];
const PROJECT_FIELDS: [&str; 4] = ["Part Number", "Description", "Revision Number", "Designer"];
const NO_THUMBNAIL: &str = "No saved image";

/// What the owner read from one closed file.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiInspectedFile {
    pub path: String,
    pub design_views: Vec<String>,
    pub properties: Vec<(String, String)>,
    /// The picture stored in the file.
    pub picture: Option<crate::GuiPreviewImage>,
}
impl GuiInspectedFile {
    fn field(&self, name: &str) -> &str {
        self.properties
            .iter()
            .find(|(field, _)| field == name)
            .map_or("", |(_, value)| value.as_str())
    }
}
pub(super) struct FileProperties {
    folders: Rc<dyn GuiFolders>,
    path: Option<String>,
    dock_state: crate::dockable_lifecycle::DockableState,
    /// Read once when the window opens: a frame loop must not touch the disk.
    facts: Result<FileFacts, String>,
    opening: bool,
}
impl DockableWindow for FileProperties {}
impl FileProperties {
    pub(super) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.dock_state)
    }
    /// The file its window is showing, whose stored properties it must have read.
    pub(super) fn shown_path(&self) -> Option<String> {
        self.path.clone()
    }
    pub(super) fn new(folders: Rc<dyn GuiFolders>) -> Self {
        Self {
            folders,
            path: None,
            dock_state: Default::default(),
            facts: Ok(FileFacts::default()),
            opening: false,
        }
    }
    /// Read a file's facts for its panes; example: when a click selects it.
    pub(super) fn read_facts(&mut self, path: &str) {
        self.facts = self.folders.file_facts(path);
    }
    pub(super) fn request(&mut self, path: &str) {
        self.read_facts(path);
        self.path = Some(path.to_owned());
        self.dock_state.set_open(true);
        self.opening = true;
    }
    pub(super) fn is_opening(&self) -> bool {
        self.opening
    }
    /// Close with its parent; example: `properties.close()` after Open is dismissed.
    pub(super) fn close(&mut self) {
        self.path = None;
        self.dock_state.set_open(false);
        self.opening = false;
    }
    pub(super) fn panel(&mut self, ui: &Ui, inspections: &FileInspections) {
        self.opening = false;
        let Some(path) = self.path.clone() else {
            return;
        };
        Self::center_dockable(ui, TITLE, SIZE);
        let inspected = inspections.of(&path);
        let mut open = true;
        let mut closed = false;
        Self::dockable_window(ui, TITLE, &mut open).build(|| {
            if let Some(_pages) = ui.tab_bar("##file-property-pages") {
                if let Some(_general) = ui.tab_item("General") {
                    general_rows(ui, &path, &self.facts);
                }
                if let Some(_summary) = ui.tab_item("Summary") {
                    field_rows(ui, &SUMMARY_FIELDS, inspected);
                }
                if let Some(_project) = ui.tab_item("Project") {
                    field_rows(ui, &PROJECT_FIELDS, inspected);
                }
                if let Some(_thumbnail) = ui.tab_item("Thumbnail") {
                    match inspections.picture_of(&path) {
                        Some(image) => draw_preview_image(ui, image),
                        None => ui.text_disabled(NO_THUMBNAIL),
                    }
                }
            }
            ui.separator();
            closed = end_buttons(ui);
        });
        if closed || !open {
            self.close();
        }
    }
}
fn end_buttons(ui: &Ui) -> bool {
    let closed = crate::widget_abi::button(ui, "OK", [80., 0.])
        || crate::widget_abi::button(ui, "Cancel", [80., 0.]);
    ui.same_line();
    {
        // Nothing here can be changed yet, so there is nothing to apply.
        let _muted = ui.begin_disabled(true);
        crate::widget_abi::button(ui, "Apply", [80., 0.]);
    }
    closed || ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
}
fn general_rows(ui: &Ui, path: &str, facts: &Result<FileFacts, String>) {
    ui.text(format!("Type: {}", file_type(path)));
    ui.text(format!("Location: {}", parent_of(path)));
    let Ok(facts) = facts else {
        ui.text_disabled("Size: Not available");
        return;
    };
    ui.text(format!("Size: {} B", facts.size));
    for (label, time) in [
        ("Created", facts.created),
        ("Modified", facts.modified),
        ("Accessed", facts.accessed),
    ] {
        let shown = time.map_or_else(String::new, timestamp_text);
        ui.text(format!("{label}: {shown}"));
    }
}
fn field_rows(ui: &Ui, fields: &[&str], inspected: Option<&GuiInspectedFile>) {
    for field in fields {
        let value = inspected.map_or("", |file| file.field(field));
        ui.text(format!("{field}: {value}"));
    }
}
fn file_type(path: &str) -> &'static str {
    match Path::new(path).extension().and_then(|kind| kind.to_str()) {
        Some("opd") => "Part",
        Some("oad") => "Assembly",
        Some("odd") => "Drawing",
        Some("ord") => "Presentation",
        _ => "File",
    }
}
fn parent_of(path: &str) -> String {
    Path::new(path)
        .parent()
        .map_or_else(String::new, |folder| folder.display().to_string())
}
/// What the dialog has read from files, one file per frame so a frame stays short.
#[derive(Default)]
pub(super) struct FileInspections {
    files: BTreeMap<String, GuiInspectedFile>,
    asked: Option<String>,
}
impl FileInspections {
    /// Take the answer the owner has for what was asked; an older answer may be out of date.
    pub(super) fn follow(&mut self, state: &GuiCommandState, browser: &mut FolderBrowser) {
        let Some(file) = state.inspected_file.as_ref() else {
            return;
        };
        if self.asked.as_deref() == Some(file.path.as_str()) {
            self.asked = None;
            browser.note_file_picture(&file.path, file.picture.as_ref());
            self.files.insert(file.path.clone(), file.clone());
        }
    }
    /// Ask for the first file not read yet, unless an answer is still due.
    pub(super) fn ask_next(
        &mut self,
        wanted: impl Iterator<Item = String>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.asked.is_some() {
            return;
        }
        let mut unread = wanted.filter(|path| !self.files.contains_key(path));
        let next = unread.find(|path| FileKind::of(path) != FileKind::Other);
        if let Some(path) = next {
            actions.push(GuiShellAction::InspectFile(path.clone()));
            self.asked = Some(path);
        }
    }
    pub(super) fn of(&self, path: &str) -> Option<&GuiInspectedFile> {
        self.files.get(path)
    }
    pub(super) fn picture_of(&self, path: &str) -> Option<&crate::GuiPreviewImage> {
        self.of(path).and_then(|file| file.picture.as_ref())
    }
}
/// The Details pane: what a selection can tell without the separate window.
pub(super) fn details_rows(
    ui: &Ui,
    properties: &FileProperties,
    path: &str,
    inspected: Option<&GuiInspectedFile>,
) {
    ui.text(format!("Type: {}", file_type(path)));
    if let Ok(facts) = &properties.facts {
        let modified = facts.modified.map_or_else(String::new, timestamp_text);
        ui.text(format!("Modified: {modified}"));
        ui.text(format!("Size: {} B", facts.size));
    }
    let field = |name: &str| inspected.map_or("", |file| file.field(name)).to_owned();
    ui.text(format!("Author: {}", field("Author")));
    ui.text(format!("Part Number: {}", field("Part Number")));
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn parent_close_notifies_file_properties_once() {
        let mut registry = crate::dockable_lifecycle::DockRegistry::default();
        let mut dialog =
            FileProperties::new(Rc::new(crate::application::file_system::NativeFolders));
        dialog.register_dockable(&mut registry).unwrap();
        dialog.path = Some("part.opd".into());
        dialog.dock_state.set_open(true);
        registry.reconcile(|_| {}).unwrap();
        dialog.close();
        let mut titles = Vec::new();
        assert!(
            registry
                .reconcile(|title| titles.push(title.to_string_lossy().into_owned()))
                .unwrap()
        );
        assert_eq!(titles, ["File Properties"]);
        assert!(!registry.reconcile(|_| {}).unwrap());
    }
    #[test]
    fn a_missing_field_reads_as_empty() {
        let file = GuiInspectedFile {
            path: "/a.opd".into(),
            design_views: Vec::new(),
            properties: vec![("Author".into(), "Ada".into())],
            picture: None,
        };
        assert_eq!(file.field("Author"), "Ada");
        assert_eq!(file.field("Title"), "");
        assert_eq!(file_type("/a.odd"), "Drawing");
        assert_eq!(parent_of("/work/a.odd"), "/work");
    }
}
