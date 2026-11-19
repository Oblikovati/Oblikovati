// SPDX-License-Identifier: GPL-2.0-only
//! Match and draw filtered model-browser rows.
use super::super::browser_origin;
use super::BrowserView;
use super::{
    GuiBrowserKind, GuiCommandState, GuiModelBrowser, GuiShellAction, ORIGIN_ROWS, Ui, drawing_tree,
};
use super::{HIGHLIGHT, NO_RESULTS, match_start};

/// One line of a search's results: a match, or a row above one, kept so its place can be read.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct ResultRow {
    pub(super) depth: usize,
    pub(super) text: String,
    pub(super) is_match: bool,
    expandable: bool,
    pub(super) origin_index: Option<usize>,
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
        expandable: false,
        origin_index: None,
    });
    results.extend(rows.into_iter().map(|text| ResultRow {
        depth: depth + 1,
        text,
        is_match: true,
        expandable: false,
        origin_index: None,
    }));
}
/// A feature shows where it lives in the tree: under its solid, and among the features.
pub(super) fn matching_rows(browser: &GuiModelBrowser, query: &str) -> Vec<ResultRow> {
    let mut results = Vec::new();
    match_bodies(&mut results, browser, query);
    if browser.has_origin {
        push_matching_branch(&mut results, "Origin", &ORIGIN_ROWS, query, true);
    }
    let sketches: Vec<&str> = browser
        .sketches
        .iter()
        .map(|row| row.name.as_str())
        .collect();
    push_matching_branch(&mut results, "Sketches", &sketches, query, false);
    let features: Vec<&str> = browser
        .features
        .iter()
        .map(|row| row.name.as_str())
        .collect();
    push_matching_branch(&mut results, "Features", &features, query, false);
    results.extend(matching_occurrences(&browser.occurrences, query, 0));
    results
}
fn match_bodies(results: &mut Vec<ResultRow>, browser: &GuiModelBrowser, query: &str) {
    if browser.bodies.is_empty() {
        return;
    }
    if match_start("Solid Bodies", query).is_some() {
        let bodies: Vec<&str> = browser
            .bodies
            .iter()
            .map(|body| body.name.as_str())
            .collect();
        push_matching_branch(results, "Solid Bodies", &bodies, query, false);
        return;
    }
    let bodies = matching_bodies(browser, query);
    if bodies.is_empty() {
        return;
    }
    results.push(ResultRow {
        depth: 0,
        text: "Solid Bodies".into(),
        is_match: false,
        expandable: false,
        origin_index: None,
    });
    for (body, feature) in bodies {
        push_matching_body(results, body, feature, query);
    }
}
fn matching_bodies<'a>(browser: &'a GuiModelBrowser, query: &str) -> Vec<(&'a str, String)> {
    browser
        .bodies
        .iter()
        .filter_map(|body| {
            let feature = browser.features.iter().find(|row| row.id == body.feature)?;
            (match_start(&feature.name, query).is_some()
                || match_start(&body.name, query).is_some())
            .then_some((body.name.as_str(), feature.name.clone()))
        })
        .collect()
}
fn push_matching_body(results: &mut Vec<ResultRow>, body: &str, feature: String, query: &str) {
    if match_start(body, query).is_none() {
        push_branch(results, body, 1, vec![feature]);
        return;
    }
    let start = results.len();
    push_matching_branch(results, body, &[&feature], query, false);
    for row in &mut results[start..] {
        row.depth += 1;
    }
}
fn push_matching_branch(
    results: &mut Vec<ResultRow>,
    branch: &str,
    children: &[&str],
    query: &str,
    origin: bool,
) {
    let branch_matches = match_start(branch, query).is_some();
    let rows: Vec<(usize, &&str)> = children
        .iter()
        .enumerate()
        .filter(|(_, name)| branch_matches || match_start(name, query).is_some())
        .collect();
    if rows.is_empty() && !branch_matches {
        return;
    }
    results.push(ResultRow {
        depth: 0,
        text: branch.into(),
        is_match: branch_matches,
        expandable: branch_matches,
        origin_index: None,
    });
    results.extend(rows.into_iter().map(|(index, text)| ResultRow {
        depth: 1,
        text: (*text).into(),
        is_match: !branch_matches,
        expandable: false,
        origin_index: origin.then_some(index),
    }));
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
                expandable: false,
                origin_index: None,
            });
            results.extend(children);
        }
    }
    results
}
/// A drawing's rows under the branches that hold them; a branch with no match is left out.
pub(super) fn matching_branches(
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
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
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
    draw_matching_rows(ui, state, view, actions, &rows, query);
}
fn draw_matching_rows(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
    rows: &[ResultRow],
    query: &str,
) {
    let _filter = ui.push_id("##filtered-browser-results");
    let mut position = 0;
    while position < rows.len() {
        let row = &rows[position];
        if row.expandable {
            let end = rows[position + 1..]
                .iter()
                .position(|next| next.depth <= row.depth)
                .map_or(rows.len(), |offset| position + 1 + offset);
            draw_parent(
                ui,
                state,
                view,
                actions,
                row,
                &rows[position + 1..end],
                query,
            );
            position = end;
        } else {
            draw_row(ui, state, view, actions, row, query);
            position += 1;
        }
    }
}
fn draw_parent(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
    row: &ResultRow,
    children: &[ResultRow],
    query: &str,
) {
    let open = ui.tree_node_config(&row.text).push();
    let label_offset = ui.current_font_size() + 2. * ui.clone_style().frame_padding[0];
    highlight_item(ui, &row.text, query, label_offset);
    if let Some(_open) = open {
        for child in children {
            draw_row(ui, state, view, actions, child, query);
        }
    }
}
fn draw_row(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
    row: &ResultRow,
    query: &str,
) {
    let indent = row.depth as f32 * ui.current_font_size();
    if indent > 0. {
        ui.indent_by(indent);
    }
    if let Some(index) = row.origin_index {
        draw_origin_result(ui, state, view, actions, row, query, index);
    } else if row.is_match {
        result_row(ui, &row.text, query);
    } else {
        ui.text_disabled(&row.text);
    }
    if indent > 0. {
        ui.unindent_by(indent);
    }
}
fn draw_origin_result(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
    row: &ResultRow,
    query: &str,
    index: usize,
) {
    let color = (!row.is_match).then(|| {
        ui.push_style_color(
            imgui::StyleColor::Text,
            ui.style_color(imgui::StyleColor::TextDisabled),
        )
    });
    browser_origin::origin_row(ui, state, view, index, &row.text, actions);
    drop(color);
    if row.is_match {
        highlight_last_item(ui, &row.text, query);
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
    highlight_last_item(ui, row, query);
}
fn highlight_last_item(ui: &Ui, row: &str, query: &str) {
    highlight_item(ui, row, query, 0.);
}
fn highlight_item(ui: &Ui, row: &str, query: &str, label_offset: f32) {
    let Some(start) = match_start(row, query) else {
        return;
    };
    // Lower-casing can change byte lengths; only a char boundary may slice the row.
    let end = start + query.len();
    let (Some(before), Some(matched)) = (row.get(..start), row.get(start..end)) else {
        return;
    };
    let minimum = ui.item_rect_min();
    let left = minimum[0] + label_offset + ui.calc_text_size(before)[0];
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
