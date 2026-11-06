// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiCommandState;
use crate::application::file_system::{FileFacts, GuiFolders, NativeFolders, timestamp_text};
use imgui::Ui;
use oblikovati_api::wire::DocumentInfo;
use std::{path::Path, rc::Rc, time::SystemTime};

const POPUP: &str = "###document-properties";
const NOT_AVAILABLE: &str = "Not available";
const NOT_SAVED: &str = "Not saved";

struct PropertiesTarget {
    id: u64,
    /// A file that is not open, from the Home list; an open document names its file itself.
    file: Option<String>,
    /// Read once when the dialog opens: a frame loop must not touch the disk.
    facts: Result<FileFacts, String>,
}
pub(crate) struct DocumentProperties {
    folders: Rc<dyn GuiFolders>,
    target: Option<PropertiesTarget>,
    opening: bool,
}
impl Default for DocumentProperties {
    fn default() -> Self {
        Self::with_folders(Rc::new(NativeFolders))
    }
}
impl DocumentProperties {
    pub(crate) fn with_folders(folders: Rc<dyn GuiFolders>) -> Self {
        Self {
            folders,
            target: None,
            opening: false,
        }
    }
    /// Inspect a document without activating it; example: `properties.request(&state, 7)`.
    pub(crate) fn request(&mut self, state: &GuiCommandState, id: u64) {
        let path = state
            .document_storage
            .get(&id)
            .and_then(|row| row.path.as_deref());
        let facts = path.map_or(Ok(FileFacts::default()), |path| {
            self.folders.file_facts(path)
        });
        self.target = Some(PropertiesTarget {
            id,
            file: None,
            facts,
        });
        self.opening = true;
    }
    /// Inspect a file that need not be open; example: `properties.request_file("/work/a.opd")`.
    pub(crate) fn request_file(&mut self, path: &str) {
        self.target = Some(PropertiesTarget {
            id: 0,
            file: Some(path.to_owned()),
            facts: self.folders.file_facts(path),
        });
        self.opening = true;
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening
    }
    /// Draw the read-only dialog; example: `properties.panel(ui, &state)`.
    pub(crate) fn panel(&mut self, ui: &Ui, state: &GuiCommandState) {
        let Some(target) = &self.target else {
            return;
        };
        let row = state.documents.iter().find(|row| row.id == target.id);
        let stem = target
            .file
            .as_deref()
            .and_then(|file| Path::new(file).file_stem());
        let file_name = stem.map(|stem| stem.to_string_lossy().into_owned());
        let name = file_name
            .as_deref()
            .or(row.map(|row| row.name.as_str()))
            .unwrap_or("Document");
        if std::mem::take(&mut self.opening) {
            ui.open_popup(POPUP);
        }
        let title = format!("{name} Properties{POPUP}");
        let Some(_dialog) = ui.begin_modal_popup(&title) else {
            return;
        };
        if let Some(_pages) = ui.tab_bar("##property-pages")
            && let Some(_general) = ui.tab_item("General")
        {
            general_page(ui, state, target, row);
        }
        let closed = crate::widget_abi::button(ui, "Close", [80., 0.])
            || ui.is_key_pressed(imgui::Key::Escape);
        if closed {
            ui.close_current_popup();
        }
    }
}
fn general_page(
    ui: &Ui,
    state: &GuiCommandState,
    target: &PropertiesTarget,
    row: Option<&DocumentInfo>,
) {
    let stored = state
        .document_storage
        .get(&target.id)
        .and_then(|row| row.path.as_deref());
    let path = target.file.as_deref().or(stored);
    let fields = general_fields(row, path, &target.facts);
    for (label, value) in &fields {
        ui.text(format!("{label}: {value}"));
    }
}
/// The General page's rows; an unsaved document names what it does not have yet.
fn general_fields(
    row: Option<&DocumentInfo>,
    path: Option<&str>,
    facts: &Result<FileFacts, String>,
) -> Vec<(&'static str, String)> {
    let path = path.map(Path::new);
    let stem = path
        .and_then(Path::file_stem)
        .map(|stem| stem.to_string_lossy().into_owned());
    let name = row.map_or_else(|| stem.unwrap_or_default(), |row| row.name.clone());
    // A file that is not open has its kind only in its extension.
    let extension = path
        .and_then(Path::extension)
        .map(|extension| extension.to_string_lossy().into_owned());
    let kind = row.map_or_else(|| extension.unwrap_or_default(), |row| row.type_.clone());
    let file_name = path
        .and_then(Path::file_name)
        .map(|file| file.to_string_lossy().into_owned());
    let location = path
        .and_then(Path::parent)
        .map(|folder| folder.display().to_string());
    vec![
        ("Name", name),
        (
            "File name",
            file_name.unwrap_or_else(|| NOT_SAVED.to_string()),
        ),
        ("Type", kind),
        (
            "Location",
            location.unwrap_or_else(|| NOT_SAVED.to_string()),
        ),
        ("Size", size_text(path.is_some(), facts)),
        ("Size on disk", disk_size_text(facts)),
        ("Attributes", attributes_text(path.is_some(), facts)),
        ("Created", timestamp_field(facts, |facts| facts.created)),
        ("Modified", timestamp_field(facts, |facts| facts.modified)),
        ("Accessed", timestamp_field(facts, |facts| facts.accessed)),
    ]
}
fn size_text(saved: bool, facts: &Result<FileFacts, String>) -> String {
    match facts {
        Ok(facts) if saved => format!("{} bytes", facts.size),
        Ok(_) => NOT_AVAILABLE.to_string(),
        Err(cause) => cause.clone(),
    }
}
fn disk_size_text(facts: &Result<FileFacts, String>) -> String {
    facts
        .as_ref()
        .ok()
        .and_then(|facts| facts.size_on_disk)
        .map_or_else(|| NOT_AVAILABLE.to_string(), |size| format!("{size} bytes"))
}
fn attributes_text(saved: bool, facts: &Result<FileFacts, String>) -> String {
    match facts {
        Ok(facts) if saved && facts.read_only => "Read-only".into(),
        Ok(_) if saved => "Normal".into(),
        _ => NOT_AVAILABLE.to_string(),
    }
}
fn timestamp_field(
    facts: &Result<FileFacts, String>,
    read: impl Fn(&FileFacts) -> Option<SystemTime>,
) -> String {
    let stamp = facts.as_ref().ok().and_then(read);
    stamp.map_or_else(|| NOT_AVAILABLE.to_string(), timestamp_text)
}
#[cfg(test)]
#[path = "../../../tests/application/document-tabs/properties.rs"]
mod tests;
