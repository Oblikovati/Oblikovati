// SPDX-License-Identifier: GPL-2.0-only
//! The model browser's own commands: expanding its tree and searching its rows.
use crate::application::model_browser::{
    feature_properties::FeatureProperties, find_dialog::FindDialog,
    geometry_properties::GeometryProperties, relationships::RelationshipsDialog,
};
use crate::browser::{GuiBrowserKind, GuiModelBrowser, ORIGIN_ROWS, drawing_tree};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

const BROWSER_MENU: &str = "##browser-menu";
const NO_RESULTS: &str = "No matching entries";
const SEARCH_HINT: &str = "Type Keyword";
const HIGHLIGHT: [f32; 4] = [0.95, 0.75, 0.2, 0.45];
const LISTED_DISPLAY_PREFERENCES: [&str; 4] = [
    "Hide Notes",
    "Hide Documents",
    "Hide Warnings",
    "Hide Blue Dot Indicator",
];

/// A row that has rows of its own under it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum BrowserBranch {
    Origin,
    Feature(oblikovati_model::feature::FeatureId),
}
#[derive(Debug, Default)]
pub(crate) struct BrowserView {
    #[cfg(test)]
    pub(crate) test_sketch_bounds: std::collections::BTreeMap<u64, [[f32; 2]; 2]>,
    #[cfg(test)]
    pub(crate) test_edit_bounds: Option<[[f32; 2]; 2]>,
    #[cfg(test)]
    pub(crate) test_visibility_bounds: Option<[[f32; 2]; 2]>,
    pub(crate) icons: Option<crate::icons::GuiRibbonIcons>,
    pub(super) menu_requested: bool,
    autoscroll: bool,
    /// Present while the field is open; an empty query still shows the whole tree.
    query: Option<String>,
    pub(super) focus_search: bool,
    expansion: Option<bool>,
    /// The drawing tree row a click chose, as its branch and row.
    pub selected_row: Option<(usize, usize)>,
    /// The origin row a click chose.
    pub selected_origin_row: Option<usize>,
    pub(super) picked_plane_reference: Option<String>,
    /// The last origin row retains a focus outline after viewport selection clears.
    pub focused_origin_row: Option<usize>,
    pub arrow_consumed: bool,
    pub feature_keyboard_focus: bool,
    pub feature_rows: Vec<crate::application::model_browser::feature_keys::FeatureRow>,
    pub feature_open: std::collections::BTreeMap<oblikovati_model::feature::FeatureId, bool>,
    pub feature_expansion: Option<(oblikovati_model::feature::FeatureId, bool)>,
    pub model_state_edit: crate::browser::browser_model_state::ModelStateEdit,
    pub part_view_edit: crate::browser::browser_part_view_edit::PartViewEdit,
    pub part_view_scale: crate::browser::browser_part_view_scale::PartViewScaleDialog,
    pub hide_work_features: bool,
    /// Feature rows also say their operation and depth.
    pub show_extended_names: bool,
    pub body_properties: super::body_properties::BodyPropertiesDialog,
    /// A navigation command owns the next click on the view.
    pub view_clicks_taken: bool,
    /// The screen left for the document, the only place a click means the model.
    pub document_area: Option<crate::application::document_creation::workspaces::WorkspaceArea>,
    /// Framebuffer pixels per layout unit, for what a click reports.
    pub layout_scale: [f32; 2],
    /// The branch a row's Expand All Children opens on the next frame.
    opened_branch: Option<BrowserBranch>,
    /// What the Solid Bodies menu asked of everything under the folder.
    bodies_expansion: Option<bool>,
    /// The branch was just brought back: it opens closed, once.
    pub origin_closes_on_return: bool,
    /// Origin's own row is chosen, by the arrow keys.
    pub origin_folder_chosen: bool,
    /// The Origin branch shows its rows, as of the last drawn tree.
    pub origin_open: bool,
    /// What the arrow keys asked of the branch, for the next tree.
    origin_open_request: Option<bool>,
    /// Origin rows made visible, by document and row: view state, as a body's visibility is.
    pub visible_origin_rows: std::collections::BTreeSet<(u64, usize)>,
    /// Explicit non-automatic datum displays, scoped to each open document.
    origin_fixed_size: std::collections::BTreeSet<(u64, usize)>,
    design_view_expansion: bool,
    auto_expand_representations: AutoExpandRepresentations,
    pub find: FindDialog,
    pub relationships: RelationshipsDialog,
    pub geometry: GeometryProperties,
    pub feature_properties: FeatureProperties,
}
#[derive(Debug)]
struct AutoExpandRepresentations(bool);
impl Default for AutoExpandRepresentations {
    fn default() -> Self {
        Self(true)
    }
}
impl BrowserView {
    /// Draw the menu and search controls above the tree; example: `view.header(ui)`.
    pub(crate) fn header(&mut self, ui: &Ui, drawing: bool) {
        if std::mem::take(&mut self.menu_requested) {
            ui.open_popup(BROWSER_MENU);
        }
        if !crate::widget_abi::window_is_docked(ui) {
            if crate::widget_abi::button(ui, "Menu", [0.; 2]) {
                ui.open_popup(BROWSER_MENU);
            }
            ui.same_line();
            if crate::widget_abi::button(ui, "Search", [0.; 2]) {
                self.open_search();
            }
        }
        self.menu(ui, drawing);
        self.search_field(ui);
    }
    fn menu(&mut self, ui: &Ui, drawing: bool) {
        let Some(_menu) = ui.begin_popup(BROWSER_MENU) else {
            return;
        };
        if !drawing {
            // The choice is kept for the tree's selection sync, which scrolls only when it is on.
            ui.checkbox("Autoscroll", &mut self.autoscroll);
        }
        if ui.menu_item("Expand All") {
            self.expansion = Some(true);
        }
        if ui.menu_item("Collapse All") {
            self.expansion = Some(false);
        }
        if !drawing {
            if ui.menu_item("Find") {
                self.find.request();
            }
            self.display_preferences(ui);
        }
        muted_menu_item(ui, "Help");
    }
    /// Display preferences that control the browser tree; unsupported choices stay unavailable.
    fn display_preferences(&mut self, ui: &Ui) {
        ui.menu("Display Preferences", || {
            muted_menu_item(ui, "Hide UCS");
            let hidden = self.hide_work_features;
            if ui
                .menu_item_config("Hide Work Features")
                .selected(hidden)
                .build()
            {
                self.hide_work_features = !hidden;
                self.origin_closes_on_return = hidden;
            }
            let [notes, documents, warnings, rest @ ..] = LISTED_DISPLAY_PREFERENCES;
            for row in [notes, documents, warnings] {
                muted_menu_item(ui, row);
            }
            ui.menu_item_config("Show Extended Names")
                .build_with_ref(&mut self.show_extended_names);
            for row in rest {
                muted_menu_item(ui, row);
            }
            ui.menu_item_config("Auto expand Representations Folder on Open")
                .build_with_ref(&mut self.auto_expand_representations.0);
            ui.menu_item_config("Design View Expansion")
                .build_with_ref(&mut self.design_view_expansion);
        });
    }
    /// Read the initial representation-tree preference; example: `view.auto_expand_representations()`.
    pub(crate) fn auto_expand_representations(&self) -> bool {
        self.auto_expand_representations.0
    }
    /// Restore the saved representation-tree preference; example: `view.set_auto_expand_representations(false)`.
    pub(crate) fn set_auto_expand_representations(&mut self, enabled: bool) {
        self.auto_expand_representations.0 = enabled;
    }
    /// Read whether an origin datum follows scene bounds; example: `view.origin_auto_resize(1, 0)`.
    pub(crate) fn origin_auto_resize(&self, document: u64, row: usize) -> bool {
        !self.origin_fixed_size.contains(&(document, row))
    }
    /// Toggle an origin datum independently; example: `view.toggle_origin_auto_resize(1, 0)`.
    pub(crate) fn toggle_origin_auto_resize(&mut self, document: u64, row: usize) {
        let key = (document, row);
        if !self.origin_fixed_size.remove(&key) {
            self.origin_fixed_size.insert(key);
        }
    }
    /// Release overrides for documents that have closed; example: `view.prune_origin_sizes(&state)`.
    pub(crate) fn prune_origin_sizes(&mut self, state: &GuiCommandState) {
        self.origin_fixed_size.retain(|(document, _)| {
            state.document == Some(*document)
                || state.documents.iter().any(|row| row.id == *document)
        });
    }
    /// Open or fold the whole tree on the next frame; example: the root row's commands.
    pub(crate) fn set_expansion(&mut self, open: bool) {
        self.expansion = Some(open);
    }
    /// A frame must follow so a dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.feature_expansion.is_some()
            || self.body_properties.needs_frame()
            || self.find.needs_frame()
            || self.geometry.needs_frame()
            || self.feature_properties.needs_frame()
            || self.part_view_scale.needs_frame()
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState) {
        let suspended = state.browser.is_none();
        self.body_properties.sync_before_docking(state, suspended);
        self.geometry.sync_before_docking(state, suspended);
        self.feature_properties
            .sync_before_docking(state, suspended);
        self.find.set_suspended(suspended);
        self.relationships.sync_before_docking(state, suspended);
        self.part_view_scale.sync_before_docking(state, suspended);
    }
    /// Draw browser-owned windows independently of the current tree filter.
    pub(crate) fn dialogs(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.body_properties.panel(ui, state, browser, actions);
        self.relationships.panel(ui, state, browser, actions);
        self.geometry.panel(ui, state, actions);
        self.feature_properties.panel(ui, state, browser, actions);
        self.part_view_scale.panel(ui, state.document, actions);
    }
    /// A row chosen in the Relationships dialog reads as chosen in the tree.
    pub(crate) fn follow_relationships(&mut self) {
        if let Some(row) = self.relationships.take_chosen_origin() {
            self.selected_origin_row = Some(row);
            self.focused_origin_row = Some(row);
        }
    }
    pub(crate) fn set_origin_open(&mut self, open: Option<bool>) {
        if open.is_some() {
            self.origin_open_request = open;
        }
    }
    pub(crate) fn take_origin_open(&mut self) -> Option<bool> {
        self.origin_open_request.take()
    }
    /// Open or fold everything under Solid Bodies on the next frame; example: from its menu.
    pub(crate) fn set_bodies_expansion(&mut self, open: bool) {
        self.bodies_expansion = Some(open);
    }
    pub(crate) fn take_bodies_expansion(&mut self) -> Option<bool> {
        self.bodies_expansion.take()
    }
    pub(crate) fn take_feature_expansion(
        &mut self,
        feature: oblikovati_model::feature::FeatureId,
    ) -> Option<bool> {
        if self.feature_expansion.is_some_and(|(id, _)| id == feature) {
            return self.feature_expansion.take().map(|(_, open)| open);
        }
        None
    }
    /// Open one branch on the next frame; example: from its menu's Expand All Children.
    pub(crate) fn open_branch(&mut self, branch: BrowserBranch) {
        self.opened_branch = Some(branch);
    }
    /// This frame's tree opens the branch, once: later a user may close it again.
    pub(crate) fn take_opened_branch(&mut self, branch: BrowserBranch) -> bool {
        let opened = self.opened_branch == Some(branch);
        if opened {
            self.opened_branch = None;
        }
        opened
    }
    pub(crate) fn open_search(&mut self) {
        self.query.get_or_insert_with(String::new);
        self.focus_search = true;
    }
    fn search_field(&mut self, ui: &Ui) {
        let Some(query) = &mut self.query else {
            return;
        };
        use imgui::{StyleColor, StyleVar};
        let [left, top] = ui.cursor_screen_pos();
        let width = ui.content_region_avail()[0];
        let height = ui.frame_height();
        let clear_width = 24.;
        let bounds = [[left, top], [left + width, top + height]];
        let draw = ui.get_window_draw_list();
        draw.add_rect(bounds[0], bounds[1], ui.style_color(StyleColor::FrameBg))
            .rounding(4.)
            .filled(true)
            .build();
        let _border = ui.push_style_var(StyleVar::FrameBorderSize(0.));
        ui.set_next_item_width((width - clear_width).max(1.));
        let field = ui.push_style_color(StyleColor::FrameBg, [0.; 4]);
        let hint = ui.push_style_color(StyleColor::TextDisabled, ui.style_color(StyleColor::Text));
        if std::mem::take(&mut self.focus_search) {
            ui.set_keyboard_focus_here();
        }
        ui.input_text("##browser-search", query)
            .hint(SEARCH_HINT)
            .build();
        let focused = ui.is_item_active();
        drop(hint);
        drop(field);
        ui.same_line_with_spacing(0., 0.);
        let background = ui.push_style_color(StyleColor::Button, [0.; 4]);
        let clear = crate::widget_abi::button(ui, "×##browser-search-close", [clear_width, height]);
        drop(background);
        if ui.is_item_hovered() {
            ui.tooltip_text(if query.is_empty() {
                "Close search"
            } else {
                "Clear search"
            });
        }
        let outline = if focused {
            StyleColor::NavHighlight
        } else {
            StyleColor::Border
        };
        draw.add_rect(bounds[0], bounds[1], ui.style_color(outline))
            .rounding(4.)
            .build();
        if clear {
            self.close_or_clear();
        }
    }
    /// The first press clears a query; a press on an empty field closes it.
    fn close_or_clear(&mut self) {
        match &mut self.query {
            Some(query) if !query.is_empty() => query.clear(),
            _ => self.query = None,
        }
    }
    /// The text to search for, when there is one.
    pub(crate) fn query(&self) -> Option<&str> {
        self.query.as_deref().filter(|query| !query.is_empty())
    }
    /// The state for this frame's tree, once: a later frame lets a user open branches again.
    pub(crate) fn take_expansion(&mut self) -> Option<bool> {
        self.expansion.take()
    }
}
pub(crate) fn muted_menu_item(ui: &Ui, label: &str) {
    ui.menu_item_config(label).enabled(false).build();
    if ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED) {
        ui.tooltip_text("This command is not available in this build.");
    }
}
/// Where the query starts in a row, ignoring case; its length in bytes is the query's.
pub(crate) fn match_start(row: &str, query: &str) -> Option<usize> {
    row.to_lowercase().find(&query.to_lowercase())
}
#[path = "browser_search.rs"]
mod search;
pub(crate) use search::search_results;
#[cfg(test)]
use search::{matching_branches, matching_rows};

#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/browser_view.rs"]
mod tests;
