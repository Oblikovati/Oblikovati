// SPDX-License-Identifier: GPL-2.0-only
//! A feature's browser row: its label, its menu, and the outline of its solid over the view.
use super::browser_view::{BrowserBranch, BrowserView, muted_menu_item};
use crate::application::model_browser::{browser_root::find_in_window, relationships::RelatedRow};
use crate::{
    GuiBrowserChoice, GuiBrowserFeature, GuiCommandState, GuiFrameRequest, GuiModelBrowser,
    GuiShellAction,
};
use imgui::{Condition, MouseButton, TreeNodeFlags, Ui};

#[path = "face_toolbar_placement.rs"]
mod face_toolbar_placement;
#[path = "browser_feature_states.rs"]
mod states;

#[path = "browser_feature_edit.rs"]
mod edit;
use edit::feature_edit_action;

const FEATURE_MENU: &str = "##feature-menu";
const BODIES_MENU: &str = "##bodies-menu";
const BODY_MENU: &str = "##body-menu";
const EDGE_COLOR: [f32; 4] = [45. / 255., 212. / 255., 191. / 255., 1.];
const EDGE_HOVER_COLOR: [f32; 4] = [94. / 255., 234. / 255., 212. / 255., 1.];
const EDGE_HOVER_REACH: f32 = 8.;
const EDGE_THICKNESS: f32 = 2.;
const SELECTION_TOOLBAR: &str = "##selection-toolbar";
const TOOLBAR_RISE: f32 = 40.;
#[cfg(test)]
fn feature_label(feature: &GuiBrowserFeature, extended_names: bool) -> String {
    crate::browser::browser_import::plain_feature_label(feature, extended_names)
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
    // The States control shares the full-width row; it must receive its own click (#4128).
    let mut flags = TreeNodeFlags::OPEN_ON_ARROW
        | TreeNodeFlags::SPAN_AVAIL_WIDTH
        | TreeNodeFlags::ALLOW_ITEM_OVERLAP;
    if browser.selected_feature == Some(feature.id) {
        flags |= TreeNodeFlags::SELECTED;
    }
    let glyph = crate::browser::browser_art::feature_glyph(feature.kind);
    let icon_inset = if glyph.is_some() { "      " } else { "" };
    let node = ui
        .tree_node_config("feature-row")
        .label::<String, String>(format!(
            "{icon_inset}{}",
            crate::browser::browser_import::feature_label(
                browser,
                feature,
                view.show_extended_names
            )
        ))
        .flags(flags)
        .default_open(feature.kind != "User Coordinate System");
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
    let label = crate::browser::browser_art::tree_label_anchor(ui);
    if let Some(glyph) = glyph {
        crate::browser::browser_art::prefix(ui, view, glyph, label);
    }
    crate::browser::browser_import::draw_link_indicator(ui, browser, feature, label);
    if feature.suppressed {
        crate::browser::browser_art::strike_feature_name(
            ui,
            &feature.name,
            label[0] + ui.calc_text_size(icon_inset)[0],
        );
    }
    let row_hovered = ui.is_item_hovered();
    let row_left = ui.item_rect_min()[0];
    let indicator_hovered = states::model_state_indicator(ui, browser, feature);
    choose_on_press(
        ui,
        state,
        (browser, feature),
        view,
        actions,
        row_hovered && !indicator_hovered,
        row_left,
    );
    feature_menu(ui, state, browser, feature, view, actions);
    token
}
/// The arrow only folds the branch; the rest of the row chooses the feature.
fn choose_on_press(
    ui: &Ui,
    state: &GuiCommandState,
    (browser, feature): (&GuiModelBrowser, &GuiBrowserFeature),
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
    hovered: bool,
    row_left: f32,
) {
    let asks_menu = hovered && ui.is_mouse_clicked(MouseButton::Right);
    if asks_menu {
        ui.open_popup(FEATURE_MENU);
    }
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
    if clicked && feature.kind == "Work Plane" && !feature.suppressed {
        view.picked_plane_reference = Some(format!("datum/v1/{}", feature.id.0));
    }
    if clicked && ui.is_mouse_double_clicked(MouseButton::Left) {
        let double_click = edit::feature_double_click_action(browser, feature);
        if double_click.enabled(state) {
            actions.push(double_click);
        }
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
    if crate::browser::browser_ucs::custom_feature_menu(ui, state, browser, feature, view, actions)
    {
        return;
    }
    edit_sketch_row(ui, state, browser, feature, actions);
    let edit = feature_edit_action(browser, feature);
    if ui
        .menu_item_config(edit::feature_edit_label(browser, feature))
        .enabled(edit.enabled(state))
        .build()
    {
        actions.push(edit);
    }
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
    if state.selected_body_keys.contains(&body.key) {
        flags |= TreeNodeFlags::SELECTED;
    }
    let node = ui
        .tree_node_config(&body.name)
        .label::<String, String>(format!("      {}", body.name))
        .flags(flags);
    let node = match expansion {
        Some(open) => node.opened(open, Condition::Always),
        None => node,
    };
    let muted = (!body.visible).then(|| {
        ui.push_style_color(
            imgui::StyleColor::Text,
            ui.style_color(imgui::StyleColor::TextDisabled),
        )
    });
    let token = node.push();
    drop(muted);
    let label = crate::browser::browser_art::tree_label_anchor(ui);
    crate::browser::browser_art::prefix(ui, view, crate::icons::IconGlyph::NewPart, label);
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
                toggle: !asks_menu && (ui.io().key_ctrl || ui.io().key_shift),
            },
        });
    }
    if let Some(_menu) = ui.begin_popup(BODY_MENU) {
        if browser.has_solid(body.feature) && ui.menu_item("Properties") {
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
    if commands.selected_edge_count > 0 {
        edge_selection_toolbar(ui, request, actions);
        return;
    }
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
        let edit = feature_edit_action(browser, feature);
        let _muted = ui.begin_disabled(!edit.enabled(commands));
        if crate::widget_abi::button(ui, "Edit Feature", [0.; 2]) && edit.enabled(commands) {
            actions.push(edit);
        }
    });
}
fn edge_selection_toolbar(ui: &Ui, request: &GuiFrameRequest, actions: &mut Vec<GuiShellAction>) {
    let Some(corner) = outline_corner(request) else {
        return;
    };
    selection_toolbar_window(ui, corner, || {
        let action = GuiShellAction::OpenFillet;
        let disabled = ui.begin_disabled(!action.dispatchable(&request.commands));
        if crate::widget_abi::button(ui, "Fillet", [0.; 2]) {
            actions.push(action);
        }
        drop(disabled);
    });
}
fn face_selection_toolbar(ui: &Ui, request: &GuiFrameRequest, actions: &mut Vec<GuiShellAction>) {
    let placement = face_toolbar_placement::near_pick(ui, request)
        .or_else(|| outline_corner(request).map(|[x, y]| ([x, y - TOOLBAR_RISE], [0.; 2])));
    let Some((position, pivot)) = placement else {
        return;
    };
    selection_toolbar_at(ui, position, pivot, || {
        if crate::widget_abi::button(ui, face_toolbar_placement::ZOOM_LABEL, [0.; 2]) {
            actions.push(GuiShellAction::ViewCommand(
                crate::GuiViewCommand::ZoomSelected {
                    extent: request.extent,
                },
            ));
        }
    });
}
fn selection_toolbar_window(ui: &Ui, corner: [f32; 2], draw: impl FnOnce()) {
    selection_toolbar_at(ui, [corner[0], corner[1] - TOOLBAR_RISE], [0.; 2], draw);
}
fn selection_toolbar_at(ui: &Ui, position: [f32; 2], pivot: [f32; 2], draw: impl FnOnce()) {
    crate::widget_abi::window_position_with_pivot(ui, position, pivot, Condition::Always);
    ui.window(SELECTION_TOOLBAR)
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
    if !request.commands.selected_edge_outlines.is_empty() {
        let pointer = std::array::from_fn(|axis| ui.io().mouse_pos[axis] * scale[axis]);
        for group in &request.commands.selected_edge_outlines {
            let color = edge_color(&request.commands, edge_group_hovered(pointer, group));
            for [from, to] in group {
                list.add_line(logical(*from), logical(*to), color)
                    .thickness(EDGE_THICKNESS)
                    .build();
            }
        }
        return;
    }
    let color = feature_outline_color(&request.commands);
    for [from, to] in &request.commands.feature_edges {
        list.add_line(logical(*from), logical(*to), color)
            .thickness(EDGE_THICKNESS)
            .build();
    }
}
fn feature_outline_color(commands: &GuiCommandState) -> [f32; 4] {
    edge_color(
        commands,
        commands.model_face_chosen && commands.selected_body_keys.is_empty(),
    )
}
fn edge_color(commands: &GuiCommandState, hovered: bool) -> [f32; 4] {
    commands.color_scheme.as_ref().map_or(
        if hovered {
            EDGE_HOVER_COLOR
        } else {
            EDGE_COLOR
        },
        |scheme| scheme.geometry_selection(hovered),
    )
}
fn edge_group_hovered(pointer: [f32; 2], segments: &[[[i32; 2]; 2]]) -> bool {
    pointer.iter().all(|value| value.is_finite())
        && segments.iter().any(|segment| {
            segment_distance_squared(pointer, *segment) <= EDGE_HOVER_REACH * EDGE_HOVER_REACH
        })
}
fn segment_distance_squared(point: [f32; 2], segment: [[i32; 2]; 2]) -> f32 {
    let a = segment[0].map(|value| value as f32);
    let b = segment[1].map(|value| value as f32);
    let v = [b[0] - a[0], b[1] - a[1]];
    let w = [point[0] - a[0], point[1] - a[1]];
    let length = v[0] * v[0] + v[1] * v[1];
    let ratio = if length > 0. {
        ((w[0] * v[0] + w[1] * v[1]) / length).clamp(0., 1.)
    } else {
        0.
    };
    (w[0] - ratio * v[0]).powi(2) + (w[1] - ratio * v[1]).powi(2)
}

#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/browser_feature_unit.rs"]
mod tests;

#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/browser_feature_click.rs"]
mod click_tests;
