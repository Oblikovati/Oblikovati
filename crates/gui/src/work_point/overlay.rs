// SPDX-License-Identifier: GPL-2.0-only
//! Viewport work-point markers, 3D move triad, and Grounded Point coordinate HUD (#3918).
use crate::{
    GuiCommandState, GuiFrameRequest, GuiShellAction,
    application::document_creation::workspaces::WorkspaceArea,
    dockable_window::DockableWindow,
    widget_abi,
    work_point::{GuiGroundedPointState, GuiWorkPointAction},
};
use imgui::{Condition, Ui, WindowFlags};

const MARKER_COLOR: [f32; 4] = [0.95, 0.78, 0.20, 1.0];
const SELECTED_MARKER_COLOR: [f32; 4] = [0.25, 0.78, 1.0, 1.0];
const X_AXIS_COLOR: [f32; 4] = [0.92, 0.28, 0.25, 1.0];
const Y_AXIS_COLOR: [f32; 4] = [0.24, 0.80, 0.36, 1.0];
const Z_AXIS_COLOR: [f32; 4] = [0.25, 0.56, 0.96, 1.0];
const MARKER_HALF: f32 = 5.0;
const GROUNDED_WINDOW: &str = "Grounded Point";

/// Persistent input buffers for the Grounded Point coordinate HUD.
#[derive(Default)]
pub(crate) struct GroundedPointHud {
    dock_state: crate::dockable_lifecycle::DockableState,
    seed: Option<[f64; 3]>,
    fields: [String; 3],
}

impl DockableWindow for GroundedPointHud {}

impl GroundedPointHud {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(GROUNDED_WINDOW, &self.dock_state)
    }

    pub(crate) fn synchronize(&mut self, state: &GuiCommandState) {
        let active = state
            .work_point
            .as_ref()
            .and_then(|wp| wp.grounded.as_ref())
            .is_some();
        self.dock_state.set_open(active);
    }

    /// Draw the Grounded Point floating coordinate HUD when a vertex has been placed.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        commands: &GuiCommandState,
        area: WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(state) = commands.work_point.as_ref() else {
            self.seed = None;
            return;
        };
        let Some(grounded) = state.grounded.as_ref() else {
            self.seed = None;
            return;
        };
        let scale = unit_display_scale(commands);
        let unit_label = unit_display_suffix(commands);
        if self.seed != Some(grounded.coordinates) {
            self.seed = Some(grounded.coordinates);
            for (index, field) in self.fields.iter_mut().enumerate() {
                *field = format!("{:.3}", grounded.coordinates[index] * scale);
            }
        }
        let pos = hud_position(area, grounded);
        widget_abi::window_position(ui, pos, Condition::Appearing);
        let flags = WindowFlags::ALWAYS_AUTO_RESIZE | WindowFlags::NO_SAVED_SETTINGS;
        let mut open = true;
        Self::dockable_window(ui, GROUNDED_WINDOW, &mut open)
            .flags(flags)
            .build(|| {
                ui.text("Move Triad / Coordinates");
                ui.separator();
                let mut edited = false;
                let labels = ["X", "Y", "Z"];
                let mut parsed = grounded.coordinates;
                let mut all_valid = true;
                for (axis, label) in labels.iter().enumerate() {
                    ui.text(label);
                    ui.same_line();
                    ui.set_next_item_width(88.0);
                    let id = format!("##grounded-{label}");
                    if ui.input_text(&id, &mut self.fields[axis]).build() {
                        edited = true;
                    }
                    ui.same_line();
                    ui.text(unit_label);
                    if let Ok(value) = self.fields[axis].trim().parse::<f64>()
                        && value.is_finite()
                    {
                        parsed[axis] = value / scale;
                    } else {
                        all_valid = false;
                    }
                }
                if edited && all_valid {
                    self.seed = Some(parsed);
                    actions.push(GuiShellAction::WorkPoint(
                        GuiWorkPointAction::UpdateGroundedCoords(parsed),
                    ));
                }
                ui.separator();
                let _disabled = ui.begin_disabled(!all_valid);
                if widget_abi::button(ui, "OK", [68.0, 24.0]) && all_valid {
                    actions.push(GuiShellAction::WorkPoint(
                        GuiWorkPointAction::AcceptGrounded {
                            coordinates: parsed,
                        },
                    ));
                }
                drop(_disabled);
                ui.same_line();
                if widget_abi::button(ui, "Cancel", [68.0, 24.0]) {
                    actions.push(GuiShellAction::WorkPoint(GuiWorkPointAction::Cancel));
                }
            });
        if !open {
            actions.push(GuiShellAction::WorkPoint(GuiWorkPointAction::Cancel));
        }
    }
}

/// Draw visible evaluated Work Point markers and the active Grounded Point 3D move triad.
pub(crate) fn draw_viewport_overlays(ui: &Ui, request: &GuiFrameRequest) {
    let scale = request.layout_scale();
    let logical = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
    let list = ui.get_background_draw_list();
    for marker in &request.commands.work_points {
        let Some(pixel) = marker.pixel else {
            continue;
        };
        let center = logical(pixel);
        let color = if marker.selected {
            SELECTED_MARKER_COLOR
        } else {
            MARKER_COLOR
        };
        let r = if marker.selected {
            MARKER_HALF + 1.5
        } else {
            MARKER_HALF
        };
        list.add_line(
            [center[0] - r, center[1] - r],
            [center[0] + r, center[1] + r],
            color,
        )
        .thickness(2.0)
        .build();
        list.add_line(
            [center[0] - r, center[1] + r],
            [center[0] + r, center[1] - r],
            color,
        )
        .thickness(2.0)
        .build();
        list.add_circle(center, 2.5, color).filled(true).build();
    }
    if let Some(grounded) = request
        .commands
        .work_point
        .as_ref()
        .and_then(|wp| wp.grounded.as_ref())
    {
        draw_move_triad(&list, grounded, logical);
    }
}

fn draw_move_triad(
    list: &imgui::DrawListMut<'_>,
    grounded: &GuiGroundedPointState,
    logical: impl Fn([i32; 2]) -> [f32; 2],
) {
    let (Some(origin_px), Some(axis_px)) = (grounded.screen_origin, grounded.screen_axes) else {
        return;
    };
    let origin = logical(origin_px);
    let colors = [X_AXIS_COLOR, Y_AXIS_COLOR, Z_AXIS_COLOR];
    let labels = ["X", "Y", "Z"];
    for ((end_px, color), label) in axis_px.into_iter().zip(colors).zip(labels) {
        let end = logical(end_px);
        list.add_line(origin, end, color).thickness(2.5).build();
        list.add_circle(end, 3.5, color).filled(true).build();
        list.add_text([end[0] + 4.0, end[1] - 6.0], color, label);
    }
    list.add_circle(origin, 4.5, SELECTED_MARKER_COLOR)
        .filled(true)
        .build();
}

fn hud_position(area: WorkspaceArea, grounded: &GuiGroundedPointState) -> [f32; 2] {
    let fallback = [area.left + 24.0, area.top + 24.0];
    let Some(origin) = grounded.screen_origin else {
        return fallback;
    };
    let x = (origin[0] as f32 + 42.0)
        .clamp(area.left + 12.0, (area.right - 220.0).max(area.left + 12.0));
    let y = (origin[1] as f32 - 48.0)
        .clamp(area.top + 12.0, (area.bottom - 160.0).max(area.top + 12.0));
    [x, y]
}

fn unit_display_scale(commands: &GuiCommandState) -> f64 {
    match commands
        .document_units
        .as_ref()
        .map(|u| u.length_unit.as_str())
    {
        Some("mm") => 10.0,
        Some("m") => 0.01,
        Some("in") => 1.0 / 2.54,
        _ => 1.0,
    }
}

fn unit_display_suffix(commands: &GuiCommandState) -> &str {
    commands
        .document_units
        .as_ref()
        .map(|u| u.length_unit.as_str())
        .unwrap_or("cm")
}
