// SPDX-License-Identifier: GPL-2.0-only
//! The location, listing and bottom row that the Save As and Open dialogs share.
use crate::application::file_system::{FolderEntry, GuiFolders};
use imgui::{Condition, Key, Ui};
use oblikovati_api::types::{
    DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PRESENTATION, DocumentType,
};
use std::{path::Path, rc::Rc};

const LISTING_HEIGHT: f32 = 200.;
const LOCATION_FIELD_WIDTH: f32 = 230.;
const BREADCRUMB_STEPS: usize = 3;
/// Space between a field and its label, wide enough to read as two things.
const LABEL_GAP: f32 = 16.;
const ALL_FILES: &str = "All Files (*.*)";

/// One native file kind: the list's words and the extension without its dot.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) struct NativeFileType {
    label: &'static str,
    pub extension: &'static str,
}
const PART_FILES: NativeFileType = NativeFileType {
    label: "Part Files (*.opd)",
    extension: "opd",
};
const ASSEMBLY_FILES: NativeFileType = NativeFileType {
    label: "Assembly Files (*.oad)",
    extension: "oad",
};
const DRAWING_FILES: NativeFileType = NativeFileType {
    label: "Drawing Files (*.odd)",
    extension: "odd",
};
const PRESENTATION_FILES: NativeFileType = NativeFileType {
    label: "Presentation Files (*.ord)",
    extension: "ord",
};
const NATIVE_FILE_TYPES: [NativeFileType; 4] = [
    PART_FILES,
    ASSEMBLY_FILES,
    DRAWING_FILES,
    PRESENTATION_FILES,
];
/// The file type of a document; an unknown kind falls back to the part file a new document is.
pub(crate) fn native_file_type(kind: Option<DocumentType>) -> NativeFileType {
    match kind {
        Some(DOCUMENT_ASSEMBLY) => ASSEMBLY_FILES,
        Some(DOCUMENT_DRAWING) => DRAWING_FILES,
        Some(DOCUMENT_PRESENTATION) => PRESENTATION_FILES,
        _ => PART_FILES,
    }
}
/// The rows a dialog's file type list offers, each with the extension it keeps.
pub(crate) struct FileTypeList {
    rows: Vec<(&'static str, Option<&'static str>)>,
}
impl FileTypeList {
    /// One document's own type and All Files; example: `FileTypeList::saving(PART_FILES)`.
    pub(crate) fn saving(native: NativeFileType) -> Self {
        Self {
            rows: vec![(native.label, Some(native.extension)), (ALL_FILES, None)],
        }
    }
    /// Every native type and All Files; example: the Open dialog.
    pub(crate) fn opening() -> Self {
        let native = NATIVE_FILE_TYPES.map(|kind| (kind.label, Some(kind.extension)));
        let mut rows = native.to_vec();
        rows.push((ALL_FILES, None));
        Self { rows }
    }
    /// Only assemblies; example: inserting a model into a presentation.
    pub(crate) fn assemblies() -> Self {
        Self {
            rows: vec![(ASSEMBLY_FILES.label, Some(ASSEMBLY_FILES.extension))],
        }
    }
    /// All files, for choices that are not documents; example: a preview image.
    pub(crate) fn any() -> Self {
        Self {
            rows: vec![(ALL_FILES, None)],
        }
    }
    /// The extension a row keeps, or `None` for all files.
    pub(crate) fn filter(&self, row: usize) -> Option<&'static str> {
        self.rows.get(row).and_then(|(_, extension)| *extension)
    }
    fn labels(&self) -> Vec<&'static str> {
        self.rows.iter().map(|(label, _)| *label).collect()
    }
}

pub(crate) struct FolderBrowser {
    folders: Rc<dyn GuiFolders>,
    pub folder: String,
    listing: Result<Vec<FolderEntry>, String>,
    /// Folders left behind, most recent last; Back returns to them.
    history: Vec<String>,
    selected: Option<String>,
}
impl FolderBrowser {
    pub(crate) fn new(folders: Rc<dyn GuiFolders>) -> Self {
        Self {
            folder: folders.default_folder(),
            folders,
            listing: Ok(Vec::new()),
            history: Vec::new(),
            selected: None,
        }
    }
    /// Read the current folder again; example: `browser.refresh()` after a save.
    pub(crate) fn refresh(&mut self) {
        self.listing = self.folders.entries(&self.folder);
        self.selected = None;
    }
    /// Start over in the initial folder: a cancelled dialog keeps no location.
    pub(crate) fn reset(&mut self) {
        self.folder = self.folders.default_folder();
        self.history.clear();
        self.refresh();
    }
    /// Draw the location and listing; returns the file row a click chose.
    /// Example: `browser.controls(ui, "Save in", Some("opd"))`.
    pub(crate) fn controls(
        &mut self,
        ui: &Ui,
        label: &str,
        filter: Option<&str>,
    ) -> Option<String> {
        self.location(ui, label);
        ui.text_disabled(breadcrumb(&self.folder));
        let (chosen, opened) = self.listing_rows(ui, filter);
        ui.text_disabled(status_text(
            self.selected.is_some(),
            self.shown_count(filter),
        ));
        if let Some(folder) = opened {
            self.navigate(&folder);
        }
        if let Some(name) = &chosen {
            self.selected = Some(name.clone());
        }
        chosen
    }
    fn location(&mut self, ui: &Ui, label: &str) {
        ui.text(label);
        ui.same_line_with_spacing(0., LABEL_GAP);
        ui.set_next_item_width(LOCATION_FIELD_WIDTH);
        let entered = ui
            .input_text(format!("##folder-{label}"), &mut self.folder)
            .auto_select_all(true)
            .enter_returns_true(true)
            .build();
        ui.same_line();
        let up = crate::widget_abi::button(ui, "Up", [0.; 2]);
        ui.same_line();
        let back = crate::widget_abi::button(ui, "Back", [0.; 2]);
        if back {
            self.back();
        } else if up {
            self.up();
        } else if entered {
            self.refresh();
        }
    }
    fn up(&mut self) {
        let parent = Path::new(&self.folder).parent().map(Path::to_path_buf);
        if let Some(parent) = parent {
            self.history.push(self.folder.clone());
            self.folder = parent.display().to_string();
            self.refresh();
        }
    }
    fn back(&mut self) {
        if let Some(previous) = self.history.pop() {
            self.folder = previous;
            self.refresh();
        }
    }
    fn navigate(&mut self, name: &str) {
        self.history.push(self.folder.clone());
        self.folder = Path::new(&self.folder).join(name).display().to_string();
        self.refresh();
    }
    fn shown_count(&self, filter: Option<&str>) -> usize {
        let rows = self.listing.as_deref().unwrap_or_default();
        rows.iter().filter(|row| listed(row, filter)).count()
    }
    fn listing_rows(&self, ui: &Ui, filter: Option<&str>) -> (Option<String>, Option<String>) {
        let (mut chosen, mut opened) = (None, None);
        ui.child_window("##folder-listing")
            .size([0., LISTING_HEIGHT])
            .border(true)
            .build(|| match &self.listing {
                Err(cause) => ui.text_disabled(cause),
                Ok(rows) => {
                    for row in rows.iter().filter(|row| listed(row, filter)) {
                        let highlighted = self.selected.as_deref() == Some(row.name.as_str());
                        if crate::widget_abi::dialog_selectable(ui, &row_label(row), highlighted) {
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
/// The folder's last steps; a dialog column cannot show a long path, and the end is what matters.
/// Example: `... > work > parts > bolts`.
fn breadcrumb(folder: &str) -> String {
    let steps: Vec<String> = Path::new(folder)
        .iter()
        .map(|step| step.to_string_lossy().into_owned())
        .collect();
    let shown = steps.len().min(BREADCRUMB_STEPS);
    let tail = steps[steps.len() - shown..].join(" > ");
    if shown < steps.len() {
        return format!("... > {tail}");
    }
    tail
}
fn status_text(selected: bool, count: usize) -> String {
    if selected {
        return "1 item selected".to_string();
    }
    format!("{count} items")
}
fn listed(row: &FolderEntry, filter: Option<&str>) -> bool {
    row.is_folder || filter.is_none_or(|extension| has_extension(&row.name, extension))
}
fn has_extension(name: &str, extension: &str) -> bool {
    Path::new(name)
        .extension()
        .is_some_and(|found| found.eq_ignore_ascii_case(extension))
}
fn row_label(row: &FolderEntry) -> String {
    if row.is_folder {
        format!("{}/", row.name)
    } else {
        row.name.clone()
    }
}
/// The document's own name as a file name; example: `proposed_file_name("Part1", "opd")`.
pub(crate) fn proposed_file_name(name: &str, extension: &str) -> String {
    if has_extension(name, extension) {
        return name.to_string();
    }
    format!("{name}.{extension}")
}
/// Centre the next window when it first appears, inside the display.
pub(crate) fn center_dialog(ui: &Ui, size: [f32; 2]) {
    let display = ui.io().display_size;
    let fitted = [size[0].min(display[0] - 24.), size[1].min(display[1] - 24.)];
    let position = std::array::from_fn(|axis| (display[axis] - fitted[axis]) * 0.5);
    crate::widget_abi::window_geometry(ui, position, fitted, Condition::Appearing);
}
/// A name typed as an absolute path wins over the folder; example: `full_path("/a", "b.opd")`.
pub(crate) fn full_path(folder: &str, name: &str) -> String {
    Path::new(folder).join(name).display().to_string()
}
/// Draw the file name field; `true` when Enter accepted it. The whole name starts selected.
pub(crate) fn file_name_field(ui: &Ui, name: &mut String) -> bool {
    ui.text("File name");
    ui.same_line_with_spacing(0., LABEL_GAP);
    if ui.is_window_appearing() {
        ui.set_keyboard_focus_here();
    }
    ui.set_next_item_width(-1.);
    ui.input_text("##file-name", name)
        .auto_select_all(true)
        .enter_returns_true(true)
        .build()
}
/// Draw the file type list; example: `file_type_field(ui, &types, &mut draft.file_type)`.
pub(crate) fn file_type_field(ui: &Ui, types: &FileTypeList, file_type: &mut usize) {
    ui.text("Files of type");
    ui.same_line_with_spacing(0., LABEL_GAP);
    ui.set_next_item_width(-1.);
    ui.combo_simple_string("##file-type", file_type, &types.labels());
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
        let pressed = ui.is_key_pressed_no_repeat(Key::Escape) && !self.nested_open;
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
