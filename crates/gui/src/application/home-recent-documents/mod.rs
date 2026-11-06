// SPDX-License-Identifier: GPL-2.0-only
//! The Home workspace: starting points on the left, recent documents on the right.
#[path = "open_options.rs"]
mod open_options;
#[path = "recent_filter.rs"]
pub(crate) mod recent_filter;
#[path = "recent_rows.rs"]
pub(crate) mod recent_rows;
#[path = "recent_view.rs"]
pub(crate) mod recent_view;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use recent_filter::{DateFilter, RecentFilter};
use recent_view::{RecentChecks, RecentColumns, RecentSort, SortField};

/// One recent file with the facts Home shows; example: a row in List View.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiRecentDocument {
    pub path: String,
    pub name: String,
    pub location: String,
    pub kind: crate::GuiTemplateKind,
    pub modified: Option<u64>,
    pub modified_text: String,
    pub modified_age_days: Option<u64>,
    pub opened: u64,
    pub opened_text: String,
    pub size_bytes: Option<u64>,
    pub size_text: String,
    pub pinned: bool,
    /// Named views saved in the file, listed by its open options.
    pub design_views: Vec<String>,
}
/// A recent-list change only the owner of the stored list can make.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiRecentAction {
    Remove(String),
    SetPinned { path: String, pinned: bool },
}
const RAIL_WIDTH: f32 = 150.;
const SIDE_PANEL_WIDTH: f32 = 240.;
const NOTICE_TITLE: &str = "Information";
const SORT_POPUP: &str = "##home-sort";

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum SidePanel {
    Filter,
    Columns,
}
/// How the list is shown and narrowed for this session.
#[derive(Default)]
pub(crate) struct HomeWorkspace {
    pub active: bool,
    /// The window in front was reached from Home, so closing it returns there.
    pub reached_from_home: bool,
    grid: bool,
    /// What the field holds; the list changes only when Search is activated.
    query: String,
    applied: String,
    history: Vec<String>,
    show_history: bool,
    filter: RecentFilter,
    /// The side panel showing, which stays up while its choices change the list.
    side_panel: Option<SidePanel>,
    /// A file waiting for the once-a-session notice before its properties open.
    noticed_file: Option<String>,
    notice_seen: bool,
    notice_opening: bool,
    sort: RecentSort,
    columns: RecentColumns,
    checks: RecentChecks,
    /// The card under the pointer as of the last frame, which shows its Pin control.
    hovered_card: Option<String>,
    pub hover_changed: bool,
    /// The row whose actions menu is open, which keeps its control when the pointer leaves it.
    menu_row: Option<String>,
    open_options: open_options::FileOpenOptions,
    /// A displayed column being dragged in Column Settings.
    dragged_column: Option<usize>,
    pub requests: HomeRequests,
}
#[derive(Default)]
pub(crate) struct HomeRequests {
    pub new_document: bool,
    pub open_document: bool,
    pub file_properties: Option<String>,
}
impl HomeWorkspace {
    /// Draw the workspace over the document area while it is active.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.active {
            return;
        }
        crate::widget_abi::window_geometry(
            ui,
            [area.left, area.top],
            [area.right - area.left, area.bottom - area.top],
            imgui::Condition::Always,
        );
        ui.window("##home-workspace")
            .flags(
                imgui::WindowFlags::NO_DECORATION
                    | imgui::WindowFlags::NO_MOVE
                    | imgui::WindowFlags::NO_DOCKING
                    | imgui::WindowFlags::NO_SAVED_SETTINGS
                    | imgui::WindowFlags::NO_FOCUS_ON_APPEARING
                    | imgui::WindowFlags::NO_BRING_TO_FRONT_ON_FOCUS,
            )
            .build(|| {
                self.rail(ui);
                ui.same_line_with_pos(RAIL_WIDTH);
                ui.group(|| self.recent_area(ui, state, actions));
                self.open_options.panel(ui, actions);
                self.properties_notice(ui);
            });
        self.side_panel_window(ui, area);
    }
    fn rail(&mut self, ui: &Ui) {
        ui.group(|| {
            ui.text("Home");
            self.requests.new_document |= ui.button("New");
            self.requests.open_document |= ui.button("Open");
        });
    }
    fn recent_area(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        ui.text("Recent");
        self.view_controls(ui);
        self.search_controls(ui);
        self.search_history(ui);
        let rows = self.shown_rows(state);
        self.checks.keep_only(&rows);
        self.selected_actions(ui, actions);
        if rows.is_empty() {
            self.no_results(ui);
        } else if self.grid {
            self.sort_controls(ui);
            self.cards(ui, &rows, actions);
        } else {
            self.list(ui, &rows, actions);
        }
    }
    fn view_controls(&mut self, ui: &Ui) {
        if ui.button("List View") {
            self.grid = false;
        }
        ui.same_line();
        if ui.button("Grid View") {
            self.grid = true;
        }
        ui.same_line();
        self.filter_control(ui);
        ui.same_line();
        self.columns_control(ui);
    }
    fn search_controls(&mut self, ui: &Ui) {
        ui.set_next_item_width(260.);
        ui.input_text("##search-recent", &mut self.query)
            .hint("Search recent")
            .build();
        if ui.is_item_activated() && self.query.is_empty() {
            self.show_history = true;
        }
        if !self.query.is_empty() && ui.is_item_active() {
            self.show_history = false;
        }
        ui.same_line();
        if ui.button("Search") {
            self.apply_search();
        }
        ui.same_line();
        if ui.button("x##clear-search") {
            self.query.clear();
            self.show_history = true;
        }
    }
    /// Activating Search is what changes the list; clearing the field alone leaves it filtered.
    fn apply_search(&mut self) {
        self.applied = self.query.trim().to_owned();
        self.show_history = false;
        if !self.applied.is_empty() && !self.history.contains(&self.applied) {
            self.history.insert(0, self.applied.clone());
        }
    }
    fn search_history(&mut self, ui: &Ui) {
        if !self.show_history || self.history.is_empty() {
            return;
        }
        ui.text_disabled("Search History");
        for term in self.history.clone() {
            if ui.selectable(&term) {
                self.query = term;
                self.apply_search();
            }
        }
        ui.separator();
    }
    fn toggle_panel(&mut self, panel: SidePanel) {
        self.side_panel = (self.side_panel != Some(panel)).then_some(panel);
    }
    fn filter_control(&mut self, ui: &Ui) {
        let count = self.filter.active_count();
        let label = if count == 0 {
            "Filter###home-filter-control".to_owned()
        } else {
            format!("Filter ({count})###home-filter-control")
        };
        if ui.button(label) {
            self.toggle_panel(SidePanel::Filter);
        }
        if count > 0 {
            ui.same_line();
            if ui.button("x##clear-filter") {
                self.filter = RecentFilter::default();
            }
        }
    }
    /// A panel over the list's right side, not a popup: a choice in it, or the clear control
    /// beside Filter, leaves it up.
    fn side_panel_window(
        &mut self,
        ui: &Ui,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
    ) {
        let Some(panel) = self.side_panel else {
            return;
        };
        if ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.side_panel = None;
            return;
        }
        let position = [area.right - SIDE_PANEL_WIDTH - 8., area.top + 8.];
        ui.window("##home-side-panel")
            .position(position, imgui::Condition::Always)
            .size([SIDE_PANEL_WIDTH, 0.], imgui::Condition::Always)
            .flags(
                imgui::WindowFlags::NO_DECORATION
                    | imgui::WindowFlags::ALWAYS_AUTO_RESIZE
                    | imgui::WindowFlags::NO_SAVED_SETTINGS,
            )
            .build(|| match panel {
                SidePanel::Filter => self.filter_rows(ui),
                SidePanel::Columns => self.column_rows(ui),
            });
    }
    fn filter_rows(&mut self, ui: &Ui) {
        ui.text("Date Modified");
        for choice in DateFilter::ALL {
            if ui.radio_button_bool(choice.label(), self.filter.date == choice) {
                self.filter.date = choice;
            }
        }
        ui.separator();
        ui.text("File Types");
        for (choice, mut chosen) in self.filter.type_rows() {
            if ui.checkbox(choice.label(), &mut chosen) {
                self.filter.set_type(choice, chosen);
            }
        }
        ui.separator();
        if ui.button("Clear filters") {
            self.filter = RecentFilter::default();
        }
    }
    fn columns_control(&mut self, ui: &Ui) {
        if ui.button("Column Settings") {
            self.toggle_panel(SidePanel::Columns);
        }
    }
    fn column_rows(&mut self, ui: &Ui) {
        ui.text("Displayed Columns");
        for (index, column) in self.columns.shown.clone().into_iter().enumerate() {
            self.displayed_column(ui, index, column);
        }
        ui.separator();
        ui.text("Available Columns");
        for column in recent_view::ADJUSTABLE {
            let mut shown = self.columns.shown.contains(&column);
            if ui.checkbox(format!("{}##available", column.label()), &mut shown) {
                self.columns.set_shown(column, shown);
            }
        }
        ui.separator();
        if ui.button("Reset to default") {
            self.columns = RecentColumns::default();
        }
    }
    /// A row moves to whichever displayed row the pointer is over while it is held.
    fn displayed_column(&mut self, ui: &Ui, index: usize, column: SortField) {
        ui.selectable(format!("{}##displayed", column.label()));
        if ui.is_item_activated() {
            self.dragged_column = Some(index);
        }
        let over = ui
            .is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_BLOCKED_BY_ACTIVE_ITEM);
        if !ui.is_mouse_down(imgui::MouseButton::Left) {
            self.dragged_column = None;
        }
        if let Some(from) = self.dragged_column.filter(|from| over && *from != index) {
            self.columns.move_to(from, index);
            self.dragged_column = Some(index);
        }
    }
    /// Properties are read from the file: said once a session before the first one opens.
    pub(super) fn request_properties(&mut self, path: &str) {
        if self.notice_seen {
            self.requests.file_properties = Some(path.to_owned());
            return;
        }
        self.noticed_file = Some(path.to_owned());
        self.notice_opening = true;
    }
    fn properties_notice(&mut self, ui: &Ui) {
        if std::mem::take(&mut self.notice_opening) {
            ui.open_popup(NOTICE_TITLE);
        }
        let Some(_notice) = ui.begin_modal_popup(NOTICE_TITLE) else {
            return;
        };
        ui.text("These properties come from the file on disk.");
        ui.text("Open the document to change its own properties.");
        if crate::widget_abi::button(ui, "OK", [80., 0.]) {
            self.notice_seen = true;
            self.requests.file_properties = self.noticed_file.take();
            ui.close_current_popup();
        }
    }
    fn sort_controls(&mut self, ui: &Ui) {
        if ui.button(format!(
            "Sort by {}###home-sort-control",
            self.sort.field.label()
        )) {
            ui.open_popup(SORT_POPUP);
        }
        ui.same_line();
        if ui.button(format!("{}##sort-direction", self.sort.mark())) {
            self.sort.descending = !self.sort.descending;
        }
        if let Some(_popup) = ui.begin_popup(SORT_POPUP) {
            for field in SortField::MENU {
                if ui
                    .menu_item_config(field.label())
                    .selected(self.sort.field == field)
                    .build()
                {
                    self.sort.choose(field);
                }
            }
        }
    }
    fn no_results(&mut self, ui: &Ui) {
        ui.dummy([0., 60.]);
        ui.text("No results");
        ui.text_disabled("Try a different search, or remove the filters.");
        if ui.button("Clear search and filters") {
            self.query.clear();
            self.applied.clear();
            self.filter = RecentFilter::default();
        }
    }
    fn shown_rows<'a>(&self, state: &'a GuiCommandState) -> Vec<&'a GuiRecentDocument> {
        let mut rows =
            recent_filter::matching_rows(&state.recent_documents, &self.applied, &self.filter);
        self.sort.apply(&mut rows);
        rows
    }
}
const DAY_SECONDS: u64 = 86_400;
impl GuiRecentDocument {
    /// Read a recent file's facts; a missing file keeps its row with blank facts.
    /// Example: `GuiRecentDocument::of_file("/work/Part1.opd", opened, false, SystemTime::now())`.
    pub fn of_file(path: &str, last_opened: u64, pinned: bool, now: std::time::SystemTime) -> Self {
        use crate::application::file_system::timestamp_text;
        let file = std::path::Path::new(path);
        let part = |text: Option<&std::ffi::OsStr>| {
            text.map_or_else(String::new, |text| text.to_string_lossy().into_owned())
        };
        let facts = std::fs::metadata(file).ok();
        let modified = facts.as_ref().and_then(|facts| facts.modified().ok());
        let opened = std::time::UNIX_EPOCH + std::time::Duration::from_secs(last_opened);
        Self {
            path: path.to_owned(),
            name: part(file.file_stem()),
            location: part(file.parent().map(std::path::Path::as_os_str)),
            kind: kind_of_extension(file.extension().and_then(|text| text.to_str())),
            modified: modified.map(unix_seconds),
            modified_text: modified.map_or_else(String::new, timestamp_text),
            modified_age_days: modified
                .and_then(|stamp| now.duration_since(stamp).ok())
                .map(|age| age.as_secs() / DAY_SECONDS),
            opened: last_opened,
            opened_text: timestamp_text(opened),
            size_bytes: facts.as_ref().map(std::fs::Metadata::len),
            size_text: facts.map_or_else(String::new, |facts| format!("{} KB", facts.len() / 1024)),
            pinned,
            design_views: Vec::new(),
        }
    }
}
fn unix_seconds(stamp: std::time::SystemTime) -> u64 {
    stamp
        .duration_since(std::time::UNIX_EPOCH)
        .map_or(0, |since| since.as_secs())
}
fn kind_of_extension(extension: Option<&str>) -> crate::GuiTemplateKind {
    use crate::GuiTemplateKind::{Assembly, Drawing, Part, Presentation};
    match extension {
        Some("oad") => Assembly,
        Some("odd") => Drawing,
        Some("ord") => Presentation,
        _ => Part,
    }
}
