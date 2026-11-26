// SPDX-License-Identifier: GPL-2.0-only
//! How a file dialog's list draws one entry, by the view it is in.
use super::{FileListView, row_label};
use crate::application::document_saving::save_as::draw_preview_cells;
use crate::application::file_system::FolderEntry;
use imgui::Ui;
use std::path::Path;

const DATE_COLUMN: f32 = 190.;
const TYPE_COLUMN: f32 = 370.;
const SIZE_COLUMN: f32 = 450.;
const TILE_HEIGHT: f32 = 44.;
const PICTURE_IN_ROW: f32 = 40.;
const CELL_GAP: f32 = 4.;

/// True when a press landed on the Size heading.
pub(in super::super) fn details_heading(ui: &Ui) -> bool {
    ui.text("Name");
    ui.same_line_with_pos(DATE_COLUMN);
    ui.text("Date modified");
    ui.same_line_with_pos(TYPE_COLUMN);
    ui.text("Type");
    ui.same_line_with_pos(SIZE_COLUMN);
    ui.text("Size");
    ui.is_item_clicked()
}
pub(in super::super) fn details_row(ui: &Ui, row: &FolderEntry, highlighted: bool) -> RowHit {
    let chosen = crate::widget_abi::dialog_selectable(ui, &row_label(row), highlighted);
    let hit = RowHit::current(ui, chosen);
    ui.same_line_with_pos(DATE_COLUMN);
    ui.text_disabled(modified_text(row));
    ui.same_line_with_pos(TYPE_COLUMN);
    ui.text_disabled(entry_type(row));
    if !row.is_folder {
        ui.same_line_with_pos(SIZE_COLUMN);
        ui.text_disabled(format!("{} B", row.size));
    }
    hit
}
pub(in super::super) struct RowHit {
    pub chosen: bool,
    pub hovered: bool,
    pub minimum: [f32; 2],
    pub maximum: [f32; 2],
}
impl RowHit {
    pub fn current(ui: &Ui, chosen: bool) -> Self {
        Self {
            chosen,
            hovered: ui.is_item_hovered(),
            minimum: ui.item_rect_min(),
            maximum: ui.item_rect_max(),
        }
    }
}
/// How a picture view lays out one file: its size, the picture's side, and where the name goes.
/// A width of zero takes the row.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(in super::super) struct FileCell {
    pub size: [f32; 2],
    picture: f32,
    name_beside: bool,
    /// Whether a second line gives the file's type and size.
    described: bool,
}
impl FileCell {
    /// The view's cell; the List and Details views have rows instead.
    pub(in super::super) fn of_view(view: FileListView) -> Option<Self> {
        let cell = |size, picture, name_beside, described| Self {
            size,
            picture,
            name_beside,
            described,
        };
        match view {
            FileListView::ExtraLargeIcons => Some(cell([148., 132.], 108., false, false)),
            FileListView::LargeIcons => Some(cell([116., 100.], 76., false, false)),
            FileListView::MediumIcons => Some(cell([96., 72.], 48., false, false)),
            FileListView::SmallIcons => Some(cell([200., 22.], 18., true, false)),
            FileListView::Tiles => Some(cell([250., TILE_HEIGHT], PICTURE_IN_ROW, true, true)),
            FileListView::Content => Some(cell([0., TILE_HEIGHT], PICTURE_IN_ROW, true, true)),
            FileListView::List | FileListView::Details => None,
        }
    }
}
/// Keep every tile in the layout, drawing pictures only where the listing can show them.
pub(in super::super) fn cell_row(
    ui: &Ui,
    row: &FolderEntry,
    highlighted: bool,
    cell: FileCell,
    picture: Option<&crate::GuiPreviewImage>,
) -> bool {
    let chosen = ui
        .selectable_config(format!("##cell-{}", row.name))
        .flags(imgui::SelectableFlags::DONT_CLOSE_POPUPS)
        .selected(highlighted)
        .size(cell.size)
        .build();
    if !ui.is_item_visible() {
        return chosen;
    }
    let minimum = ui.item_rect_min();
    let maximum = ui.item_rect_max();
    if let Some(picture) = picture {
        draw_preview_cells(ui, picture, minimum, [cell.picture; 2]);
    }
    let name_at = if cell.name_beside {
        [minimum[0] + cell.picture + CELL_GAP, minimum[1]]
    } else {
        [minimum[0], minimum[1] + cell.picture + CELL_GAP]
    };
    let list = ui.get_window_draw_list();
    list.with_clip_rect_intersect(minimum, maximum, || {
        let color = ui.style_color(imgui::StyleColor::Text);
        list.add_text(name_at, color, row_label(row));
        if cell.described {
            let muted = ui.style_color(imgui::StyleColor::TextDisabled);
            let line = [name_at[0], name_at[1] + ui.text_line_height_with_spacing()];
            list.add_text(line, muted, format!("{} {} B", entry_type(row), row.size));
        }
    });
    chosen
}
pub(super) fn entry_type(row: &FolderEntry) -> &'static str {
    if row.is_folder {
        return "Folder";
    }
    match Path::new(&row.name)
        .extension()
        .and_then(|extension| extension.to_str())
    {
        Some("opd") => "Part",
        Some("oad") => "Assembly",
        Some("odd") => "Drawing",
        Some("ord") => "Presentation",
        _ => "File",
    }
}
fn modified_text(row: &FolderEntry) -> String {
    row.modified
        .map_or_else(String::new, crate::application::file_system::timestamp_text)
}
