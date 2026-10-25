// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiShellAction;
use imgui::{Condition, StyleColor, Ui, WindowFlags};

use super::workspace_shell::RIBBON_BOTTOM;

// Go's DockBuilder splits the work area 22% left and 7% down after fixed chrome claims its bands.
const MODEL_WIDTH_FRACTION: f32 = 0.22;
const COMMAND_HEIGHT_FRACTION: f32 = 0.07;
const COMMAND_MIN_HEIGHT: f32 = 43.;
// Go's status side bar is one 16 px text line with the default 8 px top/bottom padding.
const STATUS_HEIGHT: f32 = 32.;
const VIEWPORT_HEADER_HEIGHT: f32 = 22.;
// A tab strip lives inside the viewport window: 8 px top/bottom padding plus its 18 px tab row.
const DOCUMENT_TABS_HEIGHT: f32 = 34.;
const VIEW_CUBE_SIZE: f32 = 86.;
const VIEW_CUBE_EDGE_BAND: f32 = 0.2;
const VIEW_CUBE_HOME_SIZE: f32 = 24.;
const VIEW_CUBE_HOME_GAP: f32 = 5.;
const VIEW_CUBE_HOME_LEFT_INSET: f32 = 12.;
const OVERLAY_MARGIN: f32 = 10.;
const NAVIGATION_BAR_HEIGHT: f32 = 146.;
const AXIS_TRIAD_MARGIN: f32 = 46.;
const AXIS_TRIAD_RADIUS: f32 = 30.;
const AXIS_ARROW_HEAD: f32 = 9.;
pub(super) const GO_DOCK_ACCENT: [f32; 4] = [0.506, 0.376, 0.89, 1.];
const GO_DOCK_HEADER: [f32; 4] = [0.129, 0.129, 0.129, 1.];
const DOCK_TAB_LEFT_INSET: f32 = 26.;
const DOCK_TAB_TOP_INSET: f32 = 1.;
const DOCK_TAB_TEXT_INSET: f32 = 5.;
const DOCK_TAB_RIGHT_INSET: f32 = 13.;
const DOCK_TAB_CLOSE_RESERVE: f32 = 25.;

#[derive(Clone, Copy, Debug, PartialEq)]
struct WorkspaceLayout {
    model_position: [f32; 2],
    model_size: [f32; 2],
    viewport_header_position: [f32; 2],
    viewport_header_size: [f32; 2],
    document_tabs_position: [f32; 2],
    document_tabs_size: [f32; 2],
    command_position: [f32; 2],
    command_size: [f32; 2],
}

fn workspace_layout(display: [f32; 2]) -> WorkspaceLayout {
    let model_width = model_width(display);
    let work_width = (display[0] - model_width).max(0.);
    let work_height = (display[1] - RIBBON_BOTTOM - STATUS_HEIGHT).max(0.);
    let command_height = (work_height * COMMAND_HEIGHT_FRACTION).max(COMMAND_MIN_HEIGHT);
    WorkspaceLayout {
        model_position: [0., RIBBON_BOTTOM],
        model_size: [model_width, work_height],
        viewport_header_position: [model_width, RIBBON_BOTTOM],
        viewport_header_size: [work_width, VIEWPORT_HEADER_HEIGHT],
        document_tabs_position: [model_width, RIBBON_BOTTOM + VIEWPORT_HEADER_HEIGHT],
        document_tabs_size: [work_width, DOCUMENT_TABS_HEIGHT],
        command_position: [model_width, display[1] - STATUS_HEIGHT - command_height],
        command_size: [work_width, command_height],
    }
}

fn model_width(display: [f32; 2]) -> f32 {
    display[0] * MODEL_WIDTH_FRACTION
}

pub(super) fn draw_status_bar(ui: &Ui, status: Option<&str>) {
    let display = ui.io().display_size;
    ui.window("Status")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .position([0., display[1] - STATUS_HEIGHT], Condition::Always)
        .size([display[0], STATUS_HEIGHT], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| ui.text(status_text(status)));
}

fn status_text(status: Option<&str>) -> &str {
    status
        .filter(|text| !text.is_empty())
        .unwrap_or("Press F1 for more help")
}

pub(super) fn draw_document_tabs(
    ui: &Ui,
    documents: &[crate::GuiDocumentTab],
    selected: &mut Vec<GuiShellAction>,
    workspace: &mut super::workspace_shell::WorkspaceShellState,
) {
    if documents.is_empty() {
        return;
    }
    let layout = workspace_layout(ui.io().display_size);
    ui.window("Documents")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .position(layout.document_tabs_position, Condition::Always)
        .size(layout.document_tabs_size, Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| {
            let Some(_tabs) = ui.tab_bar("##document-tabs") else {
                return;
            };
            for document in documents {
                draw_document_tab(ui, document, selected, workspace);
            }
        });
}

pub(super) fn draw_viewport_header(ui: &Ui, has_document: bool) {
    if !has_document {
        return;
    }
    let layout = workspace_layout(ui.io().display_size);
    let _title = ui.push_style_color(StyleColor::TitleBg, GO_DOCK_HEADER);
    let _active = ui.push_style_color(StyleColor::TitleBgActive, GO_DOCK_HEADER);
    ui.window("##viewport-window")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .position(layout.viewport_header_position, Condition::Always)
        .size(layout.viewport_header_size, Condition::Always)
        .movable(false)
        .resizable(false)
        .build(|| {});
    draw_dock_tab(ui, layout.viewport_header_position, "Viewport", false);
}

fn draw_dock_tab(ui: &Ui, header_position: [f32; 2], label: &str, closable: bool) {
    let text_width = ui.calc_text_size(label)[0];
    let (min, max) = dock_tab_bounds(header_position, text_width, closable);
    let draw = ui.get_foreground_draw_list();
    draw.add_rect(min, max, GO_DOCK_ACCENT).filled(true).build();
    draw.add_text(
        [min[0] + DOCK_TAB_TEXT_INSET, min[1] + 3.],
        [0.773, 0.773, 0.773, 1.],
        label,
    );
    if closable {
        draw.add_text([max[0] - 16., min[1] + 3.], [0.86, 0.86, 0.86, 1.], "×");
    }
}

fn dock_tab_bounds(position: [f32; 2], text_width: f32, closable: bool) -> ([f32; 2], [f32; 2]) {
    let close_reserve = if closable { DOCK_TAB_CLOSE_RESERVE } else { 0. };
    let min = [
        position[0] + DOCK_TAB_LEFT_INSET,
        position[1] + DOCK_TAB_TOP_INSET,
    ];
    let max = [
        min[0] + DOCK_TAB_TEXT_INSET + text_width + DOCK_TAB_RIGHT_INSET + close_reserve,
        position[1] + VIEWPORT_HEADER_HEIGHT,
    ];
    (min, max)
}

pub(super) fn draw_viewport_overlays(
    ui: &Ui,
    has_document: bool,
    model_open: bool,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut super::workspace_shell::WorkspaceShellState,
) {
    if !has_document {
        return;
    }
    let (origin, size) = viewport_rect(ui.io().display_size, model_open);
    draw_axis_triad(ui, origin, size);
    draw_view_cube(ui, origin, size, selected);
    draw_navigation_bar(ui, origin, size, selected, icons, workspace);
}

fn draw_axis_triad(ui: &Ui, origin: [f32; 2], size: [f32; 2]) {
    let center = [
        origin[0] + AXIS_TRIAD_MARGIN,
        origin[1] + size[1] - AXIS_TRIAD_MARGIN,
    ];
    for (offset, color, label) in axis_triad_arrows() {
        let tip = [
            center[0] + offset[0] * AXIS_TRIAD_RADIUS,
            center[1] + offset[1] * AXIS_TRIAD_RADIUS,
        ];
        draw_axis_arrow(ui, center, tip, color, label);
    }
}

fn axis_triad_arrows() -> [([f32; 2], [f32; 4], &'static str); 3] {
    [
        ([0.86, 0.32], [0.90, 0.23, 0.27, 1.], "X"),
        ([-0.48, 0.58], [0.42, 0.74, 0.28, 1.], "Y"),
        ([0., -1.], [0.24, 0.50, 0.92, 1.], "Z"),
    ]
}

fn draw_axis_arrow(ui: &Ui, center: [f32; 2], tip: [f32; 2], color: [f32; 4], label: &str) {
    let draw = ui.get_foreground_draw_list();
    draw.add_line(center, tip, color).thickness(2.4).build();
    if let Some(head) = axis_arrowhead(center, tip) {
        draw.add_polyline(head.to_vec(), color).filled(true).build();
    }
    draw.add_text([tip[0] + 3., tip[1] - 7.], color, label);
}

fn axis_arrowhead(center: [f32; 2], tip: [f32; 2]) -> Option<[[f32; 2]; 3]> {
    let direction = [tip[0] - center[0], tip[1] - center[1]];
    let length = direction[0].hypot(direction[1]);
    if length < f32::EPSILON {
        return None;
    }
    let unit = [direction[0] / length, direction[1] / length];
    let perpendicular = [-unit[1], unit[0]];
    let base = [
        tip[0] - unit[0] * AXIS_ARROW_HEAD,
        tip[1] - unit[1] * AXIS_ARROW_HEAD,
    ];
    let half_width = AXIS_ARROW_HEAD * 0.45;
    Some([
        tip,
        [
            base[0] + perpendicular[0] * half_width,
            base[1] + perpendicular[1] * half_width,
        ],
        [
            base[0] - perpendicular[0] * half_width,
            base[1] - perpendicular[1] * half_width,
        ],
    ])
}

fn viewport_rect(display: [f32; 2], model_open: bool) -> ([f32; 2], [f32; 2]) {
    let layout = workspace_layout(display);
    let left = if model_open { layout.model_size[0] } else { 0. };
    let top = layout.document_tabs_position[1] + layout.document_tabs_size[1];
    let bottom = layout.command_position[1];
    (
        [left, top],
        [(display[0] - left).max(0.), (bottom - top).max(0.)],
    )
}

fn draw_view_cube(ui: &Ui, origin: [f32; 2], size: [f32; 2], selected: &mut Vec<GuiShellAction>) {
    let position = [
        origin[0] + size[0] - VIEW_CUBE_SIZE - OVERLAY_MARGIN,
        origin[1] + OVERLAY_MARGIN,
    ];
    ui.window("##view-cube")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .position(position, Condition::Always)
        .size([VIEW_CUBE_SIZE, VIEW_CUBE_SIZE], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .scroll_bar(false)
        .draw_background(false)
        .build(|| draw_view_cube_faces(ui, selected));
    draw_view_cube_home(ui, position, selected);
}

fn draw_view_cube_faces(ui: &Ui, selected: &mut Vec<GuiShellAction>) {
    let origin = ui.cursor_screen_pos();
    let clicked = ui.invisible_button("##view-cube-regions", [VIEW_CUBE_SIZE, VIEW_CUBE_SIZE]);
    draw_cube_faces(ui, origin);
    draw_cube_labels(ui, origin);
    if clicked {
        selected.push(GuiShellAction::Home);
    }
}

fn draw_view_cube_home(ui: &Ui, cube: [f32; 2], selected: &mut Vec<GuiShellAction>) {
    let position = view_cube_home_position(cube);
    ui.window("##view-cube-home")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .position(position, Condition::Always)
        .size(
            [VIEW_CUBE_HOME_SIZE, VIEW_CUBE_HOME_SIZE],
            Condition::Always,
        )
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .scroll_bar(false)
        .build(|| draw_view_cube_home_button(ui, selected));
}

fn view_cube_home_position(cube: [f32; 2]) -> [f32; 2] {
    [
        cube[0] + VIEW_CUBE_HOME_LEFT_INSET,
        cube[1] + VIEW_CUBE_SIZE + VIEW_CUBE_HOME_GAP,
    ]
}

fn draw_view_cube_home_button(ui: &Ui, selected: &mut Vec<GuiShellAction>) {
    let origin = ui.cursor_screen_pos();
    let clicked = ui.invisible_button(
        "##view-cube-go-home",
        [VIEW_CUBE_HOME_SIZE, VIEW_CUBE_HOME_SIZE],
    );
    let draw = ui.get_window_draw_list();
    let color = [0.72, 0.76, 0.82, 0.95];
    draw.add_polyline(
        vec![
            [origin[0] + 4., origin[1] + 12.],
            [origin[0] + 12., origin[1] + 5.],
            [origin[0] + 20., origin[1] + 12.],
        ],
        color,
    )
    .thickness(1.8)
    .build();
    draw.add_polyline(
        vec![
            [origin[0] + 6., origin[1] + 11.],
            [origin[0] + 6., origin[1] + 20.],
            [origin[0] + 18., origin[1] + 20.],
            [origin[0] + 18., origin[1] + 11.],
        ],
        color,
    )
    .thickness(1.8)
    .build();
    if clicked {
        selected.push(GuiShellAction::Home);
    }
}

fn draw_cube_faces(ui: &Ui, origin: [f32; 2]) {
    let points = cube_points(origin);
    draw_cube_compass(ui, origin);
    draw_cube_face_fills(ui, points);
    draw_cube_grids(ui, points);
}

fn draw_cube_face_fills(ui: &Ui, points: [[[f32; 2]; 4]; 3]) {
    let draw = ui.get_window_draw_list();
    for (face, color) in points.into_iter().zip([
        [0.82, 0.84, 0.88, 0.92],
        [0.65, 0.68, 0.74, 0.92],
        [0.72, 0.75, 0.80, 0.92],
    ]) {
        draw.add_polyline(face.to_vec(), color).filled(true).build();
    }
}

fn draw_cube_compass(ui: &Ui, origin: [f32; 2]) {
    let center = [origin[0] + 43., origin[1] + 43.];
    let draw = ui.get_window_draw_list();
    draw.add_circle(center, 39., [0.64, 0.68, 0.74, 0.9])
        .thickness(1.2)
        .build();
    for (offset, label) in [
        ([0., -42.], "N"),
        ([42., 0.], "E"),
        ([0., 37.], "S"),
        ([-42., 0.], "W"),
    ] {
        draw.add_text(
            [center[0] + offset[0] - 3., center[1] + offset[1]],
            [0.75, 0.78, 0.84, 0.9],
            label,
        );
    }
}

fn draw_cube_grids(ui: &Ui, points: [[[f32; 2]; 4]; 3]) {
    let draw = ui.get_window_draw_list();
    for face in points {
        for fraction in cube_grid_fractions() {
            draw.add_line(
                quad_point(face, fraction, 0.),
                quad_point(face, fraction, 1.),
                [0.30, 0.33, 0.39, 0.7],
            )
            .thickness(1.)
            .build();
            draw.add_line(
                quad_point(face, 0., fraction),
                quad_point(face, 1., fraction),
                [0.30, 0.33, 0.39, 0.7],
            )
            .thickness(1.)
            .build();
        }
    }
}

fn cube_grid_fractions() -> [f32; 2] {
    [VIEW_CUBE_EDGE_BAND, 1. - VIEW_CUBE_EDGE_BAND]
}

fn quad_point(quad: [[f32; 2]; 4], horizontal: f32, vertical: f32) -> [f32; 2] {
    let top = interpolate_point(quad[0], quad[1], horizontal);
    let bottom = interpolate_point(quad[3], quad[2], horizontal);
    interpolate_point(top, bottom, vertical)
}

fn interpolate_point(start: [f32; 2], end: [f32; 2], fraction: f32) -> [f32; 2] {
    [
        start[0] + (end[0] - start[0]) * fraction,
        start[1] + (end[1] - start[1]) * fraction,
    ]
}

fn draw_cube_labels(ui: &Ui, origin: [f32; 2]) {
    let draw = ui.get_window_draw_list();
    for (offset, label) in [([32., 25.], "TOP"), ([21., 51.], "F"), ([59., 51.], "R")] {
        draw.add_text(
            [origin[0] + offset[0], origin[1] + offset[1]],
            [0.12, 0.14, 0.18, 1.],
            label,
        );
    }
}

fn cube_points(origin: [f32; 2]) -> [[[f32; 2]; 4]; 3] {
    let [x, y] = origin;
    [
        [
            [x + 43., y + 8.],
            [x + 76., y + 27.],
            [x + 43., y + 45.],
            [x + 10., y + 27.],
        ],
        [
            [x + 10., y + 27.],
            [x + 43., y + 45.],
            [x + 43., y + 77.],
            [x + 10., y + 60.],
        ],
        [
            [x + 43., y + 45.],
            [x + 76., y + 27.],
            [x + 76., y + 60.],
            [x + 43., y + 77.],
        ],
    ]
}

fn draw_navigation_bar(
    ui: &Ui,
    origin: [f32; 2],
    size: [f32; 2],
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut super::workspace_shell::WorkspaceShellState,
) {
    let position = [
        origin[0] + size[0] - 42.,
        origin[1] + (size[1] - NAVIGATION_BAR_HEIGHT) / 2.,
    ];
    ui.window("##navigation-bar")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .position(position, Condition::Always)
        .size([34., NAVIGATION_BAR_HEIGHT], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .scroll_bar(false)
        .build(|| draw_navigation_buttons(ui, selected, icons, workspace));
}

fn draw_navigation_buttons(
    ui: &Ui,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut super::workspace_shell::WorkspaceShellState,
) {
    for button in &navigation_buttons() {
        draw_navigation_button(ui, button, selected, icons, workspace);
    }
}

struct NavigationButton {
    label: &'static str,
    icon: &'static str,
    action: GuiShellAction,
    enabled: bool,
}

fn navigation_buttons() -> [NavigationButton; 5] {
    [
        NavigationButton {
            label: "Home",
            icon: "home",
            action: GuiShellAction::Home,
            enabled: true,
        },
        NavigationButton {
            label: "Fit",
            icon: "zoom-all",
            action: GuiShellAction::Home,
            enabled: true,
        },
        NavigationButton {
            label: "Zoom Window",
            icon: "zoom-window",
            action: GuiShellAction::Home,
            enabled: false,
        },
        NavigationButton {
            label: "Constrained Orbit",
            icon: "orbit-constrained",
            action: GuiShellAction::Home,
            enabled: false,
        },
        NavigationButton {
            label: "Look At",
            icon: "look-at",
            action: GuiShellAction::Home,
            enabled: false,
        },
    ]
}

fn draw_navigation_button(
    ui: &Ui,
    button: &NavigationButton,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut super::workspace_shell::WorkspaceShellState,
) {
    let disabled = ui.begin_disabled(!button.enabled);
    let clicked = icons.texture_id(button.icon).map_or_else(
        || ui.button(button.label),
        |id| {
            ui.image_button(
                format!("##navigation-{}", button.label),
                id.into(),
                [22., 22.],
            )
        },
    );
    drop(disabled);
    workspace.record_target(
        &format!("Navigation.{}", button.label),
        button.enabled,
        ui.item_rect_min(),
        ui.item_rect_max(),
    );
    if clicked {
        selected.push(button.action.clone());
    }
    if ui.is_item_hovered() {
        ui.tooltip_text(button.label);
    }
}

fn draw_document_tab(
    ui: &Ui,
    document: &crate::GuiDocumentTab,
    selected: &mut Vec<GuiShellAction>,
    workspace: &mut super::workspace_shell::WorkspaceShellState,
) {
    let mut open = true;
    let label = document_tab_label(document);
    let active = ui.tab_item_with_opened(label, &mut open).is_some();
    workspace.record_target(
        &format!("Document.{}", document.id),
        true,
        ui.item_rect_min(),
        ui.item_rect_max(),
    );
    if active && !document.active {
        selected.push(GuiShellAction::ActivateDocument(document.id));
    }
    if !open {
        selected.push(GuiShellAction::CloseDocument(document.id));
    }
}

fn document_tab_label(document: &crate::GuiDocumentTab) -> String {
    let dirty = if document.dirty { " *" } else { "" };
    format!("{}{dirty}###document-{}", document.name, document.id)
}

pub(super) fn draw_model_browser(
    ui: &Ui,
    tree: Option<&crate::GuiModelTree>,
    selected: &mut Vec<GuiShellAction>,
    open: &mut bool,
) {
    let layout = workspace_layout(ui.io().display_size);
    let _title = ui.push_style_color(StyleColor::TitleBg, GO_DOCK_HEADER);
    let _active = ui.push_style_color(StyleColor::TitleBgActive, GO_DOCK_HEADER);
    ui.window("##model-window")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .opened(open)
        .position(layout.model_position, Condition::Always)
        .size(layout.model_size, Condition::Always)
        .movable(false)
        .resizable(false)
        .build(|| {
            let Some(tree) = tree else {
                ui.text("No active part");
                return;
            };
            draw_model_tree(ui, tree, selected);
        });
    draw_dock_tab(ui, layout.model_position, "Model", true);
}

fn draw_model_tree(ui: &Ui, tree: &crate::GuiModelTree, selected: &mut Vec<GuiShellAction>) {
    let Some(_document) = ui
        .tree_node_config(&tree.document)
        .default_open(true)
        .push()
    else {
        return;
    };
    draw_model_origin(ui);
    draw_model_parameters(ui, &tree.parameters);
    draw_model_sketches(ui, tree.sketches);
    draw_model_features(ui, &tree.features);
    draw_model_bodies(ui, &tree.bodies, selected);
}

fn draw_model_origin(ui: &Ui) {
    if let Some(_origin) = ui.tree_node("Origin") {
        ui.bullet_text("XY Plane");
        ui.bullet_text("XZ Plane");
        ui.bullet_text("YZ Plane");
    }
}

fn draw_model_parameters(ui: &Ui, parameters: &[String]) {
    if parameters.is_empty() {
        ui.bullet_text("Parameters");
        return;
    }
    let label = format!("Parameters ({})", parameters.len());
    let Some(_parameters) = ui.tree_node(&label) else {
        return;
    };
    for parameter in parameters {
        ui.bullet_text(parameter);
    }
}

fn draw_model_sketches(ui: &Ui, count: usize) {
    if count == 0 {
        ui.bullet_text("Sketches");
        return;
    }
    let label = format!("Sketches ({count})");
    let Some(_sketches) = ui.tree_node(&label) else {
        return;
    };
    for index in 1..=count {
        ui.bullet_text(format!("Sketch {index}"));
    }
}

fn draw_model_features(ui: &Ui, features: &[crate::GuiModelFeature]) {
    if features.is_empty() {
        ui.bullet_text("Features");
        return;
    }
    let label = format!("Features ({})", features.len());
    let Some(_features) = ui.tree_node(&label) else {
        return;
    };
    for feature in features {
        ui.bullet_text(model_feature_label(feature));
    }
}

fn model_feature_label(feature: &crate::GuiModelFeature) -> String {
    format!("{} ({}) — {}", feature.name, feature.kind, feature.health)
}

fn draw_model_bodies(ui: &Ui, bodies: &[crate::GuiModelBody], selected: &mut Vec<GuiShellAction>) {
    if bodies.is_empty() {
        ui.bullet_text("Bodies");
        return;
    }
    let label = format!("Bodies ({})", bodies.len());
    let Some(_bodies) = ui.tree_node_config(&label).default_open(true).push() else {
        return;
    };
    for body in bodies {
        draw_body_row(ui, body, selected);
    }
}

fn draw_body_row(ui: &Ui, body: &crate::GuiModelBody, selected: &mut Vec<GuiShellAction>) {
    let mut visible = body.visible;
    if ui.checkbox(format!("##body-visible-{}", body.index), &mut visible) {
        selected.push(GuiShellAction::SetBodyVisible {
            index: body.index,
            visible,
        });
    }
    ui.same_line();
    ui.text(format!("{} — {} faces", body.name, body.faces));
}

pub(super) fn draw_command_window(ui: &Ui, status: Option<&str>, open: &mut bool) {
    let layout = workspace_layout(ui.io().display_size);
    let _title = ui.push_style_color(StyleColor::TitleBg, GO_DOCK_ACCENT);
    let _active = ui.push_style_color(StyleColor::TitleBgActive, GO_DOCK_ACCENT);
    ui.window("Command")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .opened(open)
        .position(layout.command_position, Condition::Always)
        .size(layout.command_size, Condition::Always)
        .movable(false)
        .resizable(false)
        .build(|| {
            ui.text_wrapped(status_text(status));
        });
}

#[cfg(test)]
mod tests {
    use super::{
        AXIS_ARROW_HEAD, axis_arrowhead, axis_triad_arrows, cube_grid_fractions, cube_points,
        dock_tab_bounds, model_feature_label, navigation_buttons, quad_point, status_text,
        view_cube_home_position, viewport_rect, workspace_layout,
    };
    use crate::{
        GuiCommandState, GuiDocumentTab, GuiFrameEngine, GuiFrameRequest, GuiModelFeature,
    };

    #[test]
    fn feature_rows_keep_name_kind_and_health() {
        let label = model_feature_label(&GuiModelFeature {
            name: "Extrude1".into(),
            kind: "Extrude".into(),
            health: "Healthy".into(),
        });
        assert_eq!(label, "Extrude1 (Extrude) — Healthy");
    }

    #[test]
    fn fallback_layout_matches_the_go_model_viewport_command_arrangement() {
        let layout = workspace_layout([1440., 900.]);
        assert_eq!(layout.model_position, [0., 192.]);
        assert_eq!(layout.model_size, [316.8, 676.]);
        assert_eq!(layout.viewport_header_position, [316.8, 192.]);
        assert_eq!(layout.viewport_header_size, [1123.2, 22.]);
        assert_eq!(layout.document_tabs_position, [316.8, 214.]);
        assert_eq!(layout.document_tabs_size, [1123.2, 34.]);
        assert_eq!(layout.command_position, [316.8, 820.68]);
        assert_eq!(layout.command_size, [1123.2, 47.32]);
    }

    #[test]
    fn compact_layout_preserves_go_command_minimum_and_status_band() {
        let layout = workspace_layout([800., 600.]);
        assert_eq!(layout.model_position, [0., 192.]);
        assert_eq!(layout.model_size, [176., 376.]);
        assert_eq!(layout.command_position, [176., 525.]);
        assert_eq!(layout.command_size, [624., 43.]);
    }

    #[test]
    fn dock_tab_bounds_reserves_the_go_close_affordance_only_when_needed() {
        assert_eq!(
            dock_tab_bounds([176., 192.], 58., false),
            ([202., 193.], [278., 214.])
        );
        assert_eq!(
            dock_tab_bounds([0., 192.], 40., true),
            ([26., 193.], [109., 214.])
        );
    }

    #[test]
    fn idle_status_matches_the_go_status_presentation() {
        assert_eq!(status_text(None), "Press F1 for more help");
        assert_eq!(status_text(Some("")), "Press F1 for more help");
        assert_eq!(status_text(Some("Click a face")), "Click a face");
    }

    #[test]
    fn overlays_anchor_to_the_visible_viewport() {
        assert_eq!(
            viewport_rect([1440., 900.], true),
            ([316.8, 248.], [1123.2, 572.68])
        );
        assert_eq!(
            viewport_rect([1440., 900.], false),
            ([0., 248.], [1440., 572.68])
        );
    }

    #[test]
    fn view_cube_faces_share_the_front_top_right_corner() {
        let faces = cube_points([100., 200.]);
        assert_eq!(faces[0][2], [143., 245.]);
        assert_eq!(faces[1][1], [143., 245.]);
        assert_eq!(faces[2][0], [143., 245.]);
    }

    #[test]
    fn view_cube_keeps_the_go_thin_edge_grid() {
        assert_eq!(cube_grid_fractions(), [0.2, 0.8]);
        assert_eq!(
            quad_point([[0., 0.], [10., 0.], [10., 10.], [0., 10.]], 0.2, 0.8),
            [2., 8.]
        );
    }

    #[test]
    fn view_cube_home_button_sits_below_the_compass() {
        assert_eq!(view_cube_home_position([100., 200.]), [112., 291.]);
    }

    #[test]
    fn navigation_bar_keeps_the_five_go_navigation_tools() {
        let buttons = navigation_buttons();
        assert_eq!(buttons.len(), 5);
        assert_eq!(buttons[0].label, "Home");
        assert_eq!(buttons[4].label, "Look At");
    }

    #[test]
    fn axis_triad_keeps_the_go_xyz_order_and_colours() {
        let arrows = axis_triad_arrows();
        assert_eq!(arrows.map(|arrow| arrow.2), ["X", "Y", "Z"]);
        assert_eq!(arrows[0].1, [0.90, 0.23, 0.27, 1.]);
        assert_eq!(arrows[1].1, [0.42, 0.74, 0.28, 1.]);
        assert_eq!(arrows[2].1, [0.24, 0.50, 0.92, 1.]);
    }

    #[test]
    fn axis_arrowhead_keeps_the_go_length_and_width() {
        let head = axis_arrowhead([0., 0.], [30., 0.]).unwrap();
        assert_eq!(head[0], [30., 0.]);
        assert_eq!(head[1][0], 21.);
        assert_eq!(head[2][0], 21.);
        assert_eq!(head[1][1], AXIS_ARROW_HEAD * 0.45);
        assert_eq!(head[2][1], -AXIS_ARROW_HEAD * 0.45);
        assert_eq!(axis_arrowhead([2., 2.], [2., 2.]), None);
    }

    #[test]
    fn document_frame_includes_the_view_cube_faces() {
        let mut engine = GuiFrameEngine::new().unwrap();
        let request = GuiFrameRequest {
            extent: [1440, 900],
            commands: GuiCommandState {
                documents: vec![GuiDocumentTab {
                    id: 1,
                    name: "Part".into(),
                    active: true,
                    ..Default::default()
                }],
                ..Default::default()
            },
            delta_seconds: 1. / 60.,
            framebuffer_scale: [1.; 2],
            events: Vec::new(),
            status: None,
        };
        let output = engine.frame(&request).unwrap();
        assert!(
            output
                .targets
                .iter()
                .any(|target| target.command_id == "Navigation.Home" && target.enabled)
        );
        let frame = output.draws;
        assert!(
            frame
                .batches
                .iter()
                .flat_map(|batch| &batch.vertices)
                .any(|vertex| { vertex.srgb == [209, 214, 224, 235] })
        );
        assert!(!frame.batches.iter().any(default_debug_surface));
    }

    fn default_debug_surface(batch: &crate::GuiBatch) -> bool {
        let Some(first) = batch.vertices.first() else {
            return false;
        };
        first.srgb == [22, 22, 22, 255]
            && batch.clip_rect == [0., 0., 1440., 900.]
            && bounds(batch.vertices.as_slice())
                .iter()
                .zip([59., 59., 461., 461.])
                .all(|(actual, expected)| (actual - expected).abs() < 0.01)
    }

    fn bounds(vertices: &[crate::GuiVertex]) -> [f32; 4] {
        vertices.iter().fold(
            [f32::MAX, f32::MAX, f32::MIN, f32::MIN],
            |[left, top, right, bottom], vertex| {
                [
                    left.min(vertex.position[0]),
                    top.min(vertex.position[1]),
                    right.max(vertex.position[0]),
                    bottom.max(vertex.position[1]),
                ]
            },
        )
    }
}
