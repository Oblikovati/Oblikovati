// SPDX-License-Identifier: GPL-2.0-only
//! The dialogs' file list: its rows, their order, and how they are chosen, one or several.
use super::folder_browser::file_rows::{FileCell, RowHit, cell_row, details_heading, details_row};
use super::folder_browser::{FileFilter, FileListView, FolderBrowser, LISTING_HEIGHT, listed};
use crate::application::file_system::FolderEntry;
use imgui::{MouseButton, Ui};

const ROW_MENU: &str = "##file-row-menu";
const RECTANGLE_FILL: [f32; 4] = [0.3, 0.55, 0.9, 0.25];
const RECTANGLE_EDGE: [f32; 4] = [0.3, 0.55, 0.9, 0.9];
/// A press that travels less than this is a click on nothing, not a rectangle.
const DRAG_REACH: f32 = 4.;

/// What happened in the list this frame.
#[derive(Default)]
pub(super) struct ListingOutcome {
    pub chosen: Option<String>,
    pub entered: Option<String>,
}
/// What a row's menu or a double-click asks of the dialog; it takes these once.
#[derive(Debug, Default, PartialEq, Eq)]
pub(crate) enum FileRequest {
    #[default]
    None,
    Open(String),
    Properties(String),
}
/// A grid view's next cell shares the row while it still fits.
fn continue_the_grid(ui: &Ui, view: FileListView) {
    let Some(cell) = FileCell::of_view(view) else {
        return;
    };
    let next_end = ui.item_rect_max()[0] + ui.clone_style().item_spacing[0] + cell.size[0];
    if cell.size[0] > 0. && next_end < ui.window_pos()[0] + ui.window_content_region_max()[0] {
        ui.same_line();
    }
}
impl FolderBrowser {
    pub(super) fn sorted_rows(&self, filter: FileFilter) -> Vec<&FolderEntry> {
        let rows = self.listing.as_deref().unwrap_or_default();
        let mut shown: Vec<&FolderEntry> = rows.iter().filter(|row| listed(row, filter)).collect();
        if let Some(descending) = self.size_order_descending {
            shown.sort_by_key(|row| !row.is_folder);
            let files = shown.iter().filter(|row| row.is_folder).count();
            shown[files..].sort_by_key(|row| row.size);
            if descending {
                shown[files..].reverse();
            }
        }
        shown
    }
    /// The first press on Size lists the largest first; each after that turns the order round.
    fn toggle_size_order(&mut self) {
        self.size_order_descending = Some(self.size_order_descending != Some(true));
    }
    pub(super) fn listing_rows(&mut self, ui: &Ui, filter: FileFilter) -> ListingOutcome {
        let mut outcome = ListingOutcome::default();
        ui.child_window("##folder-listing")
            .size([0., LISTING_HEIGHT])
            .border(true)
            // A press on empty space starts a selection rectangle, not a move of the dialog.
            .flags(imgui::WindowFlags::NO_MOVE)
            .build(|| self.listing_contents(ui, filter, &mut outcome));
        outcome
    }
    fn listing_contents(&mut self, ui: &Ui, filter: FileFilter, outcome: &mut ListingOutcome) {
        if let Err(cause) = &self.listing {
            ui.text_disabled(cause);
            return;
        }
        if self.view == FileListView::Details && details_heading(ui) {
            self.toggle_size_order();
        }
        let rows: Vec<FolderEntry> = self.sorted_rows(filter).into_iter().cloned().collect();
        let mut over_a_row = false;
        let mut boxes = Vec::with_capacity(rows.len());
        for row in &rows {
            let hit = self.file_row(ui, row);
            boxes.push((row.name.clone(), hit.minimum, hit.maximum));
            over_a_row |= hit.hovered;
            self.row_actions(ui, row, hit.chosen, hit.hovered, outcome);
            continue_the_grid(ui, self.view);
        }
        let menu_bounds = self.row_menu_bounds;
        self.row_menu_bounds = self.row_menu(ui);
        self.selection_rectangle(ui, over_a_row, &boxes, menu_bounds);
    }
    fn row_actions(
        &mut self,
        ui: &Ui,
        row: &FolderEntry,
        chosen: bool,
        hovered: bool,
        outcome: &mut ListingOutcome,
    ) {
        if hovered && ui.is_mouse_double_clicked(MouseButton::Left) {
            if row.is_folder {
                outcome.entered = Some(row.name.clone());
            } else {
                self.request = FileRequest::Open(row.name.clone());
            }
            return;
        }
        if hovered && ui.is_mouse_clicked(MouseButton::Right) && !row.is_folder {
            self.menu_row = Some(row.name.clone());
            ui.open_popup(ROW_MENU);
        }
        if !chosen {
            return;
        }
        self.several.clear();
        if !row.is_folder {
            self.folder_highlight = None;
            outcome.chosen = Some(row.name.clone());
        } else if self.folders_select_first {
            self.selected = None;
            self.folder_highlight = Some(row.name.clone());
        } else {
            outcome.entered = Some(row.name.clone());
        }
    }
    fn row_menu(&mut self, ui: &Ui) -> Option<([f32; 2], [f32; 2])> {
        let _menu = ui.begin_popup(ROW_MENU)?;
        let name = self.menu_row.clone()?;
        let origin = ui.window_pos();
        let size = ui.window_size();
        if ui.menu_item("Open") {
            self.request = FileRequest::Open(name.clone());
        }
        for listed in ["Copy", "Rename"] {
            let _muted = ui.begin_disabled(true);
            ui.menu_item(listed);
        }
        if ui.menu_item("Properties") {
            self.request = FileRequest::Properties(name);
        }
        Some((origin, size))
    }
    /// A drag from empty list space chooses what it crosses; a click there chooses nothing.
    fn selection_rectangle(
        &mut self,
        ui: &Ui,
        over_a_row: bool,
        boxes: &[(String, [f32; 2], [f32; 2])],
        menu_bounds: Option<([f32; 2], [f32; 2])>,
    ) {
        let pointer = ui.io().mouse_pos;
        let over_a_row = over_a_row
            || boxes.iter().any(|(_, low, high)| {
                (0..2).all(|axis| pointer[axis] >= low[axis] && pointer[axis] < high[axis])
            });
        // The popup closes after the press; its dismissal click must also reach empty list space.
        let in_list = ui.is_window_hovered()
            || (menu_bounds.is_some_and(|(origin, size)| {
                (0..2).any(|axis| {
                    pointer[axis] < origin[axis] || pointer[axis] >= origin[axis] + size[axis]
                })
            }) && ui.is_window_hovered_with_flags(
                imgui::WindowHoveredFlags::ALLOW_WHEN_BLOCKED_BY_POPUP,
            ));
        if in_list && !over_a_row && ui.is_mouse_clicked(MouseButton::Left) {
            self.rectangle_start = Some(pointer);
        }
        let Some(start) = self.rectangle_start else {
            return;
        };
        let low = [start[0].min(pointer[0]), start[1].min(pointer[1])];
        let high = [start[0].max(pointer[0]), start[1].max(pointer[1])];
        if ui.is_mouse_down(MouseButton::Left) {
            let list = ui.get_window_draw_list();
            list.add_rect(low, high, RECTANGLE_FILL)
                .filled(true)
                .build();
            list.add_rect(low, high, RECTANGLE_EDGE).build();
            return;
        }
        self.rectangle_start = None;
        let travelled = (high[0] - low[0]).max(high[1] - low[1]);
        if travelled < DRAG_REACH {
            self.selected = None;
            self.folder_highlight = None;
            self.several.clear();
            return;
        }
        self.folder_highlight = None;
        self.several = boxes
            .iter()
            .filter(|(_, minimum, maximum)| {
                minimum[0] < high[0]
                    && maximum[0] > low[0]
                    && minimum[1] < high[1]
                    && maximum[1] > low[1]
            })
            .map(|(name, _, _)| name.clone())
            .collect();
        if let [only] = self.several.as_slice() {
            self.selected = Some(only.clone());
            self.several.clear();
        } else {
            self.selected = None;
        }
    }
    fn file_row(&self, ui: &Ui, row: &FolderEntry) -> RowHit {
        let highlighted = self.selected.as_deref() == Some(row.name.as_str())
            || self.folder_highlight.as_deref() == Some(row.name.as_str())
            || self.several.contains(&row.name);
        match (self.view, FileCell::of_view(self.view)) {
            (FileListView::Details, _) => details_row(ui, row, highlighted),
            (_, Some(cell)) => {
                let path = super::folder_browser::full_path(&self.folder, &row.name);
                let chosen = cell_row(ui, row, highlighted, cell, self.file_pictures.get(&path));
                RowHit::current(ui, chosen)
            }
            (_, None) => {
                let chosen = crate::widget_abi::dialog_selectable(
                    ui,
                    &super::folder_browser::row_label(row),
                    highlighted,
                );
                RowHit::current(ui, chosen)
            }
        }
    }
    /// The files a view needs a stored picture of, in the order it lists them.
    pub(crate) fn listed_for_pictures(&self, filter: FileFilter) -> Vec<String> {
        if !self.view.shows_pictures() {
            return Vec::new();
        }
        self.sorted_rows(filter)
            .into_iter()
            .filter(|row| !row.is_folder)
            .map(|row| super::folder_browser::full_path(&self.folder, &row.name))
            .collect()
    }
    /// Remember a file's stored picture for the rows that show one.
    pub(crate) fn note_file_picture(
        &mut self,
        path: &str,
        picture: Option<&crate::GuiPreviewImage>,
    ) {
        if let Some(picture) = picture {
            self.file_pictures.insert(path.to_owned(), picture.clone());
        }
    }
    /// A dialog opened again starts with nothing chosen, and reads the pictures again: a file
    /// may have been saved since.
    pub(crate) fn begin_showing(&mut self) {
        self.file_pictures.clear();
        self.folder_highlight = None;
        self.several.clear();
    }
    /// What a double-click or a row's menu asked for, once.
    pub(crate) fn take_request(&mut self) -> FileRequest {
        std::mem::take(&mut self.request)
    }
    /// Show a file as chosen, as when a dialog comes back to the one it last opened.
    pub(crate) fn select_file(&mut self, name: &str) {
        self.selected = Some(name.to_owned());
    }
    /// The names a rectangle chose; empty when one file, or none, is chosen.
    pub(crate) fn several_selected(&self) -> &[String] {
        &self.several
    }
    /// The file a click chose, while the list still shows it chosen.
    pub(crate) fn selected_file(&self) -> Option<&str> {
        self.selected.as_deref()
    }
    /// The folder row a click chose, or the one just left through Up.
    pub(crate) fn highlighted_folder(&self) -> Option<&str> {
        self.folder_highlight.as_deref()
    }
    /// A chosen folder's row becomes the shown folder; example: Open on a highlighted folder.
    pub(crate) fn enter_highlighted_folder(&mut self) -> bool {
        let Some(name) = self.folder_highlight.take() else {
            return false;
        };
        self.navigate(&name);
        true
    }
}
