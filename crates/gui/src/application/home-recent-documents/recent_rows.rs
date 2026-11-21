// SPDX-License-Identifier: GPL-2.0-only
//! The rows and cards of the recent list, with their checks, menus and pins.
use super::recent_view::{HeaderCheck, SortField, cell_text, column_width};
use super::{GuiRecentAction, GuiRecentDocument, HomeWorkspace};
use crate::GuiShellAction;
use crate::application::workspace_layout::browser_view::match_start;
use imgui::Ui;

#[path = "recent_cell_text.rs"]
mod recent_cell_text;
use recent_cell_text::{
    fit_recent_cell, fit_recent_timestamp, reveal_full_cell, visible_source_text,
};

const CHECK_COLUMN: f32 = 28.;
/// The kind mark, wide enough that its letter never reads as part of the name.
const KIND_COLUMN: f32 = 22.;
const NAME_COLUMN: f32 = 170.;
const ACTION_GAP: f32 = 4.;
const CARD_SIZE: [f32; 2] = [180., 90.];
const MAX_CARDS_PER_ROW: usize = 4;
const NOT_AVAILABLE: &str = "This command is not available in this build.";
type RecentColumnWidths = [(SortField, f32); SortField::MENU.len()];

#[derive(Clone, Copy)]
struct RecentListLayout {
    widths: RecentColumnWidths,
    spacing: f32,
    frame_padding: f32,
}

impl RecentListLayout {
    fn for_columns(ui: &Ui, columns: &[SortField]) -> Self {
        let style = ui.clone_style();
        let spacing = style.item_spacing[0];
        let frame_padding = 2. * style.frame_padding[0];
        let actions = ui.calc_text_size("...")[0] + frame_padding + spacing + ACTION_GAP;
        let available = ui.content_region_avail()[0] - CHECK_COLUMN - KIND_COLUMN - actions;
        let marks = ui.calc_text_size(" v")[0].max(ui.calc_text_size(" ^")[0]);
        let minimum = SortField::MENU.map(|field| {
            let width = ui.calc_text_size(field.label())[0] + marks + spacing + frame_padding;
            (field, width)
        });
        Self {
            widths: fit_recent_columns(minimum, columns, available),
            spacing,
            frame_padding,
        }
    }

    fn width(self, column: SortField) -> f32 {
        self.widths
            .iter()
            .find(|(field, _)| *field == column)
            .unwrap_or_else(|| panic!("recent column {column:?}: expected a measured width"))
            .1
    }
}

fn fit_recent_columns(
    minima: RecentColumnWidths,
    shown: &[SortField],
    available: f32,
) -> RecentColumnWidths {
    let (minimum, extra) = minima
        .iter()
        .filter(|(field, _)| *field == SortField::Name || shown.contains(field))
        .fold((0., 0.), |(minimum, extra), &(field, width)| {
            (
                minimum + width,
                extra + extra_recent_column_width(field, width),
            )
        });
    // Keep headings and sort marks legible before assigning the space for longer cell values.
    let heading_scale = (available / minimum).clamp(0., 1.);
    let extra_scale = if extra > 0. {
        ((available - minimum) / extra).clamp(0., 1.)
    } else {
        0.
    };
    minima.map(|(field, width)| {
        (
            field,
            heading_scale * width + extra_scale * extra_recent_column_width(field, width),
        )
    })
}

fn extra_recent_column_width(field: SortField, minimum: f32) -> f32 {
    let preferred = if field == SortField::Name {
        NAME_COLUMN
    } else {
        column_width(field)
    };
    (preferred - minimum).max(0.)
}

fn open_action(path: &str) -> GuiShellAction {
    GuiShellAction::OpenDocument(oblikovati_api::wire::OpenDocumentArgs {
        full_document_name: path.to_owned(),
        ..Default::default()
    })
}
fn remove_action(path: &str) -> GuiShellAction {
    GuiShellAction::Recent(GuiRecentAction::Remove(path.to_owned()))
}
fn metadata_cell(ui: &Ui, full: &str, width: f32, timestamp: bool) {
    let measure = |text: &str| ui.calc_text_size(text)[0];
    let shown = if timestamp {
        fit_recent_timestamp(full, width, measure)
    } else {
        fit_recent_cell(full, width, measure)
    };
    ui.text(shown.as_ref());
    reveal_full_cell(ui, full, &shown);
}
fn accent_row(ui: &Ui, height: f32, width: f32) {
    let [x, y] = ui.cursor_screen_pos();
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
        let layout = RecentListLayout::for_columns(ui, &self.columns.shown);
        self.list_header(ui, rows, layout);
        ui.separator();
        for row in rows {
            self.list_row(ui, row, layout, actions);
        }
    }
    fn list_header(&mut self, ui: &Ui, rows: &[&GuiRecentDocument], layout: RecentListLayout) {
        let state = self.checks.header(rows);
        let mut all = state == HeaderCheck::All;
        if ui.checkbox("##recent-all", &mut all) {
            self.checks.toggle_header(rows);
        }
        if state == HeaderCheck::Partial {
            partial_mark(ui);
        }
        self.header_cell(ui, SortField::Name, CHECK_COLUMN + KIND_COLUMN, layout);
        let mut x = CHECK_COLUMN + KIND_COLUMN + layout.width(SortField::Name);
        for column in self.columns.shown.clone() {
            self.header_cell(ui, column, x, layout);
            x += layout.width(column);
        }
    }
    /// A header chooses the sort; the active one carries the direction at its right edge.
    fn header_cell(&mut self, ui: &Ui, column: SortField, x: f32, layout: RecentListLayout) {
        ui.same_line_with_pos(x);
        let marked = self.sort.field == column;
        let label = if marked {
            format!("{} {}", column.label(), self.sort.mark())
        } else {
            column.label().to_owned()
        };
        let width = layout.width(column) - layout.spacing - layout.frame_padding;
        let shown = fit_recent_cell(&label, width, |text| ui.calc_text_size(text)[0]);
        if ui.button(format!("{shown}###header-{}", column.label())) {
            self.sort.choose(column);
        }
        reveal_full_cell(ui, &label, &shown);
    }
    fn list_row(
        &mut self,
        ui: &Ui,
        row: &GuiRecentDocument,
        layout: RecentListLayout,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let hovered = self.note_row_hover(ui);
        let checked = self.checks.contains(&row.path);
        if checked || hovered {
            accent_row(ui, ui.frame_height(), ui.content_region_avail()[0]);
        }
        let mut box_state = checked;
        if ui.checkbox(format!("##check-{}", row.path), &mut box_state) {
            self.checks.set(&row.path, box_state);
        }
        ui.same_line_with_pos(CHECK_COLUMN);
        ui.text(kind_mark(row.kind));
        ui.same_line_with_pos(CHECK_COLUMN + KIND_COLUMN);
        self.name_cell(ui, row, layout, actions);
        let mut x = CHECK_COLUMN + KIND_COLUMN + layout.width(SortField::Name);
        for column in &self.columns.shown {
            ui.same_line_with_pos(x);
            let width = layout.width(*column);
            let timestamp = matches!(column, SortField::Modified | SortField::Opened);
            metadata_cell(
                ui,
                cell_text(row, *column),
                width - layout.spacing,
                timestamp,
            );
            x += width;
        }
        ui.same_line_with_pos(x + ACTION_GAP);
        self.row_menu_when_shown(ui, row, hovered, actions);
    }
    fn name_cell(
        &self,
        ui: &Ui,
        row: &GuiRecentDocument,
        layout: RecentListLayout,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let width = layout.width(SortField::Name) - layout.spacing - layout.frame_padding;
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
        let (card_width, columns) = recent_card_grid(
            ui.content_region_avail()[0],
            ui.clone_style().item_spacing[0],
        );
        let mut hovered = None;
        for (index, row) in rows.iter().enumerate() {
            if index % columns != 0 {
                ui.same_line();
            }
            ui.group(|| self.card(ui, row, card_width, actions));
            if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::RECT_ONLY) {
                hovered = Some(row.path.clone());
            }
        }
        // The control appears on the frame after the pointer arrives, so that frame must follow.
        self.hover_changed = hovered != self.hovered_card;
        self.hovered_card = hovered;
    }
    fn card(
        &mut self,
        ui: &Ui,
        row: &GuiRecentDocument,
        width: f32,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let checked = self.checks.contains(&row.path);
        if checked {
            accent_row(ui, CARD_SIZE[1], width);
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
            [width, CARD_SIZE[1] - 56.],
        ) {
            actions.push(open_action(&row.path));
        }
        metadata_cell(ui, &row.name, width, false);
        let _dim = ui.push_style_color(
            imgui::StyleColor::Text,
            ui.style_color(imgui::StyleColor::TextDisabled),
        );
        metadata_cell(ui, &row.modified_text, width, true);
    }
}

fn recent_card_grid(available: f32, gap: f32) -> (f32, usize) {
    let width = available.clamp(1., CARD_SIZE[0]);
    // The last card reaches the edge without a trailing gap.
    let columns = ((available + gap) / (width + gap)).floor() as usize;
    (width, columns.clamp(1, MAX_CARDS_PER_ROW))
}

#[cfg(test)]
#[test]
fn recent_cards_wrap_at_the_available_width() {
    assert_eq!(recent_card_grid(1000., 8.), (180., 4));
    assert_eq!(recent_card_grid(600., 8.), (180., 3));
    assert_eq!(recent_card_grid(367., 8.), (180., 1));
    assert_eq!(recent_card_grid(368., 8.), (180., 2));
    assert_eq!(recent_card_grid(140., 8.), (140., 1));
}

#[cfg(test)]
#[test]
fn compact_recent_columns_keep_width_for_size_and_its_sort_mark() {
    let shown = &super::recent_view::ADJUSTABLE;
    let minimum =
        SortField::MENU.map(|field| (field, if field == SortField::Size { 62. } else { 108. }));
    let widths = fit_recent_columns(minimum, shown, 560.);
    let size = widths
        .into_iter()
        .find(|(field, _)| *field == SortField::Size)
        .unwrap()
        .1;
    assert!(size >= 62.);
    for width in [560., 320.] {
        let widths = fit_recent_columns(minimum, shown, width);
        let visible = widths
            .iter()
            .filter(|(field, _)| *field == SortField::Name || shown.contains(field));
        assert!((visible.map(|(_, width)| width).sum::<f32>() - width).abs() < 0.01);
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
