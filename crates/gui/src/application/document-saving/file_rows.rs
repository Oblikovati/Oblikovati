// SPDX-License-Identifier: GPL-2.0-only
//! How a file dialog's list draws one entry, in Details or Tiles view.
use super::row_label;
use crate::application::file_system::FolderEntry;
use imgui::Ui;
use std::path::Path;

const DATE_COLUMN: f32 = 190.;
const TYPE_COLUMN: f32 = 370.;
const SIZE_COLUMN: f32 = 450.;
const TILE_HEIGHT: f32 = 36.;

pub(super) fn details_heading(ui: &Ui) {
    ui.text("Name");
    ui.same_line_with_pos(DATE_COLUMN);
    ui.text("Date modified");
    ui.same_line_with_pos(TYPE_COLUMN);
    ui.text("Type");
    ui.same_line_with_pos(SIZE_COLUMN);
    ui.text("Size");
    ui.separator();
}
pub(super) fn details_row(ui: &Ui, row: &FolderEntry, highlighted: bool) -> bool {
    let chosen = crate::widget_abi::dialog_selectable(ui, &row_label(row), highlighted);
    ui.same_line_with_pos(DATE_COLUMN);
    ui.text_disabled(modified_text(row));
    ui.same_line_with_pos(TYPE_COLUMN);
    ui.text_disabled(entry_type(row));
    if !row.is_folder {
        ui.same_line_with_pos(SIZE_COLUMN);
        ui.text_disabled(format!("{} B", row.size));
    }
    chosen
}
/// A tile names the file and gives its type and size on the line beneath.
pub(super) fn tile_row(ui: &Ui, row: &FolderEntry, highlighted: bool) -> bool {
    let chosen = ui
        .selectable_config(row_label(row))
        .flags(imgui::SelectableFlags::DONT_CLOSE_POPUPS)
        .selected(highlighted)
        .size([0., TILE_HEIGHT])
        .build();
    let text = format!("{} {} B", entry_type(row), row.size);
    let minimum = ui.item_rect_min();
    let color = ui.style_color(imgui::StyleColor::TextDisabled);
    ui.get_window_draw_list()
        .add_text([minimum[0], minimum[1] + TILE_HEIGHT * 0.5], color, text);
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
