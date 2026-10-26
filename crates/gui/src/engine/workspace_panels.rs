// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiShellAction;
use imgui::{Condition, StyleColor, StyleVar, Ui, WindowFlags};

use super::workspace_shell::RIBBON_BOTTOM;

mod model_browser;
mod viewport_overlays;

// Go's DockBuilder splits the work area 22% left and 7% down after fixed chrome claims its bands.
const MODEL_WIDTH_FRACTION: f32 = 0.22;
const COMMAND_HEIGHT_FRACTION: f32 = 0.07;
const COMMAND_MIN_HEIGHT: f32 = 43.;
// Go's status side bar is one 16 px text line with the default 8 px top/bottom padding.
const STATUS_HEIGHT: f32 = 32.;
const VIEWPORT_HEADER_HEIGHT: f32 = 22.;
// A tab strip lives inside the viewport window: 8 px top/bottom padding plus its 18 px tab row.
const DOCUMENT_TABS_HEIGHT: f32 = 34.;
pub(super) const GO_DOCK_ACCENT: [f32; 4] = [0.506, 0.376, 0.89, 1.];
const GO_DOCK_HEADER: [f32; 4] = [0.129, 0.129, 0.129, 1.];
const DOCK_TAB_LEFT_INSET: f32 = 26.;
const DOCK_TAB_TOP_INSET: f32 = 1.;
const DOCK_TAB_TEXT_INSET: f32 = 5.;
const DOCK_TAB_RIGHT_INSET: f32 = 13.;
const DOCK_TAB_CLOSE_RESERVE: f32 = 10.;

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

pub(super) use viewport_overlays::draw as draw_viewport_overlays;

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

pub(super) use model_browser::draw as draw_model_browser;

pub(super) fn draw_command_window(
    ui: &Ui,
    workspace: &mut super::workspace_shell::WorkspaceShellState,
) {
    let layout = workspace_layout(ui.io().display_size);
    let _title = ui.push_style_color(StyleColor::TitleBg, GO_DOCK_ACCENT);
    let _active = ui.push_style_color(StyleColor::TitleBgActive, GO_DOCK_ACCENT);
    ui.window("Command")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .opened(&mut workspace.command_panel_open)
        .position(layout.command_position, Condition::Always)
        .size(layout.command_size, Condition::Always)
        .movable(false)
        .resizable(false)
        .build(|| {
            let _padding = ui.push_style_var(StyleVar::WindowPadding([8., 0.]));
            let input_position = ui.cursor_screen_pos();
            ui.set_cursor_screen_pos([input_position[0], input_position[1] - 8.]);
            if workspace.command_focus_next {
                ui.set_keyboard_focus_here();
                workspace.command_focus_next = false;
            }
            ui.set_next_item_width(-1.);
            ui.input_text("##command-input", &mut workspace.command_input)
                .build();
        });
}

#[cfg(test)]
mod tests {
    use super::{dock_tab_bounds, status_text, viewport_overlays, workspace_layout};
    use crate::{GuiCommandState, GuiDocumentTab, GuiFrameEngine, GuiFrameRequest};

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
            ([26., 193.], [94., 214.])
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
            viewport_overlays::viewport_rect([1440., 900.], true),
            ([324.8, 248.], [1110.2, 564.68])
        );
        assert_eq!(
            viewport_overlays::viewport_rect([1440., 900.], false),
            ([8., 248.], [1427., 564.68])
        );
    }

    #[test]
    fn view_cube_faces_share_the_front_top_right_corner() {
        let faces = viewport_overlays::cube_points([100., 200.]);
        assert_eq!(faces[0][2], [143., 245.]);
        assert_eq!(faces[1][1], [143., 245.]);
        assert_eq!(faces[2][0], [143., 245.]);
    }

    #[test]
    fn view_cube_keeps_the_go_thin_edge_grid() {
        assert_eq!(viewport_overlays::cube_grid_fractions(), [0.2, 0.8]);
        assert_eq!(
            viewport_overlays::quad_point([[0., 0.], [10., 0.], [10., 10.], [0., 10.]], 0.2, 0.8),
            [2., 8.]
        );
    }

    #[test]
    fn view_cube_home_button_sits_below_the_compass() {
        assert_eq!(
            viewport_overlays::view_cube_home_position([100., 200.]),
            [94.04, 313.56003]
        );
    }

    #[test]
    fn view_cube_position_uses_the_go_radius_and_corner_inset() {
        assert_eq!(
            viewport_overlays::view_cube_position([176., 248.], [624., 277.]),
            [670.6, 291.4]
        );
    }

    #[test]
    fn navigation_bar_keeps_the_five_go_navigation_tools() {
        let buttons = viewport_overlays::navigation_buttons();
        assert_eq!(buttons.len(), 5);
        assert_eq!(buttons[0].label, "Home");
        assert_eq!(buttons[4].label, "Look At");
    }

    #[test]
    fn axis_triad_keeps_the_go_xyz_order_and_colours() {
        let arrows = viewport_overlays::axis_triad_arrows();
        assert_eq!(arrows.map(|arrow| arrow.2), ["X", "Y", "Z"]);
        assert_eq!(arrows[0].1, [0.90, 0.23, 0.27, 1.]);
        assert_eq!(arrows[1].1, [0.42, 0.74, 0.28, 1.]);
        assert_eq!(arrows[2].1, [0.24, 0.50, 0.92, 1.]);
    }

    #[test]
    fn axis_arrowhead_keeps_the_go_length_and_width() {
        let head = viewport_overlays::axis_arrowhead([0., 0.], [30., 0.]).unwrap();
        assert_eq!(head[0], [30., 0.]);
        assert_eq!(head[1][0], 21.);
        assert_eq!(head[2][0], 21.);
        assert_eq!(head[1][1], viewport_overlays::AXIS_ARROW_HEAD * 0.45);
        assert_eq!(head[2][1], -viewport_overlays::AXIS_ARROW_HEAD * 0.45);
        assert_eq!(viewport_overlays::axis_arrowhead([2., 2.], [2., 2.]), None);
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
