// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiShellAction;
use imgui::{Condition, Ui, WindowFlags};

use super::workspace_layout;

const VIEW_CUBE_SIZE: f32 = 86.;
const VIEW_CUBE_EDGE_BAND: f32 = 0.2;
const VIEW_CUBE_RADIUS: f32 = 36.;
const VIEW_CUBE_MARGIN_RATIO: f32 = 2.4;
const VIEW_CUBE_REACH_RATIO: f32 = 1.74;
const VIEW_CUBE_HOME_GAP_RATIO: f32 = 0.22;
const VIEW_CUBE_HOME_RADIUS_RATIO: f32 = 0.36;
const VIEW_CUBE_HOME_SIZE: f32 = VIEW_CUBE_RADIUS * VIEW_CUBE_HOME_RADIUS_RATIO * 2.;
const VIEWPORT_CONTENT_LEFT_INSET: f32 = 8.;
const VIEWPORT_CONTENT_RIGHT_INSET: f32 = 5.;
const VIEWPORT_CONTENT_BOTTOM_INSET: f32 = 8.;
const NAVIGATION_BAR_HEIGHT: f32 = 146.;
const AXIS_TRIAD_MARGIN: f32 = 46.;
const AXIS_TRIAD_RADIUS: f32 = 30.;
pub(super) const AXIS_ARROW_HEAD: f32 = 9.;

/// Draw viewport orientation and navigation controls; example: `draw(ui, true, true, actions, icons, state)`.
pub(crate) fn draw(
    ui: &Ui,
    has_document: bool,
    model_open: bool,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
) {
    if !has_document {
        return;
    }
    let (origin, size) = viewport_rect(ui.io().display_size, model_open);
    draw_axis_triad(ui, origin, size);
    draw_view_cube(ui, origin, size, selected);
    draw_navigation_bar(ui, origin, size, selected, icons, workspace);
}

pub(super) fn viewport_rect(display: [f32; 2], model_open: bool) -> ([f32; 2], [f32; 2]) {
    let layout = workspace_layout(display);
    let left = if model_open { layout.model_size[0] } else { 0. } + VIEWPORT_CONTENT_LEFT_INSET;
    let top = layout.document_tabs_position[1] + layout.document_tabs_size[1];
    let right = (display[0] - VIEWPORT_CONTENT_RIGHT_INSET).max(left);
    let bottom = (layout.command_position[1] - VIEWPORT_CONTENT_BOTTOM_INSET).max(top);
    (
        [left, top],
        [(right - left).max(0.), (bottom - top).max(0.)],
    )
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

pub(super) fn axis_triad_arrows() -> [([f32; 2], [f32; 4], &'static str); 3] {
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

pub(super) fn axis_arrowhead(center: [f32; 2], tip: [f32; 2]) -> Option<[[f32; 2]; 3]> {
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

fn draw_view_cube(ui: &Ui, origin: [f32; 2], size: [f32; 2], selected: &mut Vec<GuiShellAction>) {
    let position = view_cube_position(origin, size);
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

pub(super) fn view_cube_home_position(cube: [f32; 2]) -> [f32; 2] {
    let center = [cube[0] + VIEW_CUBE_SIZE / 2., cube[1] + VIEW_CUBE_SIZE / 2.];
    let home_offset = VIEW_CUBE_RADIUS
        * (VIEW_CUBE_REACH_RATIO + VIEW_CUBE_HOME_GAP_RATIO + VIEW_CUBE_HOME_RADIUS_RATIO);
    [
        center[0] - VIEW_CUBE_RADIUS - VIEW_CUBE_HOME_SIZE / 2.,
        center[1] + home_offset - VIEW_CUBE_HOME_SIZE / 2.,
    ]
}

pub(super) fn view_cube_position(origin: [f32; 2], size: [f32; 2]) -> [f32; 2] {
    let center = [
        origin[0] + size[0] - VIEW_CUBE_RADIUS * VIEW_CUBE_MARGIN_RATIO,
        origin[1] + VIEW_CUBE_RADIUS * VIEW_CUBE_MARGIN_RATIO,
    ];
    [
        center[0] - VIEW_CUBE_SIZE / 2.,
        center[1] - VIEW_CUBE_SIZE / 2.,
    ]
}

fn draw_view_cube_home_button(ui: &Ui, selected: &mut Vec<GuiShellAction>) {
    let origin = ui.cursor_screen_pos();
    let clicked = ui.invisible_button(
        "##view-cube-go-home",
        [VIEW_CUBE_HOME_SIZE, VIEW_CUBE_HOME_SIZE],
    );
    draw_home_icon(ui, origin);
    if clicked {
        selected.push(GuiShellAction::Home);
    }
}

fn draw_home_icon(ui: &Ui, origin: [f32; 2]) {
    let color = [0.72, 0.76, 0.82, 0.95];
    let draw = ui.get_window_draw_list();
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
}

fn draw_cube_faces(ui: &Ui, origin: [f32; 2]) {
    let points = cube_points(origin);
    draw_cube_compass(ui, origin);
    draw_cube_face_fills(ui, points);
    draw_cube_grids(ui, points);
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

pub(super) fn cube_grid_fractions() -> [f32; 2] {
    [VIEW_CUBE_EDGE_BAND, 1. - VIEW_CUBE_EDGE_BAND]
}

pub(super) fn quad_point(quad: [[f32; 2]; 4], horizontal: f32, vertical: f32) -> [f32; 2] {
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

pub(super) fn cube_points(origin: [f32; 2]) -> [[[f32; 2]; 4]; 3] {
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
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
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
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
) {
    for button in &navigation_buttons() {
        draw_navigation_button(ui, button, selected, icons, workspace);
    }
}

pub(super) struct NavigationButton {
    pub(super) label: &'static str,
    icon: &'static str,
    action: GuiShellAction,
    enabled: bool,
}

pub(super) fn navigation_buttons() -> [NavigationButton; 5] {
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
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
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
