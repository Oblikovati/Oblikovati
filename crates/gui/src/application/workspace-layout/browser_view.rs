// SPDX-License-Identifier: GPL-2.0-only
//! The model browser's own commands: expanding its tree and searching its rows.
use crate::browser::{DRAWING_TREE, GuiBrowserKind, GuiModelBrowser, ORIGIN_ROWS};
use imgui::Ui;

const BROWSER_MENU: &str = "##browser-menu";
const NO_RESULTS: &str = "No matching entries";
const SEARCH_HINT: &str = "Search";
const HIGHLIGHT: [f32; 4] = [0.95, 0.75, 0.2, 0.45];

#[derive(Debug, Default)]
pub(crate) struct BrowserView {
    autoscroll: bool,
    /// Present while the field is open; an empty query still shows the whole tree.
    query: Option<String>,
    expansion: Option<bool>,
    /// The drawing tree row a click chose, as its branch and row.
    pub selected_row: Option<(usize, usize)>,
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
            muted_menu_item(ui, "Display Preferences");
        }
        muted_menu_item(ui, "Help");
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
fn muted_menu_item(ui: &Ui, label: &str) {
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
