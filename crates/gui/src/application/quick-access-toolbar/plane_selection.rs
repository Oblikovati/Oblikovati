// SPDX-License-Identifier: GPL-2.0-only
//! Choosing an origin plane to sketch on: Start 2D Sketch, or Extrude with nothing to extrude.
use crate::application::document_creation::workspaces::WorkspaceArea;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, Ui};

/// One origin plane; each is a sketch plane the owner can start on.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiOriginPlaneKind {
    Xy,
    Xz,
    Yz,
}
impl GuiOriginPlaneKind {
    /// Name the plane as the browser does; example: `GuiOriginPlaneKind::Xy.label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Xy => "XY Plane",
            Self::Xz => "XZ Plane",
            Self::Yz => "YZ Plane",
        }
    }
}
/// A plane's outline in window pixels, projected by the owner's current camera.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiOriginPlane {
    pub kind: GuiOriginPlaneKind,
    pub corners: [[i32; 2]; 4],
}
/// Translucent and warm, so the planes read over a dark view without hiding it.
const PLANE_FILL: [f32; 4] = [0.95, 0.65, 0.25, 0.22];
const PLANE_EDGE: [f32; 4] = [0.95, 0.65, 0.25, 0.9];
const LABEL: [f32; 4] = [1., 0.9, 0.7, 1.];

/// Whether the display is up; while it is, Start 2D Sketch stays highlighted on the ribbon.
#[derive(Default)]
pub(crate) struct PlaneSelection {
    pub active: bool,
}
impl PlaneSelection {
    /// Draw the planes and take a choice; Escape closes without starting anything.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        scale: [f32; 2],
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.active {
            return;
        }
        if state.document.is_none() || state.origin_planes.is_empty() {
            self.active = false;
            return;
        }
        let planes: Vec<([[f32; 2]; 4], GuiOriginPlaneKind)> = state
            .origin_planes
            .iter()
            .map(|plane| (logical_corners(plane.corners, scale), plane.kind))
            .collect();
        draw_planes(ui, &planes, area);
        if ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.active = false;
            return;
        }
        if let Some(kind) = plane_chosen_over_workspace(ui, &planes, area) {
            actions.push(GuiShellAction::StartSketchOn(kind));
            self.active = false;
        }
    }
}
/// A clear window over the document area owns the click, so the view never reads it as well.
fn plane_chosen_over_workspace(
    ui: &Ui,
    planes: &[([[f32; 2]; 4], GuiOriginPlaneKind)],
    area: WorkspaceArea,
) -> Option<GuiOriginPlaneKind> {
    crate::widget_abi::window_geometry(
        ui,
        [area.left, area.top],
        [area.right - area.left, area.bottom - area.top],
        imgui::Condition::Always,
    );
    ui.window("##plane-selection")
        .flags(
            imgui::WindowFlags::NO_DECORATION
                | imgui::WindowFlags::NO_BACKGROUND
                | imgui::WindowFlags::NO_MOVE
                | imgui::WindowFlags::NO_SAVED_SETTINGS
                | imgui::WindowFlags::NO_BRING_TO_FRONT_ON_FOCUS,
        )
        .build(|| {
            (ui.is_window_hovered() && ui.is_mouse_clicked(MouseButton::Left))
                .then(|| chosen_plane(ui, planes, area))
                .flatten()
        })
        .flatten()
}
fn logical_corners(corners: [[i32; 2]; 4], scale: [f32; 2]) -> [[f32; 2]; 4] {
    corners.map(|corner| [corner[0] as f32 / scale[0], corner[1] as f32 / scale[1]])
}
fn draw_planes(ui: &Ui, planes: &[([[f32; 2]; 4], GuiOriginPlaneKind)], area: WorkspaceArea) {
    let draw = ui.get_foreground_draw_list();
    for (corners, kind) in planes {
        draw.add_triangle(corners[0], corners[1], corners[2], PLANE_FILL)
            .filled(true)
            .build();
        draw.add_triangle(corners[0], corners[2], corners[3], PLANE_FILL)
            .filled(true)
            .build();
        for index in 0..4 {
            draw.add_line(corners[index], corners[(index + 1) % 4], PLANE_EDGE)
                .build();
        }
        draw.add_text(
            label_position(corners, ui.calc_text_size(kind.label()), area),
            LABEL,
            kind.label(),
        );
    }
}
pub(crate) fn label_position(
    corners: &[[f32; 2]; 4],
    size: [f32; 2],
    area: WorkspaceArea,
) -> [f32; 2] {
    let projected = [
        corners[2][0] * 0.8 + corners[0][0] * 0.2,
        corners[2][1] * 0.8 + corners[0][1] * 0.2,
    ];
    [
        projected[0].clamp(area.left, (area.right - size[0]).max(area.left)),
        projected[1].clamp(area.top, (area.bottom - size[1]).max(area.top)),
    ]
}
/// A label is always unambiguous; inside overlapping planes the most face-on one wins.
fn chosen_plane(
    ui: &Ui,
    planes: &[([[f32; 2]; 4], GuiOriginPlaneKind)],
    workspace: WorkspaceArea,
) -> Option<GuiOriginPlaneKind> {
    let pointer = ui.io().mouse_pos;
    for (corners, kind) in planes {
        let size = ui.calc_text_size(kind.label());
        let at = label_position(corners, size, workspace);
        if (at[0]..=at[0] + size[0]).contains(&pointer[0])
            && (at[1]..=at[1] + size[1]).contains(&pointer[1])
        {
            return Some(*kind);
        }
    }
    planes
        .iter()
        .filter(|(corners, _)| inside(corners, pointer))
        .max_by(|(first, _), (second, _)| area(first).total_cmp(&area(second)))
        .map(|(_, kind)| *kind)
}
fn signed_edge(from: [f32; 2], to: [f32; 2], at: [f32; 2]) -> f32 {
    (to[0] - from[0]) * (at[1] - from[1]) - (to[1] - from[1]) * (at[0] - from[0])
}
/// Inside a convex outline: on the same side of all four edges.
pub(crate) fn inside(corners: &[[f32; 2]; 4], at: [f32; 2]) -> bool {
    let signs: Vec<f32> = (0..4)
        .map(|index| signed_edge(corners[index], corners[(index + 1) % 4], at))
        .collect();
    signs.iter().all(|sign| *sign >= 0.) || signs.iter().all(|sign| *sign <= 0.)
}
pub(crate) fn area(corners: &[[f32; 2]; 4]) -> f32 {
    let mut twice = 0.;
    for index in 0..4 {
        let (a, b) = (corners[index], corners[(index + 1) % 4]);
        twice += a[0] * b[1] - b[0] * a[1];
    }
    twice.abs() * 0.5
}
