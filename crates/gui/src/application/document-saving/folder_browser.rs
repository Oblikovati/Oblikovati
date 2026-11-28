// SPDX-License-Identifier: GPL-2.0-only
//! The location, listing and bottom row that the Save As and Open dialogs share.
use crate::application::file_system::{FolderEntry, GuiFolders};
use imgui::{Condition, Key, Ui};
use std::{path::Path, rc::Rc};
#[path = "file_rows.rs"]
pub(super) mod file_rows;

pub(super) const LISTING_HEIGHT: f32 = 180.;
const NAVIGATION_WIDTH: f32 = 150.;
const RECENT_FOLDERS: usize = 8;
const LOCATION_FIELD_WIDTH: f32 = 480.;
/// The width of the location and listing, when a dialog also shows a preview beside them.
pub(crate) const BROWSER_COLUMN_WIDTH: f32 = 720.;
const BREADCRUMB_STEPS: usize = 3;
/// Space between a field and its label, wide enough to read as two things.
const LABEL_GAP: f32 = 16.;
const DISPLAY_MENU: &str = "##file-display";

#[path = "file_types.rs"]
mod file_types;
use file_types::has_extension;
#[cfg(test)]
use file_types::{ALL_FILES, ASSEMBLY_FILES, DRAWING_FILES, PART_FILES, PRESENTATION_FILES};
pub(crate) use file_types::{FileFilter, FileTypeList, native_file_type};
pub(crate) struct FolderBrowser {
    folders: Rc<dyn GuiFolders>,
    pub folder: String,
    pub(super) listing: Result<Vec<FolderEntry>, String>,
    /// Folders left behind, most recent last; Back returns to them.
    history: Vec<String>,
    /// Folders Back left, most recent last; Forward returns to them.
    forward: Vec<String>,
    pub(super) selected: Option<String>,
    /// What a selection rectangle chose, when it crossed more than one file.
    pub(super) several: Vec<String>,
    /// A folder row chosen, or just left through Up; Open enters it.
    pub(super) folder_highlight: Option<String>,
    /// Rows are drawn before their click updates selection, so one more frame must show it.
    pub(crate) selection_redraw: bool,
    /// A click on a folder only chooses it, in a dialog that opens things with its button.
    pub folders_select_first: bool,
    pub(crate) show_file_properties: bool,
    pub(super) size_order_descending: Option<bool>,
    pub(super) request: super::file_list::FileRequest,
    pub(super) menu_row: Option<String>,
    pub(super) row_menu_bounds: Option<([f32; 2], [f32; 2])>,
    pub(super) rectangle_start: Option<[f32; 2]>,
    /// Folders this dialog has shown, most recent first, for the recent-folder pane.
    recent: Vec<String>,
    navigation: NavigationPane,
    pub(super) view: FileListView,
    /// Stored pictures by full path, as far as the dialog has read them.
    pub(super) file_pictures: std::collections::BTreeMap<String, crate::GuiPreviewImage>,
}
/// What the pane left of the file list offers.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum NavigationPane {
    #[default]
    RecentFolders,
    SystemLocations,
}
/// How the file list shows its entries.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum FileListView {
    ExtraLargeIcons,
    LargeIcons,
    MediumIcons,
    SmallIcons,
    List,
    #[default]
    Details,
    Tiles,
    Content,
}
impl FileListView {
    const MENU: [(Self, &'static str); 8] = [
        (Self::ExtraLargeIcons, "Extra Large Icons"),
        (Self::LargeIcons, "Large Icons"),
        (Self::MediumIcons, "Medium Icons"),
        (Self::SmallIcons, "Small Icons"),
        (Self::List, "List"),
        (Self::Details, "Details"),
        (Self::Tiles, "Tiles"),
        (Self::Content, "Content"),
    ];
    /// True when its rows draw each file's stored picture.
    pub(crate) fn shows_pictures(self) -> bool {
        !matches!(self, Self::List | Self::Details)
    }
}
impl FolderBrowser {
    pub(crate) fn new(folders: Rc<dyn GuiFolders>) -> Self {
        Self {
            folder: folders.default_folder(),
            folders,
            listing: Ok(Vec::new()),
            history: Vec::new(),
            forward: Vec::new(),
            selected: None,
            several: Vec::new(),
            folder_highlight: None,
            selection_redraw: false,
            folders_select_first: false,
            show_file_properties: true,
            size_order_descending: None,
            request: Default::default(),
            menu_row: None,
            row_menu_bounds: None,
            rectangle_start: None,
            recent: Vec::new(),
            navigation: NavigationPane::default(),
            view: FileListView::default(),
            file_pictures: Default::default(),
        }
    }
    /// Read the current folder again; example: `browser.refresh()` after a save.
    pub(crate) fn refresh(&mut self) {
        self.listing = self.folders.entries(&self.folder);
        self.selected = None;
        self.remember_folder();
    }
    fn remember_folder(&mut self) {
        let folder = self.folder.clone();
        self.recent.retain(|recent| *recent != folder);
        self.recent.insert(0, folder);
        self.recent.truncate(RECENT_FOLDERS);
    }
    /// Start over in the initial folder: a cancelled dialog keeps no location.
    pub(crate) fn reset(&mut self) {
        self.folder = self.folders.default_folder();
        self.history.clear();
        self.folder_highlight = None;
        self.navigation = NavigationPane::default();
        self.refresh();
    }
    /// Draw the location and listing; returns the file row a click chose.
    /// Example: `browser.controls(ui, "Save in", FileFilter::Extension("opd"))`.
    pub(crate) fn controls(&mut self, ui: &Ui, label: &str, filter: FileFilter) -> Option<String> {
        let previous = (
            self.selected.clone(),
            self.folder_highlight.clone(),
            self.several.clone(),
        );
        self.location(ui, label);
        ui.text_disabled(breadcrumb(&self.folder));
        self.toolbar(ui);
        let visited = self.navigation_pane(ui);
        ui.same_line();
        let outcome = self.listing_rows(ui, filter);
        let (chosen, opened) = (outcome.chosen, outcome.entered);
        let one_selected = self.selected.is_some() || self.folder_highlight.is_some();
        let selected = self.several.len().max(usize::from(one_selected));
        ui.text_disabled(status_text(selected, self.shown_count(filter)));
        if let Some(folder) = opened {
            self.navigate(&folder);
        }
        if let Some(folder) = visited {
            self.visit(folder);
        }
        if let Some(name) = &chosen {
            self.selected = Some(name.clone());
        }
        self.selection_redraw = previous
            != (
                self.selected.clone(),
                self.folder_highlight.clone(),
                self.several.clone(),
            );
        chosen
    }
    fn toolbar(&mut self, ui: &Ui) {
        let buttons = [
            ("Recent", Toolbar::Pane(NavigationPane::RecentFolders)),
            ("Locations", Toolbar::Pane(NavigationPane::SystemLocations)),
            ("Details", Toolbar::View(FileListView::Details)),
            ("Tiles", Toolbar::View(FileListView::Tiles)),
        ];
        for (label, choice) in buttons {
            if crate::widget_abi::button(ui, label, [0.; 2]) {
                match choice {
                    Toolbar::Pane(pane) => self.navigation = pane,
                    Toolbar::View(view) => self.view = view,
                }
            }
            ui.same_line();
        }
        self.display_menu(ui);
    }
    fn display_menu(&mut self, ui: &Ui) {
        if crate::widget_abi::button(ui, "Display", [0.; 2]) {
            ui.open_popup(DISPLAY_MENU);
        }
        let Some(_menu) = ui.begin_popup(DISPLAY_MENU) else {
            return;
        };
        for (view, label) in FileListView::MENU {
            if ui
                .menu_item_config(label)
                .selected(self.view == view)
                .build()
            {
                self.view = view;
            }
            if view == FileListView::SmallIcons {
                ui.separator();
            }
        }
    }
    fn navigation_rows(&self) -> Vec<(String, String)> {
        match self.navigation {
            NavigationPane::RecentFolders => self
                .recent
                .iter()
                .map(|folder| (folder_name(folder), folder.clone()))
                .collect(),
            NavigationPane::SystemLocations => self.folders.system_locations(),
        }
    }
    pub(super) fn listing_height(&self, ui: &Ui) -> f32 {
        (ui.content_region_avail()[1] - 122.).max(LISTING_HEIGHT)
    }
    /// Draw the pane; returns the folder a click chose.
    fn navigation_pane(&self, ui: &Ui) -> Option<String> {
        let mut visited = None;
        let height = self.listing_height(ui);
        ui.child_window("##folder-navigation")
            .size([NAVIGATION_WIDTH, height])
            .border(true)
            .build(|| {
                for heading in navigation_headings(self.navigation) {
                    ui.text_disabled(heading);
                }
                for (label, folder) in self.navigation_rows() {
                    let row = format!("{label}##{folder}");
                    if crate::widget_abi::dialog_selectable(ui, &row, folder == self.folder) {
                        visited = Some(folder);
                    }
                }
            });
        visited
    }
    fn visit(&mut self, folder: String) {
        if folder != self.folder {
            self.history.push(self.folder.clone());
            self.folder = folder;
            self.refresh();
        }
    }
    fn location(&mut self, ui: &Ui, label: &str) {
        ui.align_text_to_frame_padding();
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
        ui.same_line();
        let forward = crate::widget_abi::button(ui, "Forward", [0.; 2]);
        if forward {
            self.forward();
        } else if back {
            self.back();
        } else if up {
            self.up();
        } else if entered {
            self.refresh();
        }
    }
    /// The folder just left stays highlighted in its parent's list.
    fn up(&mut self) {
        let parent = Path::new(&self.folder).parent().map(Path::to_path_buf);
        if let Some(parent) = parent {
            self.folder_highlight = Some(folder_name(&self.folder));
            self.history.push(self.folder.clone());
            self.folder = parent.display().to_string();
            self.refresh();
        }
    }
    fn back(&mut self) {
        if let Some(previous) = self.history.pop() {
            self.forward.push(self.folder.clone());
            self.folder_highlight = None;
            self.folder = previous;
            self.refresh();
        }
    }
    /// Forward to a parent shows the folder Back came from, still highlighted.
    fn forward(&mut self) {
        if let Some(next) = self.forward.pop() {
            let left = folder_name(&self.folder);
            self.history.push(self.folder.clone());
            let into_parent = Path::new(&self.folder).parent() == Some(Path::new(&next));
            self.folder_highlight = into_parent.then_some(left);
            self.folder = next;
            self.refresh();
        }
    }
    pub(super) fn navigate(&mut self, name: &str) {
        self.history.push(self.folder.clone());
        self.forward.clear();
        self.folder_highlight = None;
        self.folder = Path::new(&self.folder).join(name).display().to_string();
        self.refresh();
    }
    fn shown_count(&self, filter: FileFilter) -> usize {
        let rows = self.listing.as_deref().unwrap_or_default();
        rows.iter().filter(|row| listed(row, filter)).count()
    }
}
/// The small toolbar's two groups of choices.
#[derive(Clone, Copy)]
enum Toolbar {
    Pane(NavigationPane),
    View(FileListView),
}
fn navigation_headings(pane: NavigationPane) -> &'static [&'static str] {
    match pane {
        NavigationPane::RecentFolders => &["Libraries", "Recent Folders"],
        NavigationPane::SystemLocations => &["System Locations"],
    }
}
fn folder_name(folder: &str) -> String {
    Path::new(folder).file_name().map_or_else(
        || folder.to_string(),
        |name| name.to_string_lossy().into_owned(),
    )
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
fn status_text(selected: usize, count: usize) -> String {
    match selected {
        0 => format!("{count} items"),
        1 => "1 item selected".to_string(),
        several => format!("{several} items selected"),
    }
}
pub(super) fn listed(row: &FolderEntry, filter: FileFilter) -> bool {
    row.is_folder || filter.keeps(&row.name)
}
pub(super) fn row_label(row: &FolderEntry) -> String {
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
    file_name_field_focused(ui, name, false)
}
/// Focus and select the filename after a dialog warning; example: `file_name_field_focused(ui, name, true)`.
pub(crate) fn file_name_field_focused(ui: &Ui, name: &mut String, focus: bool) -> bool {
    ui.align_text_to_frame_padding();
    ui.text("File name");
    ui.same_line_with_pos(110.);
    if focus || ui.is_window_appearing() {
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
    ui.align_text_to_frame_padding();
    ui.text("Files of type");
    ui.same_line_with_pos(110.);
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
    dialog_button_clicks(ui, accept, can_accept, entered)
        .or_else(|| escape.pressed(ui).then_some(DialogEnd::Cancelled))
}
/// Draw the footer when a caller routes Escape separately.
pub(crate) fn dialog_button_clicks(
    ui: &Ui,
    accept: &str,
    can_accept: bool,
    entered: bool,
) -> Option<DialogEnd> {
    ui.dummy([0., 2.]);
    let disabled = ui.begin_disabled(!can_accept);
    let accepted = (crate::theme::primary_button(ui, accept, [80., 28.]) || entered) && can_accept;
    drop(disabled);
    ui.same_line_with_spacing(0., 6.);
    let cancelled = crate::theme::secondary_button(ui, "Cancel", [80., 28.]);
    if accepted {
        return Some(DialogEnd::Accepted);
    }
    cancelled.then_some(DialogEnd::Cancelled)
}
#[cfg(test)]
#[path = "../../../tests/application/document-saving/folder_browser.rs"]
mod tests;
