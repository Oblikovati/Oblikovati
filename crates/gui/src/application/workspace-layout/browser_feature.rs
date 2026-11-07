// SPDX-License-Identifier: GPL-2.0-only
//! A feature's browser row: its label, its menu, and the outline of its solid over the view.
use super::browser_view::{BrowserBranch, BrowserView, muted_menu_item};
use crate::{GuiBrowserFeature, GuiCommandState, GuiFrameRequest, GuiModelBrowser, GuiShellAction};
use imgui::{Condition, MouseButton, TreeNodeFlags, Ui};

const FEATURE_MENU: &str = "##feature-menu";
const BODIES_MENU: &str = "##bodies-menu";
const BODY_MENU: &str = "##body-menu";
const EDGE_COLOR: [f32; 4] = [0.25, 0.6, 1., 1.];
const EDGE_THICKNESS: f32 = 2.;

/// Show Extended Names adds what the row's feature does; a suppressed feature says so.
fn feature_label(feature: &GuiBrowserFeature, extended_names: bool) -> String {
    let mut label = feature.name.clone();
    if feature.suppressed {
        label.push_str(" (suppressed)");
    }
    if let (true, Some(extended)) = (extended_names, &feature.extended) {
        label.push_str(&format!(" ({extended})"));
    }
    label
}
/// Draw one feature row; a press on it, with either button, chooses the feature.
/// The token holds the branch open for its children; example: the sketch it consumed.
pub(crate) fn feature_row<'ui>(
    ui: &'ui Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    expansion: Option<bool>,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) -> Option<imgui::TreeNodeToken<'ui>> {
    let mut flags = TreeNodeFlags::OPEN_ON_ARROW | TreeNodeFlags::SPAN_AVAIL_WIDTH;
    if browser.selected_feature == Some(feature.id) {
        flags |= TreeNodeFlags::SELECTED;
    }
    let node = ui
        .tree_node_config("feature-row")
        .label::<String, String>(feature_label(feature, view.show_extended_names))
        .flags(flags)
        .default_open(true);
    let opened = view.take_opened_branch(BrowserBranch::Feature(feature.id));
    let node = match (opened, expansion) {
        (true, _) => node.opened(true, Condition::Always),
        (false, Some(open)) => node.opened(open, Condition::Always),
        (false, None) => node,
    };
    let token = node.push();
    choose_on_press(ui, state, feature, actions);
    feature_menu(ui, state, browser, feature, view, actions);
    token
}
/// The arrow only folds the branch; the rest of the row chooses the feature.
fn choose_on_press(
    ui: &Ui,
    state: &GuiCommandState,
    feature: &GuiBrowserFeature,
    actions: &mut Vec<GuiShellAction>,
) {
    let hovered = ui.is_item_hovered();
    let asks_menu = hovered && ui.is_mouse_clicked(MouseButton::Right);
    if asks_menu {
        ui.open_popup(FEATURE_MENU);
    }
    let [row_left, _] = ui.item_rect_min();
    let past_arrow = ui.io().mouse_pos[0] > row_left + ui.frame_height();
    let clicked = hovered && past_arrow && ui.is_mouse_clicked(MouseButton::Left);
    if clicked || asks_menu {
        actions.push(GuiShellAction::SelectBrowserFeature {
            document: state.document.unwrap_or(0),
            feature: feature.id,
        });
    }
}
fn feature_menu(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(_menu) = ui.begin_popup(FEATURE_MENU) else {
        return;
    };
    edit_sketch_row(ui, state, browser, feature, actions);
    for row in ["Edit Feature", "Show Dimensions"] {
        muted_menu_item(ui, row);
    }
    ui.separator();
    for row in ["Copy", "Delete", "Measure", "Properties"] {
        muted_menu_item(ui, row);
    }
    ui.separator();
    if ui.menu_item("Expand All Children") {
        view.open_branch(BrowserBranch::Feature(feature.id));
    }
    // A feature's direct rows stay; only what sits below them folds, and nothing does yet.
    ui.menu_item("Collapse All Children");
}
/// One solid's row; a press on it chooses the feature that made it, which outlines the solid.
pub(crate) fn body_row<'ui>(
    ui: &'ui Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    body: &crate::GuiBrowserBody,
    expansion: Option<bool>,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) -> Option<imgui::TreeNodeToken<'ui>> {
    let mut flags = TreeNodeFlags::OPEN_ON_ARROW | TreeNodeFlags::SPAN_AVAIL_WIDTH;
    if browser.selected_feature == Some(body.feature) {
        flags |= TreeNodeFlags::SELECTED;
    }
    let node = ui.tree_node_config(&body.name).flags(flags);
    let node = match expansion {
        Some(open) => node.opened(open, Condition::Always),
        None => node,
    };
    let token = node.push();
    let hovered = ui.is_item_hovered();
    let asks_menu = hovered && ui.is_mouse_clicked(MouseButton::Right);
    if asks_menu {
        ui.open_popup(BODY_MENU);
    }
    if asks_menu || (hovered && ui.is_mouse_clicked(MouseButton::Left)) {
        actions.push(GuiShellAction::SelectBrowserFeature {
            document: state.document.unwrap_or(0),
            feature: body.feature,
        });
    }
    if let Some(_menu) = ui.begin_popup(BODY_MENU) {
        if ui.menu_item("Properties") {
            view.body_properties.request(body.feature);
        }
        muted_menu_item(ui, "Visibility");
    }
    token
}
/// A right-click on the Solid Bodies row.
pub(crate) fn bodies_menu(ui: &Ui, view: &mut BrowserView) {
    if ui.is_item_hovered() && ui.is_mouse_clicked(MouseButton::Right) {
        ui.open_popup(BODIES_MENU);
    }
    let Some(_menu) = ui.begin_popup(BODIES_MENU) else {
        return;
    };
    muted_menu_item(ui, "Visibility");
    ui.separator();
    if ui.menu_item("Expand All Children") {
        view.set_bodies_expansion(true);
    }
    if ui.menu_item("Collapse All Children") {
        view.set_bodies_expansion(false);
    }
}
/// Edit Sketch opens the sketch the feature consumed, when there is one to open.
fn edit_sketch_row(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    actions: &mut Vec<GuiShellAction>,
) {
    let source = browser
        .sketches
        .iter()
        .find(|row| row.consumed_by == Some(feature.id));
    let chosen = ui
        .menu_item_config("Edit Sketch")
        .enabled(source.is_some())
        .build();
    if let (true, Some(source)) = (chosen, source) {
        actions.push(GuiShellAction::EditSketch {
            document: state.document.unwrap_or(0),
            sketch: source.id,
        });
    }
}
/// Outline the chosen feature's solid in the accent colour, under every panel.
pub(crate) fn draw_feature_edges(ui: &Ui, request: &GuiFrameRequest) {
    let scale = request.layout_scale();
    let logical = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
    let list = ui.get_background_draw_list();
    for [from, to] in &request.commands.feature_edges {
        list.add_line(logical(*from), logical(*to), EDGE_COLOR)
            .thickness(EDGE_THICKNESS)
            .build();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::feature::FeatureId;
    fn extrude(suppressed: bool) -> GuiBrowserFeature {
        GuiBrowserFeature {
            id: FeatureId(1),
            name: "Extrude1".into(),
            suppressed,
            extended: Some("New Solid, 20 mm".into()),
        }
    }
    #[test]
    fn a_row_says_its_operation_and_depth_only_when_extended_names_are_on() {
        assert_eq!(feature_label(&extrude(false), false), "Extrude1");
        assert_eq!(
            feature_label(&extrude(false), true),
            "Extrude1 (New Solid, 20 mm)"
        );
        assert_eq!(
            feature_label(&extrude(true), false),
            "Extrude1 (suppressed)"
        );
    }
}
