// SPDX-License-Identifier: GPL-2.0-only
//! The Home workspace: starting points on the left, recent documents on the right.
#[path = "home_layout.rs"]
mod layout;
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
const SIDE_PANEL_WIDTH: f32 = 240.;
const RECENT_SCROLL_END_PADDING: f32 = 12.;
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
    observed_document: Option<bool>,
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
    pub open_projects: bool,
    pub file_properties: Option<String>,
}
impl HomeWorkspace {
    /// Follow entry and exit of the no-document workspace. On launch, `observe_document(None)` opens Home.
    pub(crate) fn observe_document(&mut self, document: Option<u64>) -> bool {
        let has_document = document.is_some();
        let previous = self.observed_document.replace(has_document);
        if previous != Some(has_document) {
            self.active = !has_document;
        }
        previous.is_none() && self.active
    }

    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.open_options.register_dockable(registry)
    }
    pub(crate) fn sync_before_docking(&mut self) {
        if !self.active {
            self.open_options.close();
        }
    }
    fn recent_area(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        icons: &crate::icons::GuiRibbonIcons,
    ) {
        let _outline = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
        self.recent_toolbar(ui, icons);
        self.search_history(ui);
        let rows = self.shown_rows(state);
        self.checks.keep_only(&rows);
        self.selected_actions(ui, actions);
        if state.recent_documents.is_empty() {
            if !self.grid {
                self.list(ui, &rows, actions, icons);
            }
            ui.text_disabled("No recent documents");
        } else if rows.is_empty() {
            self.no_results(ui);
        } else if self.grid {
            self.sort_controls(ui);
            self.cards(ui, &rows, actions);
        } else {
            self.list(ui, &rows, actions, icons);
        }
        if !rows.is_empty() {
            // Leave breathing room above the document strip when the last row scrolls into view.
            ui.dummy([0., RECENT_SCROLL_END_PADDING]);
        }
    }
    fn search_controls(&mut self, ui: &Ui, icons: &crate::icons::GuiRibbonIcons) {
        use crate::icons::{IconGlyph, IconSize};
        use imgui::{StyleColor, StyleVar};
        let width = ui.content_region_avail()[0].min(320.);
        let height = ui.frame_height();
        let clear_width = 24.;
        let search_width = ui.calc_text_size("Search")[0] + 30.;
        let [left, top] = ui.cursor_screen_pos();
        let end = [left + width, top + height];
        let background = ui.style_color(StyleColor::FrameBg);
        ui.get_window_draw_list()
            .add_rect([left, top], end, background)
            .rounding(4.)
            .filled(true)
            .build();
        let _border = ui.push_style_var(StyleVar::FrameBorderSize(0.));
        let field = ui.push_style_color(StyleColor::FrameBg, [0.; 4]);
        ui.set_next_item_width((width - clear_width - search_width).max(1.));
        ui.input_text("##search-recent", &mut self.query)
            .hint("Search recent")
            .build();
        drop(field);
        let focused = ui.is_item_active();
        if ui.is_item_activated() && self.query.is_empty() {
            self.show_history = true;
        }
        if !self.query.is_empty() && focused {
            self.show_history = false;
        }
        let _background = ui.push_style_color(StyleColor::Button, [0.; 4]);
        ui.same_line_with_spacing(0., 0.);
        if ui.button_with_size("×##clear-search", [clear_width, height]) {
            self.query.clear();
            self.show_history = true;
        }
        if ui.is_item_hovered() {
            ui.tooltip_text("Clear search");
        }
        ui.same_line_with_spacing(0., 0.);
        if crate::widget_abi::icon_button(
            ui,
            "Search",
            crate::widget_abi::IconButtonPresentation {
                texture: icons.key(IconGlyph::Find, IconSize::Small),
                size: [search_width, height],
                icon: [14.; 2],
                large: false,
                show_caption: true,
            },
        ) {
            self.apply_search();
        }
        let outline = if focused {
            StyleColor::NavHighlight
        } else {
            StyleColor::Border
        };
        ui.get_window_draw_list()
            .add_rect([left, top], end, ui.style_color(outline))
            .rounding(4.)
            .build();
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
            // The clock mark says this is a past search, not a result.
            if ui.selectable(format!("(h) {term}###history-{term}")) {
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
        crate::widget_abi::window_geometry(
            ui,
            position,
            [SIDE_PANEL_WIDTH, 0.],
            imgui::Condition::Always,
        );
        ui.window("##home-side-panel")
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

#[cfg(test)]
mod dock_tests {
    use super::*;
    #[test]
    fn home_follows_entry_and_exit_of_the_no_document_workspace() {
        let mut home = HomeWorkspace::default();
        assert!(home.observe_document(None));
        assert!(home.active);
        assert!(!home.observe_document(Some(1)));
        assert!(!home.active);
        home.active = true;
        assert!(!home.observe_document(Some(1)));
        assert!(home.active);
        assert!(!home.observe_document(None));
        assert!(home.active);
    }

    #[test]
    fn leaving_home_closes_open_options_before_docking() {
        let mut home = HomeWorkspace::default();
        let mut registry = crate::dockable_lifecycle::DockRegistry::default();
        home.register_dockables(&mut registry).unwrap();
        home.open_options.request(&GuiRecentDocument {
            path: "part.opd".into(),
            name: "part".into(),
            location: "".into(),
            kind: crate::GuiTemplateKind::Part,
            modified: None,
            modified_text: "".into(),
            modified_age_days: None,
            opened: 0,
            opened_text: "".into(),
            size_bytes: None,
            size_text: "".into(),
            pinned: false,
            design_views: Vec::new(),
        });
        registry.reconcile(|_| {}).unwrap();
        home.sync_before_docking();
        let mut closed = Vec::new();
        registry
            .reconcile(|title| closed.push(title.to_string_lossy().into_owned()))
            .unwrap();
        assert_eq!(closed.len(), 1);
    }
}
