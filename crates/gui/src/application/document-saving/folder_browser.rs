// SPDX-License-Identifier: GPL-2.0-only
//! The location, listing and bottom row that the Save As and Open dialogs share.
use crate::application::file_system::{FolderEntry, GuiFolders};
use imgui::{Key, Ui};
use std::{path::Path, rc::Rc};

pub(crate) const FILE_TYPES: [&str; 2] = ["Part Files (*.opd)", "All Files (*.*)"];
pub(crate) const ALL_FILES: usize = 1;
const NATIVE_EXTENSION: &str = "opd";
const LISTING_HEIGHT: f32 = 230.;

pub(crate) struct FolderBrowser {
    folders: Rc<dyn GuiFolders>,
    pub folder: String,
    listing: Result<Vec<FolderEntry>, String>,
}
impl FolderBrowser {
    pub(crate) fn new(folders: Rc<dyn GuiFolders>) -> Self {
        Self {
            folder: folders.default_folder(),
            folders,
            listing: Ok(Vec::new()),
        }
    }
    /// Read the current folder again; example: `browser.refresh()` when a dialog opens.
    pub(crate) fn refresh(&mut self) {
        self.listing = self.folders.entries(&self.folder);
    }
    /// Draw the location and listing; returns the file row a click chose.
    /// Example: `browser.controls(ui, "Save in", file_type)`.
    pub(crate) fn controls(&mut self, ui: &Ui, label: &str, file_type: usize) -> Option<String> {
        self.location(ui, label);
        let (chosen, opened) = self.listing_rows(ui, file_type);
        if let Some(folder) = opened {
            self.navigate(&folder);
        }
        chosen
    }
    fn location(&mut self, ui: &Ui, label: &str) {
        ui.text(label);
        ui.same_line();
        ui.set_next_item_width(-48.);
        let entered = ui
            .input_text(format!("##folder-{label}"), &mut self.folder)
            .auto_select_all(true)
            .enter_returns_true(true)
            .build();
        ui.same_line();
        if crate::widget_abi::button(ui, "Up", [0.; 2]) {
            self.up();
        } else if entered {
            self.refresh();
        }
    }
    fn up(&mut self) {
        let parent = Path::new(&self.folder).parent().map(Path::to_path_buf);
        if let Some(parent) = parent {
            self.folder = parent.display().to_string();
            self.refresh();
        }
    }
    fn navigate(&mut self, name: &str) {
        self.folder = Path::new(&self.folder).join(name).display().to_string();
        self.refresh();
    }
    fn listing_rows(&self, ui: &Ui, file_type: usize) -> (Option<String>, Option<String>) {
        let (mut chosen, mut opened) = (None, None);
        ui.child_window("##folder-listing")
            .size([0., LISTING_HEIGHT])
            .border(true)
            .build(|| match &self.listing {
                Err(cause) => ui.text_disabled(cause),
                Ok(rows) => {
                    for row in rows.iter().filter(|row| listed(row, file_type)) {
                        if crate::widget_abi::dialog_selectable(ui, &row_label(row), false) {
                            let target = if row.is_folder {
                                &mut opened
                            } else {
                                &mut chosen
                            };
                            *target = Some(row.name.clone());
                        }
                    }
                }
            });
        (chosen, opened)
    }
}
fn listed(row: &FolderEntry, file_type: usize) -> bool {
    file_type == ALL_FILES || row.is_folder || is_native_file(&row.name)
}
fn is_native_file(name: &str) -> bool {
    Path::new(name)
        .extension()
        .is_some_and(|extension| extension == NATIVE_EXTENSION)
}
fn row_label(row: &FolderEntry) -> String {
    if row.is_folder {
        format!("{}/", row.name)
    } else {
        row.name.clone()
    }
}
/// The document's own name as a file name; example: `proposed_file_name("Part")` is `Part.opd`.
pub(crate) fn proposed_file_name(name: &str) -> String {
    if is_native_file(name) {
        return name.to_string();
    }
    format!("{name}.{NATIVE_EXTENSION}")
}
/// A name typed as an absolute path wins over the folder; example: `full_path("/a", "b.opd")`.
pub(crate) fn full_path(folder: &str, name: &str) -> String {
    Path::new(folder).join(name).display().to_string()
}
/// Draw the file name field; `true` when Enter accepted it. The whole name starts selected.
pub(crate) fn file_name_field(ui: &Ui, name: &mut String) -> bool {
    ui.text("File name");
    ui.same_line();
    if ui.is_window_appearing() {
        ui.set_keyboard_focus_here();
    }
    ui.set_next_item_width(-1.);
    ui.input_text("##file-name", name)
        .auto_select_all(true)
        .enter_returns_true(true)
        .build()
}
/// Draw the file type list; example: `file_type_field(ui, &mut draft.file_type)`.
pub(crate) fn file_type_field(ui: &Ui, file_type: &mut usize) {
    ui.text("Files of type");
    ui.same_line();
    ui.set_next_item_width(-1.);
    ui.combo_simple_string("##file-type", file_type, &FILE_TYPES);
}
/// Escape cancels a dialog only when no list or nested dialog inside it took that press.
#[derive(Default)]
pub(crate) struct DialogEscape {
    nested_open: bool,
}
impl DialogEscape {
    /// Call once per frame after the dialog's nested popups; example: `escape.pressed(ui)`.
    pub(crate) fn pressed(&mut self, ui: &Ui) -> bool {
        // Dear ImGui closes the nested popup before this frame's widgets, so read last frame's state.
        let pressed = ui.is_key_pressed(Key::Escape) && !self.nested_open;
        self.nested_open = crate::widget_abi::nested_popup_open(ui);
        pressed
    }
}
/// How a user left a file dialog this frame.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum DialogEnd {
    Accepted,
    Cancelled,
}
/// Draw the accept and Cancel buttons; example: `dialog_buttons(ui, "Save", true, entered, &mut escape)`.
pub(crate) fn dialog_buttons(
    ui: &Ui,
    accept: &str,
    can_accept: bool,
    entered: bool,
    escape: &mut DialogEscape,
) -> Option<DialogEnd> {
    let disabled = ui.begin_disabled(!can_accept);
    let accepted = (crate::widget_abi::button(ui, accept, [80., 0.]) || entered) && can_accept;
    drop(disabled);
    ui.same_line();
    let cancelled = crate::widget_abi::button(ui, "Cancel", [80., 0.]);
    if accepted {
        return Some(DialogEnd::Accepted);
    }
    (escape.pressed(ui) || cancelled).then_some(DialogEnd::Cancelled)
}
#[cfg(test)]
#[path = "../../../tests/application/document-saving/folder_browser.rs"]
mod tests;
