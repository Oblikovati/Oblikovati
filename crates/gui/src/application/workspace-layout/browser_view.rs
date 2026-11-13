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
const SEARCH_HINT: &str = "Search";
const HIGHLIGHT: [f32; 4] = [0.95, 0.75, 0.2, 0.45];
const LISTED_DISPLAY_PREFERENCES: [&str; 5] = [
    "Hide Notes",
    "Hide Documents",
    "Hide Warnings",
    "Hide Blue Dot Indicator",
    "Auto expand Representations Folder on Open",
];

/// A row that has rows of its own under it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum BrowserBranch {
    Origin,
    Feature(oblikovati_model::feature::FeatureId),
}
#[derive(Debug, Default)]
pub(crate) struct BrowserView {
    autoscroll: bool,
    /// Present while the field is open; an empty query still shows the whole tree.
    query: Option<String>,
    expansion: Option<bool>,
    /// The drawing tree row a click chose, as its branch and row.
    pub selected_row: Option<(usize, usize)>,
    /// The origin row a click chose.
    pub selected_origin_row: Option<usize>,
    pub arrow_consumed: bool,
    pub feature_keyboard_focus: bool,
    pub feature_rows: Vec<crate::application::model_browser::feature_keys::FeatureRow>,
    pub feature_open: std::collections::BTreeMap<oblikovati_model::feature::FeatureId, bool>,
    pub feature_expansion: Option<(oblikovati_model::feature::FeatureId, bool)>,
    pub model_state_edit: crate::browser::browser_model_state::ModelStateEdit,
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
    design_view_expansion: bool,
    pub find: FindDialog,
    pub relationships: RelationshipsDialog,
    pub geometry: GeometryProperties,
    pub feature_properties: FeatureProperties,
}
impl BrowserView {
    /// Draw the menu and search controls above the tree; example: `view.header(ui)`.
    pub(crate) fn header(&mut self, ui: &Ui, drawing: bool) {
        if crate::widget_abi::button(ui, "Menu", [0.; 2]) {
            ui.open_popup(BROWSER_MENU);
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Search", [0.; 2]) {
            self.open_search();
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
    /// Only Hide Work Features has a tree to act on yet; the rest are listed, not offered.
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
            ui.menu_item_config("Design View Expansion")
                .build_with_ref(&mut self.design_view_expansion);
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
    }
    /// The dialogs over the browser; example: each frame, after its tree.
    pub(crate) fn dialogs(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        browser: &GuiModelBrowser,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.relationships.panel(ui, state, browser, actions);
        self.geometry.panel(ui, state, actions);
        self.feature_properties.panel(ui, state, browser, actions);
    }
    /// A row chosen in the Relationships dialog reads as chosen in the tree.
    pub(crate) fn follow_relationships(&mut self) {
        if let Some(row) = self.relationships.take_chosen_origin() {
            self.selected_origin_row = Some(row);
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
    fn open_search(&mut self) {
        self.query.get_or_insert_with(String::new);
    }
    fn search_field(&mut self, ui: &Ui) {
        let Some(query) = &mut self.query else {
            return;
        };
        ui.set_next_item_width(-32.);
        ui.input_text("##browser-search", query)
            .hint(SEARCH_HINT)
            .build();
        ui.same_line();
        if crate::widget_abi::button(ui, "x##browser-search-close", [0.; 2]) {
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
/// One line of a search's results: a match, or a row above one, kept so its place can be read.
#[derive(Clone, Debug, PartialEq, Eq)]
struct ResultRow {
    depth: usize,
    text: String,
    is_match: bool,
}
/// A branch and its matching rows; without one the branch is not shown at all.
fn push_branch(results: &mut Vec<ResultRow>, branch: &str, depth: usize, rows: Vec<String>) {
    if rows.is_empty() {
        return;
    }
    results.push(ResultRow {
        depth,
        text: branch.to_owned(),
        is_match: false,
    });
    results.extend(rows.into_iter().map(|text| ResultRow {
        depth: depth + 1,
        text,
        is_match: true,
    }));
}
/// A feature shows where it lives in the tree: under its solid, and among the features.
fn matching_rows(browser: &GuiModelBrowser, query: &str) -> Vec<ResultRow> {
    let meets = |name: &str| match_start(name, query).is_some();
    let mut results = Vec::new();
    let solids: Vec<_> = browser
        .bodies
        .iter()
        .filter_map(|body| {
            let feature = browser.features.iter().find(|row| row.id == body.feature)?;
            meets(&feature.name).then_some((body.name.as_str(), feature.name.clone()))
        })
        .collect();
    if !solids.is_empty() {
        results.push(ResultRow {
            depth: 0,
            text: "Solid Bodies".to_owned(),
            is_match: false,
        });
        for (body, feature) in solids {
            push_branch(&mut results, body, 1, vec![feature]);
        }
    }
    let origin: &[&str] = if browser.has_origin {
        &ORIGIN_ROWS
    } else {
        &[]
    };
    let kept = |names: &[&str]| -> Vec<String> {
        let met = names.iter().filter(|row| meets(row));
        met.map(|row| (*row).to_owned()).collect()
    };
    push_branch(&mut results, "Origin", 0, kept(origin));
    let sketches: Vec<&str> = browser
        .sketches
        .iter()
        .map(|row| row.name.as_str())
        .collect();
    push_branch(&mut results, "Sketches", 0, kept(&sketches));
    let features: Vec<&str> = browser
        .features
        .iter()
        .map(|row| row.name.as_str())
        .collect();
    push_branch(&mut results, "Features", 0, kept(&features));
    results.extend(matching_occurrences(&browser.occurrences, query, 0));
    results
}
fn matching_occurrences(
    rows: &[crate::GuiBrowserOccurrence],
    query: &str,
    depth: usize,
) -> Vec<ResultRow> {
    let mut results = Vec::new();
    for row in rows {
        let children = matching_occurrences(&row.children, query, depth + 1);
        let is_match = match_start(&row.name, query).is_some();
        if is_match || !children.is_empty() {
            results.push(ResultRow {
                depth,
                text: row.name.clone(),
                is_match,
            });
            results.extend(children);
        }
    }
    results
}
/// A drawing's rows under the branches that hold them; a branch with no match is left out.
fn matching_branches(
    drawing: Option<&oblikovati_model::drawing::DrawingDocument>,
    query: &str,
) -> Vec<drawing_tree::DrawingBranch> {
    drawing_tree::drawing_tree(drawing)
        .into_iter()
        .map(|(branch, rows)| {
            let matches = rows
                .into_iter()
                .filter(|row| match_start(row, query).is_some())
                .collect();
            (branch, matches)
        })
        .filter(|(_, rows): &drawing_tree::DrawingBranch| !rows.is_empty())
        .collect()
}
/// Draw the rows a query leaves, each with its match marked; example: `search_results(ui, browser, "work")`.
pub(crate) fn search_results(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    query: &str,
) {
    if browser.kind == GuiBrowserKind::Drawing {
        let drawing = state
            .document
            .and_then(|id| state.drawing_documents.get(&id));
        drawing_results(ui, drawing, query);
        return;
    }
    let rows = matching_rows(browser, query);
    if rows.is_empty() {
        ui.text_disabled(NO_RESULTS);
        return;
    }
    for row in rows {
        let indent = row.depth as f32 * ui.current_font_size();
        if indent > 0. {
            ui.indent_by(indent);
        }
        if row.is_match {
            result_row(ui, &row.text, query);
        } else {
            ui.text_disabled(&row.text);
        }
        if indent > 0. {
            ui.unindent_by(indent);
        }
    }
}
fn drawing_results(
    ui: &Ui,
    drawing: Option<&oblikovati_model::drawing::DrawingDocument>,
    query: &str,
) {
    let branches = matching_branches(drawing, query);
    if branches.is_empty() {
        ui.text_disabled(NO_RESULTS);
        return;
    }
    for (branch, rows) in branches {
        ui.text(branch);
        ui.indent();
        for row in rows {
            result_row(ui, &row, query);
        }
        ui.unindent();
    }
}
fn result_row(ui: &Ui, row: &str, query: &str) {
    ui.text(row);
    let Some(start) = match_start(row, query) else {
        return;
    };
    // Lower-casing can change byte lengths; only a char boundary may slice the row.
    let end = start + query.len();
    let (Some(before), Some(matched)) = (row.get(..start), row.get(start..end)) else {
        return;
    };
    let minimum = ui.item_rect_min();
    let left = minimum[0] + ui.calc_text_size(before)[0];
    let right = left + ui.calc_text_size(matched)[0];
    ui.get_window_draw_list()
        .add_rect(
            [left, minimum[1]],
            [right, ui.item_rect_max()[1]],
            HIGHLIGHT,
        )
        .filled(true)
        .build();
}
#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/browser_view.rs"]
mod tests;
