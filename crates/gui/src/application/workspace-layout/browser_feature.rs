// SPDX-License-Identifier: GPL-2.0-only
//! A feature's browser row: its label, its menu, and the outline of its solid over the view.
use super::browser_view::{BrowserBranch, BrowserView, muted_menu_item};
use crate::application::model_browser::{browser_root::find_in_window, relationships::RelatedRow};
use crate::{
    GuiBrowserChoice, GuiBrowserFeature, GuiCommandState, GuiFrameRequest, GuiModelBrowser,
    GuiShellAction,
};
use imgui::{Condition, MouseButton, TreeNodeFlags, Ui};

const FEATURE_MENU: &str = "##feature-menu";
const BODIES_MENU: &str = "##bodies-menu";
const BODY_MENU: &str = "##body-menu";
const EDGE_COLOR: [f32; 4] = [0.25, 0.6, 1., 1.];
const EDGE_THICKNESS: f32 = 2.;
const SELECTION_TOOLBAR: &str = "##selection-toolbar";
const TOOLBAR_RISE: f32 = 40.;

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
    let muted = feature.suppressed.then(|| {
        ui.push_style_color(
            imgui::StyleColor::Text,
            ui.style_color(imgui::StyleColor::TextDisabled),
        )
    });
    let token = node.push();
    drop(muted);
    if feature.suppressed {
        strike_feature_name(ui, &feature.name);
    }
    choose_on_press(ui, state, feature, view, actions);
    feature_menu(ui, state, browser, feature, view, actions);
    token
}
fn strike_feature_name(ui: &Ui, name: &str) {
    let [left, top] = ui.item_rect_min();
    let bottom = ui.item_rect_max()[1];
    let start = left + ui.current_font_size() + 2. * ui.clone_style().frame_padding[0];
    let end = start + ui.calc_text_size(name)[0];
    let height = (top + bottom) * 0.5;
    ui.get_window_draw_list()
        .add_line(
            [start, height],
            [end, height],
            ui.style_color(imgui::StyleColor::TextDisabled),
        )
        .thickness((ui.current_font_size() / 13.).max(1.))
        .build();
}
/// The arrow only folds the branch; the rest of the row chooses the feature.
fn choose_on_press(
    ui: &Ui,
    state: &GuiCommandState,
    feature: &GuiBrowserFeature,
    view: &mut BrowserView,
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
    // Dimensions a row asked for go away the next time its menu is asked for.
    if asks_menu {
        actions.push(GuiShellAction::BrowserChoice {
            document: state.document.unwrap_or(0),
            choice: GuiBrowserChoice::ShowFeatureDimensions(None),
        });
    }
    if clicked || asks_menu {
        view.selected_origin_row = None;
        view.origin_folder_chosen = false;
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
    muted_menu_item(ui, "Edit Feature");
    if ui.menu_item("Show Dimensions") {
        actions.push(GuiShellAction::BrowserChoice {
            document: state.document.unwrap_or(0),
            choice: GuiBrowserChoice::ShowFeatureDimensions(Some(feature.id)),
        });
    }
    ui.separator();
    for row in ["Copy", "Delete", "Measure"] {
        muted_menu_item(ui, row);
    }
    if ui.menu_item("Properties") {
        view.feature_properties.request(feature);
    }
    ui.separator();
    if ui.menu_item("Expand All Children") {
        view.open_branch(BrowserBranch::Feature(feature.id));
    }
    // A feature's direct rows stay; only what sits below them folds, and nothing does yet.
    ui.menu_item("Collapse All Children");
    ui.separator();
    if ui.menu_item("Relationships") {
        view.relationships.request(RelatedRow::Feature(feature.id));
    }
    if ui.menu_item("Find in Window") {
        find_in_window(view, ui, actions);
    }
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
    if browser.selected_body == Some(body.feature) {
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
    let document = state.document.unwrap_or(0);
    if asks_menu || (hovered && ui.is_mouse_clicked(MouseButton::Left)) {
        view.selected_origin_row = None;
        actions.push(GuiShellAction::BrowserChoice {
            document,
            choice: GuiBrowserChoice::Body {
                key: body.key.clone(),
                feature: body.feature,
            },
        });
    }
    if let Some(_menu) = ui.begin_popup(BODY_MENU) {
        if ui.menu_item("Properties") {
            view.body_properties.request(body.feature);
        }
        if ui
            .menu_item_config("Visibility")
            .selected(body.visible)
            .build()
        {
            actions.push(body_visibility(document, body, !body.visible));
        }
    }
    token
}
fn body_visibility(document: u64, body: &crate::GuiBrowserBody, visible: bool) -> GuiShellAction {
    GuiShellAction::BrowserChoice {
        document,
        choice: GuiBrowserChoice::BodyVisible {
            key: body.key.clone(),
            visible,
        },
    }
}
/// A right-click on the Solid Bodies row.
pub(crate) fn bodies_menu(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    if ui.is_item_hovered() && ui.is_mouse_clicked(MouseButton::Right) {
        ui.open_popup(BODIES_MENU);
    }
    let Some(_menu) = ui.begin_popup(BODIES_MENU) else {
        return;
    };
    let all_visible = browser.bodies.iter().all(|body| body.visible);
    if ui
        .menu_item_config("Visibility")
        .selected(all_visible)
        .build()
    {
        let document = state.document.unwrap_or(0);
        for body in &browser.bodies {
            actions.push(body_visibility(document, body, !all_visible));
        }
    }
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
/// A small bar over the chosen feature's solid, with what is most often done to it next.
pub(crate) fn selection_toolbar(
    ui: &Ui,
    request: &GuiFrameRequest,
    actions: &mut Vec<GuiShellAction>,
) {
    let commands = &request.commands;
    if commands.model_face_chosen {
        face_selection_toolbar(ui, request, actions);
        return;
    }
    let browser = commands.browser.as_ref();
    let chosen = browser.and_then(|browser| {
        let feature = browser.selected_feature?;
        browser.features.iter().find(|row| row.id == feature)
    });
    let (Some(browser), Some(feature)) = (browser, chosen) else {
        return;
    };
    let Some(corner) = outline_corner(request) else {
        return;
    };
    selection_toolbar_window(ui, corner, || {
        edit_sketch_row_button(ui, commands, browser, feature, actions);
        ui.same_line();
        let _muted = ui.begin_disabled(true);
        crate::widget_abi::button(ui, "Edit Feature", [0.; 2]);
    });
}
fn face_selection_toolbar(ui: &Ui, request: &GuiFrameRequest, actions: &mut Vec<GuiShellAction>) {
    let Some(corner) = outline_corner(request) else {
        return;
    };
    selection_toolbar_window(ui, corner, || {
        if crate::widget_abi::button(ui, "Zoom Selected", [0.; 2]) {
            actions.push(GuiShellAction::ViewCommand(
                crate::GuiViewCommand::ZoomSelected {
                    extent: request.extent,
                },
            ));
        }
    });
}
fn selection_toolbar_window(ui: &Ui, corner: [f32; 2], draw: impl FnOnce()) {
    ui.window(SELECTION_TOOLBAR)
        .position([corner[0], corner[1] - TOOLBAR_RISE], Condition::Always)
        .always_auto_resize(true)
        .no_decoration()
        .focus_on_appearing(false)
        .save_settings(false)
        .build(draw);
}
fn edit_sketch_row_button(
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
    let _muted = ui.begin_disabled(source.is_none());
    if crate::widget_abi::button(ui, "Edit Sketch", [0.; 2])
        && let Some(source) = source
    {
        actions.push(GuiShellAction::EditSketch {
            document: state.document.unwrap_or(0),
            sketch: source.id,
        });
    }
}
/// The top left of the outline, in layout units.
fn outline_corner(request: &GuiFrameRequest) -> Option<[f32; 2]> {
    let scale = request.layout_scale();
    let points = request.commands.feature_edges.iter().flatten();
    let left = points.clone().map(|pixel| pixel[0]).min()?;
    let top = points.map(|pixel| pixel[1]).min()?;
    Some([left as f32 / scale[0], top as f32 / scale[1]])
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
            kind: "Extrusion",
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
