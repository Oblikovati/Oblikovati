// SPDX-License-Identifier: GPL-2.0-only
//! The rows and cards of the recent list, with their checks, menus and pins.
use super::recent_view::{HeaderCheck, SortField, cell_text, column_width};
use super::{GuiRecentAction, GuiRecentDocument, HomeWorkspace};
use crate::GuiShellAction;
use crate::application::workspace_layout::browser_view::match_start;
use imgui::Ui;

#[path = "recent_cell_text.rs"]
mod recent_cell_text;
use recent_cell_text::{fit_recent_cell, reveal_full_cell, visible_source_text};

const CHECK_COLUMN: f32 = 28.;
/// The kind mark, wide enough that its letter never reads as part of the name.
const KIND_COLUMN: f32 = 22.;
const NAME_COLUMN: f32 = 170.;
const CARD_SIZE: [f32; 2] = [180., 90.];
const CARDS_PER_ROW: usize = 4;
const NOT_AVAILABLE: &str = "This command is not available in this build.";

fn open_action(path: &str) -> GuiShellAction {
    GuiShellAction::OpenDocument(oblikovati_api::wire::OpenDocumentArgs {
        full_document_name: path.to_owned(),
        ..Default::default()
    })
}
fn remove_action(path: &str) -> GuiShellAction {
    GuiShellAction::Recent(GuiRecentAction::Remove(path.to_owned()))
}
fn metadata_cell(ui: &Ui, full: &str, width: f32) {
    let shown = fit_recent_cell(full, width, |text| ui.calc_text_size(text)[0]);
    ui.text(shown.as_ref());
    reveal_full_cell(ui, full, &shown);
}
fn accent_row(ui: &Ui, height: f32) {
    let [x, y] = ui.cursor_screen_pos();
    let width = ui.content_region_avail()[0];
    let color = ui.style_color(imgui::StyleColor::Header);
    ui.get_window_draw_list()
        .add_rect([x, y], [x + width, y + height], color)
        .filled(true)
        .build();
}
impl HomeWorkspace {
    /// Above the list while anything is checked, in list and grid alike.
    pub(super) fn selected_actions(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        if self.checks.paths.is_empty() {
            return;
        }
        if ui.button("Open Selected") {
            self.open_selected(actions);
        }
        ui.same_line();
        if ui.button("Remove Selected") {
            self.remove_selected(actions);
        }
    }
    fn open_selected(&mut self, actions: &mut Vec<GuiShellAction>) {
        actions.extend(self.checks.paths.iter().map(|path| open_action(path)));
        self.checks.paths.clear();
    }
    fn remove_selected(&mut self, actions: &mut Vec<GuiShellAction>) {
        actions.extend(self.checks.paths.iter().map(|path| remove_action(path)));
        self.checks.paths.clear();
    }
    pub(super) fn list(
        &mut self,
        ui: &Ui,
        rows: &[&GuiRecentDocument],
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.list_header(ui, rows);
        ui.separator();
        for row in rows {
            self.list_row(ui, row, actions);
        }
    }
    fn list_header(&mut self, ui: &Ui, rows: &[&GuiRecentDocument]) {
        let state = self.checks.header(rows);
        let mut all = state == HeaderCheck::All;
        if ui.checkbox("##recent-all", &mut all) {
            self.checks.toggle_header(rows);
        }
        if state == HeaderCheck::Partial {
            partial_mark(ui);
        }
        self.header_cell(ui, SortField::Name, CHECK_COLUMN + KIND_COLUMN);
        let mut x = CHECK_COLUMN + KIND_COLUMN + NAME_COLUMN;
        for column in self.columns.shown.clone() {
            self.header_cell(ui, column, x);
            x += column_width(column);
        }
    }
    /// A header chooses the sort; the active one carries the direction at its right edge.
    fn header_cell(&mut self, ui: &Ui, column: SortField, x: f32) {
        ui.same_line_with_pos(x);
        let marked = self.sort.field == column;
        let label = if marked {
            format!(
                "{} {}###header-{}",
                column.label(),
                self.sort.mark(),
                column.label()
            )
        } else {
            format!("{}###header-{}", column.label(), column.label())
        };
        if ui.button(label) {
            self.sort.choose(column);
        }
    }
    fn list_row(&mut self, ui: &Ui, row: &GuiRecentDocument, actions: &mut Vec<GuiShellAction>) {
        let hovered = self.note_row_hover(ui);
        let checked = self.checks.contains(&row.path);
        if checked || hovered {
            accent_row(ui, ui.frame_height());
        }
        let mut box_state = checked;
        if ui.checkbox(format!("##check-{}", row.path), &mut box_state) {
            self.checks.set(&row.path, box_state);
        }
        ui.same_line_with_pos(CHECK_COLUMN);
        ui.text(kind_mark(row.kind));
        ui.same_line_with_pos(CHECK_COLUMN + KIND_COLUMN);
        self.name_cell(ui, row, actions);
        let mut x = CHECK_COLUMN + KIND_COLUMN + NAME_COLUMN;
        let spacing = ui.clone_style().item_spacing[0];
        for column in &self.columns.shown {
            ui.same_line_with_pos(x);
            let width = column_width(*column);
            metadata_cell(ui, cell_text(row, *column), width - spacing);
            x += width;
        }
        ui.same_line_with_pos(x + 4.);
        self.row_menu_when_shown(ui, row, hovered, actions);
    }
    fn name_cell(&self, ui: &Ui, row: &GuiRecentDocument, actions: &mut Vec<GuiShellAction>) {
        let style = ui.clone_style();
        let width = NAME_COLUMN - style.item_spacing[0] - 2. * style.frame_padding[0];
        let shown = fit_recent_cell(&row.name, width, |text| ui.calc_text_size(text)[0]);
        self.mark_matching_text(ui, visible_source_text(&shown, &row.name));
        if ui.button(format!("{shown}###open-{}", row.path)) {
            actions.push(open_action(&row.path));
        }
        reveal_full_cell(ui, &row.name, &shown);
    }
    /// A row under the pointer shows its trailing control; an open menu keeps it there.
    fn row_menu_when_shown(
        &mut self,
        ui: &Ui,
        row: &GuiRecentDocument,
        hovered: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if hovered || self.menu_row.as_deref() == Some(&row.path) {
            self.row_menu(ui, row, actions);
        } else {
            ui.dummy([0., ui.frame_height()]);
        }
    }
    /// The row about to be drawn: its whole width counts, including its plain text cells.
    fn note_row_hover(&mut self, ui: &Ui) -> bool {
        let [x, y] = ui.cursor_screen_pos();
        let width = ui.content_region_avail()[0];
        ui.is_mouse_hovering_rect([x, y], [x + width, y + ui.frame_height()])
    }
    /// The part of a name the applied search found, lit behind the name about to be drawn.
    fn mark_matching_text(&self, ui: &Ui, name: &str) {
        if self.applied.is_empty() {
            return;
        }
        let Some(start) = match_start(name, &self.applied) else {
            return;
        };
        let (Some(before), Some(matched)) = (
            name.get(..start),
            name.get(start..start + self.applied.len()),
        ) else {
            return;
        };
        let [x, y] = ui.cursor_screen_pos();
        let padding = ui.clone_style().frame_padding;
        let before = ui.calc_text_size(before)[0];
        let width = ui.calc_text_size(matched)[0];
        let color = ui.style_color(imgui::StyleColor::TextSelectedBg);
        let left = x + padding[0] + before;
        ui.get_foreground_draw_list()
            .add_rect(
                [left, y + padding[1]],
                [left + width, y + ui.frame_height() - padding[1]],
                color,
            )
            .filled(true)
            .build();
    }
    fn row_menu(&mut self, ui: &Ui, row: &GuiRecentDocument, actions: &mut Vec<GuiShellAction>) {
        let identity = format!("##actions-{}", row.path);
        if ui.button(format!("...##actions-button-{}", row.path)) {
            ui.open_popup(&identity);
            self.menu_row = Some(row.path.clone());
        }
        let Some(_menu) = ui.begin_popup(&identity) else {
            if self.menu_row.as_deref() == Some(&row.path) {
                self.menu_row = None;
            }
            return;
        };
        if self.checks.paths.is_empty() {
            self.single_row_items(ui, row, actions);
        } else {
            if ui.menu_item("Open Selected") {
                self.open_selected(actions);
            }
            if ui.menu_item("Remove Selected") {
                self.remove_selected(actions);
            }
        }
    }
    /// The quick opening mode a kind offers; both are named but not yet carried out by Open.
    fn kind_specific_item(&self, ui: &Ui, row: &GuiRecentDocument) {
        let label = match row.kind {
            crate::GuiTemplateKind::Assembly => "Open Express",
            crate::GuiTemplateKind::Drawing => "Defer Open",
            _ => return,
        };
        ui.menu_item_config(label).enabled(false).build();
        if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
            ui.tooltip_text(NOT_AVAILABLE);
        }
    }
    fn single_row_items(
        &mut self,
        ui: &Ui,
        row: &GuiRecentDocument,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if ui.menu_item("Open") {
            actions.push(open_action(&row.path));
        }
        if ui.menu_item("Open with options") {
            self.open_options.request(row);
        }
        self.kind_specific_item(ui, row);
        if ui.menu_item("Remove from Recent") {
            actions.push(remove_action(&row.path));
        }
        for label in ["Open Containing Folder", "Explore Containing Folder"] {
            ui.menu_item_config(label).enabled(false).build();
            if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
                ui.tooltip_text(NOT_AVAILABLE);
            }
        }
        if ui.menu_item("Document Properties") {
            self.request_properties(&row.path);
        }
    }
    /// Cards show a document kind's mark: only an open document has a live picture to offer.
    pub(super) fn cards(
        &mut self,
        ui: &Ui,
        rows: &[&GuiRecentDocument],
        actions: &mut Vec<GuiShellAction>,
    ) {
        let mut hovered = None;
        for (index, row) in rows.iter().enumerate() {
            if index % CARDS_PER_ROW != 0 {
                ui.same_line();
            }
            ui.group(|| self.card(ui, row, actions));
            if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::RECT_ONLY) {
                hovered = Some(row.path.clone());
            }
        }
        // The control appears on the frame after the pointer arrives, so that frame must follow.
        self.hover_changed = hovered != self.hovered_card;
        self.hovered_card = hovered;
    }
    fn card(&mut self, ui: &Ui, row: &GuiRecentDocument, actions: &mut Vec<GuiShellAction>) {
        let checked = self.checks.contains(&row.path);
        if checked {
            accent_row(ui, CARD_SIZE[1]);
        }
        let mut box_state = checked;
        if ui.checkbox(format!("##card-check-{}", row.path), &mut box_state) {
            self.checks.set(&row.path, box_state);
        }
        ui.same_line();
        let hovered = self.hovered_card.as_deref() == Some(&row.path);
        if hovered || row.pinned {
            pin_control(ui, row, actions);
            ui.same_line();
        }
        if hovered {
            self.row_menu(ui, row, actions);
        } else {
            ui.dummy([0., ui.frame_height()]);
        }
        if crate::widget_abi::button(
            ui,
            &format!("{:?}###recent-card-{}", row.kind, row.path),
            [CARD_SIZE[0], CARD_SIZE[1] - 56.],
        ) {
            actions.push(open_action(&row.path));
        }
        ui.text(&row.name);
        ui.text_disabled(&row.modified_text);
    }
}
/// The letter beside a row's name: the same kinds read the same everywhere on the list.
pub(crate) fn kind_mark(kind: crate::GuiTemplateKind) -> &'static str {
    use crate::GuiTemplateKind::{Assembly, Drawing, Part, Presentation, SheetMetal};
    match kind {
        Part => "P",
        SheetMetal => "S",
        Assembly => "A",
        Drawing => "D",
        Presentation => "R",
    }
}
fn pin_control(ui: &Ui, row: &GuiRecentDocument, actions: &mut Vec<GuiShellAction>) {
    let label = if row.pinned { "Unpin" } else { "Pin" };
    if ui.button(format!("{label}###pin-{}", row.path)) {
        actions.push(GuiShellAction::Recent(GuiRecentAction::SetPinned {
            path: row.path.clone(),
            pinned: !row.pinned,
        }));
    }
}
/// The header's partial state: a short horizontal mark in its box.
fn partial_mark(ui: &Ui) {
    let [minimum, maximum] = [ui.item_rect_min(), ui.item_rect_max()];
    let middle = (minimum[1] + maximum[1]) * 0.5;
    let color = ui.style_color(imgui::StyleColor::CheckMark);
    ui.get_window_draw_list()
        .add_line([minimum[0] + 4., middle], [maximum[0] - 4., middle], color)
        .thickness(2.)
        .build();
}
