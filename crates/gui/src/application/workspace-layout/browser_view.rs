// SPDX-License-Identifier: GPL-2.0-only
//! The model browser's own commands: expanding its tree and searching its rows.
use crate::browser::{DRAWING_TREE, GuiBrowserKind, GuiModelBrowser, ORIGIN_ROWS};
use imgui::Ui;

const BROWSER_MENU: &str = "##browser-menu";
const NO_RESULTS: &str = "No matching entries";
const SEARCH_HINT: &str = "Search";
const HIGHLIGHT: [f32; 4] = [0.95, 0.75, 0.2, 0.45];
const LISTED_DISPLAY_PREFERENCES: [&str; 6] = [
    "Hide Notes",
    "Hide Documents",
    "Hide Warnings",
    "Hide Blue Dot Indicator",
    "Auto expand Representations Folder on Open",
    "Design View Expansion",
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
    pub hide_work_features: bool,
    /// Feature rows also say their operation and depth.
    pub show_extended_names: bool,
    pub body_properties: super::body_properties::BodyPropertiesDialog,
    /// The branch a row's Expand All Children opens on the next frame.
    opened_branch: Option<BrowserBranch>,
    /// What the Solid Bodies menu asked of everything under the folder.
    bodies_expansion: Option<bool>,
    /// The branch was just brought back: it opens closed, once.
    pub origin_closes_on_return: bool,
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
                self.open_search();
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
        });
    }
    /// Open or fold everything under Solid Bodies on the next frame; example: from its menu.
    pub(crate) fn set_bodies_expansion(&mut self, open: bool) {
        self.bodies_expansion = Some(open);
    }
    pub(crate) fn take_bodies_expansion(&mut self) -> Option<bool> {
        self.bodies_expansion.take()
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
/// Every row a search can reach, in tree order.
fn searchable_rows(browser: &GuiModelBrowser) -> Vec<&str> {
    let origin: &[&str] = if browser.has_origin {
        &ORIGIN_ROWS
    } else {
        &[]
    };
    let sketches = browser.sketches.iter().map(|row| row.name.as_str());
    let features = browser.features.iter().map(|row| row.name.as_str());
    origin
        .iter()
        .copied()
        .chain(sketches)
        .chain(features)
        .collect()
}
/// Where the query starts in a row, ignoring case; its length in bytes is the query's.
pub(crate) fn match_start(row: &str, query: &str) -> Option<usize> {
    row.to_lowercase().find(&query.to_lowercase())
}
fn matching_rows<'a>(browser: &'a GuiModelBrowser, query: &str) -> Vec<&'a str> {
    let rows = searchable_rows(browser);
    rows.into_iter()
        .filter(|row| match_start(row, query).is_some())
        .collect()
}
/// A drawing's rows under the branches that hold them; a branch with no match is left out.
fn matching_branches(query: &str) -> Vec<(&'static str, Vec<&'static str>)> {
    DRAWING_TREE
        .iter()
        .map(|(branch, rows)| {
            let matches = rows
                .iter()
                .copied()
                .filter(|row| match_start(row, query).is_some())
                .collect();
            (*branch, matches)
        })
        .filter(|(_, rows): &(&str, Vec<&str>)| !rows.is_empty())
        .collect()
}
/// Draw the rows a query leaves, each with its match marked; example: `search_results(ui, browser, "work")`.
pub(crate) fn search_results(ui: &Ui, browser: &GuiModelBrowser, query: &str) {
    if browser.kind == GuiBrowserKind::Drawing {
        drawing_results(ui, query);
        return;
    }
    let rows = matching_rows(browser, query);
    if rows.is_empty() {
        ui.text_disabled(NO_RESULTS);
        return;
    }
    for row in rows {
        result_row(ui, row, query);
    }
}
fn drawing_results(ui: &Ui, query: &str) {
    let branches = matching_branches(query);
    if branches.is_empty() {
        ui.text_disabled(NO_RESULTS);
        return;
    }
    for (branch, rows) in branches {
        ui.text(branch);
        ui.indent();
        for row in rows {
            result_row(ui, row, query);
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
