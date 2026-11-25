// SPDX-License-Identifier: GPL-2.0-only
//! Choosing an origin plane to sketch on: Start 2D Sketch, or Extrude with nothing to extrude.
#[path = "plane_label.rs"]
mod plane_label;
use crate::application::document_creation::workspaces::WorkspaceArea;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{MouseButton, Ui};
use oblikovati_api::types::{TOKEN_PLANE_FAINT, TOKEN_PLANE_FILL};
use plane_label::PlaneLabel;

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
    /// Scene-sized outline for an automatically resized browser datum.
    pub automatic_corners: [[i32; 2]; 4],
    /// Scene-sized outline used while choosing a sketch plane.
    pub picker_corners: [[i32; 2]; 4],
}
/// Translucent and warm, so the planes read over a dark view without hiding it.
const PLANE_FILL: [f32; 4] = [245. / 255., 158. / 255., 11. / 255., 0.14];
const PLANE_EDGE: [f32; 4] = [245. / 255., 158. / 255., 11. / 255., 1.];

/// Whether the display is up; while it is, Start 2D Sketch stays highlighted on the ribbon.
#[derive(Default)]
pub(crate) struct PlaneSelection {
    pub active: bool,
}
impl PlaneSelection {
    pub(crate) fn take_browser_choice(
        &mut self,
        workspace: &mut crate::application::workspace_layout::WorkspaceLayout,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.active
            && let Some(plane) = workspace.take_selected_origin_plane()
        {
            self.active = false;
            actions.push(GuiShellAction::StartSketchOn(plane));
        }
    }

    /// Draw the planes and take a choice; Escape closes without starting anything.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        scale: [f32; 2],
        area: WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.active {
            return;
        }
        if state.document.is_none()
            || state.origin_planes.is_empty()
            || state.authoring == crate::GuiAuthoringContext::Sketch
        {
            self.active = false;
            return;
        }
        let planes: Vec<PickerPlane> = state
            .origin_planes
            .iter()
            .map(|plane| PickerPlane::projected(ui, plane, scale, area, state))
            .collect();
        draw_planes(ui, &planes, area, state);
        if ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.active = false;
            return;
        }
        if let Some((point, origin_plane)) = plane_chosen_over_workspace(ui, &planes, area) {
            actions.push(GuiShellAction::StartSketchAt {
                document: state.document.unwrap(),
                pixel: [(point[0] * scale[0]) as i32, (point[1] * scale[1]) as i32],
                origin_plane,
            });
        }
    }
}
/// A clear window over the document area owns the click, so the view never reads it as well.
fn plane_chosen_over_workspace(
    ui: &Ui,
    planes: &[PickerPlane],
    area: WorkspaceArea,
) -> Option<([f32; 2], Option<GuiOriginPlaneKind>)> {
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
                .then(|| (ui.io().mouse_pos, chosen_plane(planes, ui.io().mouse_pos)))
        })
        .flatten()
}
struct PickerPlane {
    corners: [[f32; 2]; 4],
    kind: GuiOriginPlaneKind,
    label: Option<PlaneLabel>,
}

impl PickerPlane {
    fn projected(
        ui: &Ui,
        plane: &GuiOriginPlane,
        scale: [f32; 2],
        area: WorkspaceArea,
        state: &GuiCommandState,
    ) -> Self {
        let corners = plane
            .picker_corners
            .map(|corner| [corner[0] as f32 / scale[0], corner[1] as f32 / scale[1]]);
        let facing = state.view_axes.map_or(1., |axes| match plane.kind {
            GuiOriginPlaneKind::Xy => axes.toward[2],
            GuiOriginPlaneKind::Xz => axes.toward[1],
            GuiOriginPlaneKind::Yz => axes.toward[0],
        });
        let [width, height] = ui.calc_text_size(plane.kind.label());
        let label = PlaneLabel::place(corners, [width + 8., height + 4.], facing, area);
        Self {
            corners,
            kind: plane.kind,
            label,
        }
    }
}

fn draw_planes(ui: &Ui, planes: &[PickerPlane], workspace: WorkspaceArea, state: &GuiCommandState) {
    // The background list clips at the viewport perimeter and also runs beneath the HUD.
    let draw = ui.get_background_draw_list();
    let surface = ui.style_color(imgui::StyleColor::WindowBg);
    let [mut fill, edge] = plane_colors(state, surface[0] > 0.5);
    fill[3] = display_fill_alpha(fill, reference_backdrop(state, surface));
    // Fully overlapping planes share a single opacity budget.
    let sheets = planes
        .iter()
        .filter(|plane| area(&plane.corners) > 1.)
        .count()
        .max(1);
    fill[3] = 1. - (1. - fill[3]).powf(1. / sheets as f32);
    let source = [
        (workspace.left + workspace.right) / 2.,
        (workspace.top + workspace.bottom) / 2.,
    ];
    draw.with_clip_rect(
        [workspace.left, workspace.top],
        [workspace.right, workspace.bottom],
        || {
            for plane in planes {
                draw_quad(&draw, plane.corners, fill, edge);
            }
            for plane in planes {
                if let Some(label) = &plane.label {
                    draw_label(ui, &draw, source, plane.kind.label(), label);
                }
            }
        },
    );
}

fn reference_backdrop(state: &GuiCommandState, [r, g, b, _]: [f32; 4]) -> [f32; 3] {
    use oblikovati_viewport::ViewportBackground;
    match state
        .color_scheme
        .as_ref()
        .map(|scheme| scheme.background())
    {
        Some(ViewportBackground::Solid(rgb)) => rgb.map(|channel| f32::from(channel) / 255.),
        Some(ViewportBackground::VerticalGradient { top, bottom }) => {
            std::array::from_fn(|axis| (f32::from(top[axis]) + f32::from(bottom[axis])) / 510.)
        }
        None => [r, g, b],
    }
}

fn display_fill_alpha([r, g, b, opacity]: [f32; 4], backdrop: [f32; 3]) -> f32 {
    use crate::color_scheme_state::linear_channel;
    // The design opacity is authored in display RGB. Calibrate its dominant
    // channel against the backdrop before the renderer blends in linear light.
    let Some((base, ink)) = backdrop
        .into_iter()
        .zip([r, g, b])
        .max_by(|(a, b), (c, d)| (b - a).abs().total_cmp(&(d - c).abs()))
    else {
        return opacity;
    };
    let range = linear_channel(ink) - linear_channel(base);
    if range.abs() < f32::EPSILON {
        return opacity;
    }
    let display_mix = base + (ink - base) * opacity;
    ((linear_channel(display_mix) - linear_channel(base)) / range).clamp(0., 1.)
}

fn draw_label(
    ui: &Ui,
    draw: &imgui::DrawListMut<'_>,
    source: [f32; 2],
    text: &str,
    label: &PlaneLabel,
) {
    let mut backing = ui.style_color(imgui::StyleColor::PopupBg);
    backing[3] = label.opacity;
    draw.add_polyline(label.footprint().to_vec(), backing)
        .filled(true)
        .build();
    let mut ink = ui.style_color(imgui::StyleColor::Text);
    ink[3] *= label.opacity;
    crate::widget_abi::projected_background_text(ui, draw, source, ink, text, |point| {
        label.point(point)
    });
}

fn plane_colors(state: &GuiCommandState, light: bool) -> [[f32; 4]; 2] {
    let [mut fill, edge] = state
        .theme
        .as_ref()
        .map_or([PLANE_FILL, PLANE_EDGE], |theme| {
            [TOKEN_PLANE_FILL, TOKEN_PLANE_FAINT].map(|token| {
                let color = &theme.palette()[&token];
                [color.r, color.g, color.b, color.a]
            })
        });
    fill[3] = if light { 0.12 } else { 0.14 };
    [fill, edge]
}
fn draw_quad(
    draw: &imgui::DrawListMut<'_>,
    mut corners: [[f32; 2]; 4],
    fill: [f32; 4],
    edge: [f32; 4],
) {
    // The antialias fringe assumes clockwise winding in screen coordinates.
    if signed_edge(corners[0], corners[1], corners[2]) < 0. {
        corners.reverse();
    }
    draw.add_polyline(corners.to_vec(), fill)
        .filled(true)
        .build();
    draw.add_polyline(
        [corners[0], corners[1], corners[2], corners[3], corners[0]].to_vec(),
        edge,
    )
    .thickness(1.25)
    .build();
}
/// Prefer the rendered label among overlaps, then pick the most face-on surface.
fn chosen_plane(planes: &[PickerPlane], pointer: [f32; 2]) -> Option<GuiOriginPlaneKind> {
    if let Some(plane) = planes
        .iter()
        .find(|plane| plane.label.is_some_and(|label| label.contains(pointer)))
    {
        return Some(plane.kind);
    }
    planes
        .iter()
        .filter(|plane| inside(&plane.corners, pointer))
        .map(|plane| (plane.kind, area(&plane.corners)))
        .filter(|&(_, area)| area > 1.)
        .max_by(|(_, a), (_, b)| a.total_cmp(b))
        .map(|(kind, _)| kind)
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
