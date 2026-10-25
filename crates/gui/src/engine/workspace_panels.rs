// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiShellAction;
use imgui::{Condition, TreeNodeFlags, Ui, WindowFlags};

use super::workspace_shell::RIBBON_BOTTOM;

const STATUS_HEIGHT: f32 = 28.;
const MODEL_WIDTH: f32 = 280.;
const COMMAND_HEIGHT: f32 = 130.;
const VIEW_CUBE_SIZE: f32 = 86.;
const OVERLAY_MARGIN: f32 = 10.;
const NAVIGATION_BAR_HEIGHT: f32 = 146.;
const AXIS_TRIAD_MARGIN: f32 = 46.;
const AXIS_TRIAD_RADIUS: f32 = 30.;

#[derive(Clone, Copy, Debug, PartialEq)]
struct WorkspaceLayout {
    model_position: [f32; 2],
    model_size: [f32; 2],
    document_position: [f32; 2],
    document_size: [f32; 2],
    command_position: [f32; 2],
    command_size: [f32; 2],
}

fn workspace_layout(display: [f32; 2]) -> WorkspaceLayout {
    let work_width = (display[0] - MODEL_WIDTH).max(0.);
    let work_height = (display[1] - RIBBON_BOTTOM - STATUS_HEIGHT).max(0.);
    WorkspaceLayout {
        model_position: [0., RIBBON_BOTTOM],
        model_size: [MODEL_WIDTH, work_height],
        document_position: [MODEL_WIDTH, RIBBON_BOTTOM],
        document_size: [work_width, 30.],
        command_position: [MODEL_WIDTH, display[1] - STATUS_HEIGHT - COMMAND_HEIGHT],
        command_size: [work_width, COMMAND_HEIGHT],
    }
}

pub(super) fn draw_status_bar(ui: &Ui, status: Option<&str>) {
    let display = ui.io().display_size;
    ui.window("Status")
        .position([0., display[1] - 28.], Condition::Always)
        .size([display[0], 28.], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| ui.text(status_text(status)));
}

fn status_text(status: Option<&str>) -> &str {
    status.filter(|text| !text.is_empty()).unwrap_or("Ready")
}

pub(super) fn draw_document_tabs(
    ui: &Ui,
    documents: &[crate::GuiDocumentTab],
    selected: &mut Vec<GuiShellAction>,
) {
    if documents.is_empty() {
        return;
    }
    let layout = workspace_layout(ui.io().display_size);
    ui.window("Documents")
        .position(layout.document_position, Condition::Always)
        .size(layout.document_size, Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| {
            let Some(_tabs) = ui.tab_bar("##document-tabs") else {
                return;
            };
            for document in documents {
                draw_document_tab(ui, document, selected);
            }
        });
}

pub(super) fn draw_viewport_overlays(
    ui: &Ui,
    has_document: bool,
    model_open: bool,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    if !has_document {
        return;
    }
    let (origin, size) = viewport_rect(ui.io().display_size, model_open);
    draw_axis_triad(ui, origin, size);
    draw_view_cube(ui, origin, size, selected);
    draw_navigation_bar(ui, origin, size, selected, icons);
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
    let draw = ui.get_window_draw_list();
    draw.add_line(center, tip, color).thickness(2.4).build();
    draw.add_circle(tip, 4.5, color).filled(true).build();
    draw.add_text([tip[0] + 3., tip[1] - 7.], color, label);
}

fn viewport_rect(display: [f32; 2], model_open: bool) -> ([f32; 2], [f32; 2]) {
    let left = if model_open { MODEL_WIDTH } else { 0. };
    let top = RIBBON_BOTTOM + 30.;
    let bottom = display[1] - STATUS_HEIGHT - COMMAND_HEIGHT;
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
}

fn draw_view_cube_faces(ui: &Ui, selected: &mut Vec<GuiShellAction>) {
    let origin = ui.cursor_screen_pos();
    let clicked = ui.invisible_button("##view-cube-home", [VIEW_CUBE_SIZE, VIEW_CUBE_SIZE]);
    draw_cube_faces(ui, origin);
    draw_cube_labels(ui, origin);
    if clicked {
        selected.push(GuiShellAction::Home);
    }
}

fn draw_cube_faces(ui: &Ui, origin: [f32; 2]) {
    let points = cube_points(origin);
    let draw = ui.get_window_draw_list();
    for (face, color) in points.into_iter().zip([
        [0.82, 0.84, 0.88, 0.92],
        [0.65, 0.68, 0.74, 0.92],
        [0.72, 0.75, 0.80, 0.92],
    ]) {
        draw.add_polyline(face.to_vec(), color).filled(true).build();
    }
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
        .build(|| draw_navigation_buttons(ui, selected, icons));
}

fn draw_navigation_buttons(
    ui: &Ui,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    for (label, icon, action, enabled) in navigation_buttons() {
        draw_navigation_button(ui, label, icon, action, enabled, selected, icons);
    }
}

fn navigation_buttons() -> [(&'static str, &'static str, GuiShellAction, bool); 5] {
    [
        ("Home", "home", GuiShellAction::Home, true),
        ("Fit", "zoom-all", GuiShellAction::Home, true),
        ("Zoom Window", "zoom-window", GuiShellAction::Home, false),
        (
            "Constrained Orbit",
            "orbit-constrained",
            GuiShellAction::Home,
            false,
        ),
        ("Look At", "look-at", GuiShellAction::Home, false),
    ]
}

fn draw_navigation_button(
    ui: &Ui,
    label: &str,
    icon: &str,
    action: GuiShellAction,
    enabled: bool,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    let disabled = ui.begin_disabled(!enabled);
    let clicked = icons.texture_id(icon).map_or_else(
        || ui.button(label),
        |id| ui.image_button(format!("##navigation-{label}"), id.into(), [22., 22.]),
    );
    drop(disabled);
    if clicked {
        selected.push(action);
    }
    if ui.is_item_hovered() {
        ui.tooltip_text(label);
    }
}

fn draw_document_tab(
    ui: &Ui,
    document: &crate::GuiDocumentTab,
    selected: &mut Vec<GuiShellAction>,
) {
    let mut open = true;
    let label = document_tab_label(document);
    let active = ui.tab_item_with_opened(label, &mut open).is_some();
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
    ui.window("Model")
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
}

fn draw_model_tree(ui: &Ui, tree: &crate::GuiModelTree, selected: &mut Vec<GuiShellAction>) {
    if !ui.collapsing_header(&tree.document, TreeNodeFlags::DEFAULT_OPEN) {
        return;
    }
    draw_model_origin(ui);
    draw_model_parameters(ui, &tree.parameters);
    draw_model_sketches(ui, tree.sketches);
    draw_model_features(ui, &tree.features);
    draw_model_bodies(ui, &tree.bodies, selected);
}

fn draw_model_origin(ui: &Ui) {
    if ui.collapsing_header("Origin", TreeNodeFlags::empty()) {
        ui.bullet_text("XY Plane");
        ui.bullet_text("XZ Plane");
        ui.bullet_text("YZ Plane");
    }
}

fn draw_model_parameters(ui: &Ui, parameters: &[String]) {
    if !ui.collapsing_header(
        format!("Parameters ({})", parameters.len()),
        TreeNodeFlags::empty(),
    ) {
        return;
    }
    for parameter in parameters {
        ui.bullet_text(parameter);
    }
}

fn draw_model_sketches(ui: &Ui, count: usize) {
    if ui.collapsing_header(format!("Sketches ({count})"), TreeNodeFlags::empty()) {
        for index in 1..=count {
            ui.bullet_text(format!("Sketch {index}"));
        }
    }
}

fn draw_model_features(ui: &Ui, features: &[crate::GuiModelFeature]) {
    if !ui.collapsing_header(
        format!("Features ({})", features.len()),
        TreeNodeFlags::empty(),
    ) {
        return;
    }
    for feature in features {
        ui.bullet_text(model_feature_label(feature));
    }
}

fn model_feature_label(feature: &crate::GuiModelFeature) -> String {
    format!("{} ({}) — {}", feature.name, feature.kind, feature.health)
}

fn draw_model_bodies(ui: &Ui, bodies: &[crate::GuiModelBody], selected: &mut Vec<GuiShellAction>) {
    if !ui.collapsing_header(
        format!("Bodies ({})", bodies.len()),
        TreeNodeFlags::DEFAULT_OPEN,
    ) {
        return;
    }
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
    ui.window("Command")
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
        axis_triad_arrows, cube_points, model_feature_label, navigation_buttons, status_text,
        viewport_rect, workspace_layout,
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
        assert_eq!(layout.model_position, [0., 145.]);
        assert_eq!(layout.model_size, [280., 727.]);
        assert_eq!(layout.document_position, [280., 145.]);
        assert_eq!(layout.command_position, [280., 742.]);
        assert_eq!(layout.command_size, [1160., 130.]);
    }

    #[test]
    fn idle_status_matches_the_go_status_model() {
        assert_eq!(status_text(None), "Ready");
        assert_eq!(status_text(Some("")), "Ready");
        assert_eq!(status_text(Some("Click a face")), "Click a face");
    }

    #[test]
    fn overlays_anchor_to_the_visible_viewport() {
        assert_eq!(
            viewport_rect([1440., 900.], true),
            ([280., 175.], [1160., 567.])
        );
        assert_eq!(
            viewport_rect([1440., 900.], false),
            ([0., 175.], [1440., 567.])
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
    fn navigation_bar_keeps_the_five_go_navigation_tools() {
        let buttons = navigation_buttons();
        assert_eq!(buttons.len(), 5);
        assert_eq!(buttons[0].0, "Home");
        assert_eq!(buttons[4].0, "Look At");
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
        let frame = engine.frame(&request).unwrap().draws;
        assert!(
            frame
                .batches
                .iter()
                .flat_map(|batch| &batch.vertices)
                .any(|vertex| { vertex.srgb == [209, 214, 224, 235] })
        );
    }
}
