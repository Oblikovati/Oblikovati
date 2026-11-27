// SPDX-License-Identifier: GPL-2.0-only
//! Distance A drag for a certified numerical extrusion preview.
use crate::application::document_creation::workspaces::WorkspaceArea;
use crate::{GuiCommandState, GuiParameterEngineering, widget_abi};
use imgui::{Condition, MouseButton, StyleColor, WindowFlags};
use oblikovati_model::quantity::{Dimension, Quantity};

const SIDE: f32 = 18.;
const WINDOW: &str = "##extrude-distance-manipulator";

/// The authored axis and its current viewport projection, in framebuffer pixels.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct GuiExtrudeHandle {
    pub axis: [[f64; 3]; 2],
    pub pixels: Option<[[i32; 2]; 2]>,
    pub depth_ratio: f64,
}
impl GuiExtrudeHandle {
    /// Construct the unprojected control. Example: `GuiExtrudeHandle::new([start, end])`.
    pub fn new(axis: [[f64; 3]; 2]) -> Self {
        Self {
            axis,
            pixels: None,
            depth_ratio: 1.,
        }
    }
}

#[derive(Default)]
pub(crate) struct DistanceHandle {
    drag: Option<Drag>,
}
struct Drag {
    session: (u64, u64),
    points: [[f32; 2]; 2],
    pointer: [f32; 2],
    depth_ratio: f64,
    engineering: GuiParameterEngineering,
    last: String,
}

impl DistanceHandle {
    pub(crate) fn draw(
        &mut self,
        ui: &imgui::Ui,
        state: &GuiCommandState,
        area: WorkspaceArea,
    ) -> Option<String> {
        let session = (state.document?, state.document_generation?);
        if self
            .drag
            .as_ref()
            .is_some_and(|drag| drag.session != session)
        {
            self.drag = None;
        }
        let projection = projection(ui, state).and_then(|(points, ratio, engineering)| {
            let points = clear_properties_panel(points, area, ui.window_pos(), ui.window_size())?;
            Some((points, ratio, engineering))
        });
        let points = self
            .drag
            .as_ref()
            .map(|drag| drag.points)
            .or_else(|| projection.as_ref().map(|(points, _, _)| *points))?;
        let tip = points[1];
        if self.drag.is_none() && !in_workspace(tip, area) {
            return None;
        }
        let label = dimension_label(state, self.drag.as_ref());
        let unit = state
            .document_units
            .as_ref()
            .map_or("", |units| units.length_unit.as_str());
        let caption = format!("A {label} {unit}");
        {
            let _mono = crate::ui_scale::mono_micro_font(ui);
            widget_abi::foreground_dimension(
                ui,
                projection.as_ref().map_or(points, |(p, _, _)| *p),
                crate::theme::selection_accent(ui),
                &caption,
            );
        }
        handle_window(ui, tip, || {
            widget_abi::button(ui, "##distance-a-tip", [SIDE; 2]);
            if ui.is_item_activated()
                && let Some((points, depth_ratio, engineering)) = projection
            {
                self.drag = Some(Drag {
                    session,
                    points,
                    pointer: ui.io().mouse_pos,
                    depth_ratio,
                    engineering: engineering.clone(),
                    last: label,
                });
            }
            let changed = self
                .drag
                .as_mut()
                .and_then(|drag| drag.update(ui.io().mouse_pos));
            if !ui.is_mouse_down(MouseButton::Left) {
                self.drag = None;
            }
            changed
        })
        .flatten()
    }
}

fn projection<'a>(
    ui: &imgui::Ui,
    state: &'a GuiCommandState,
) -> Option<([[f32; 2]; 2], f64, &'a GuiParameterEngineering)> {
    let draft = state.extrude_draft_preview.as_ref()?;
    let validation = state.extrude_distance_validation.as_ref()?;
    if validation.distance != draft.distance {
        return None;
    }
    let handle = state.extrude_handle?;
    let engineering = validation.engineering.as_ref()?;
    let scale = ui.io().display_framebuffer_scale;
    let mut points = handle
        .pixels?
        .map(|[x, y]| [x as f32 / scale[0], y as f32 / scale[1]]);
    let axis = delta(points[1], points[0]);
    if dot(axis, axis) >= 64. {
        return Some((points, handle.depth_ratio, engineering));
    }
    // An end-on view has no useful projected axis. Keep a vertical drag guide at the cap.
    points[0] = [points[1][0], points[1][1] + 48.];
    Some((points, 1., engineering))
}

impl Drag {
    fn update(&mut self, pointer: [f32; 2]) -> Option<String> {
        let change = delta(pointer, self.pointer);
        let axis = delta(self.points[1], self.points[0]);
        let fraction = 1. + f64::from(dot(change, axis) / dot(axis, axis));
        // Undo perspective foreshortening before applying the distance ratio.
        let depth = self.depth_ratio + (1. - self.depth_ratio) * fraction;
        if depth <= 0. {
            return None;
        }
        let length = self.engineering.nominal.value * fraction / depth;
        if !length.is_finite() || length <= 0. {
            return None;
        }
        let text = numeric_distance(&self.engineering, length);
        if text == self.last || text.parse::<f64>().is_ok_and(|value| value <= 0.) {
            return None;
        }
        self.last.clone_from(&text);
        Some(text)
    }
}

fn numeric_distance(source: &GuiParameterEngineering, length: f64) -> String {
    let length = source
        .units
        .to_preferred(Quantity::new(length, Dimension::Length));
    format!("{length:.3}")
        .trim_end_matches('0')
        .trim_end_matches('.')
        .into()
}

fn dimension_label(state: &GuiCommandState, drag: Option<&Drag>) -> String {
    drag.map(|drag| drag.last.clone())
        .or_else(|| {
            let source = state
                .extrude_distance_validation
                .as_ref()?
                .engineering
                .as_ref()?;
            Some(numeric_distance(source, source.nominal.value))
        })
        .unwrap_or_default()
}

fn handle_window<T>(ui: &imgui::Ui, tip: [f32; 2], draw: impl FnOnce() -> T) -> Option<T> {
    let origin = tip.map(|value| value - SIDE / 2.);
    let _min = widget_abi::window_min_size(ui, [1.; 2]);
    let _padding = widget_abi::window_padding(ui, [0.; 2]);
    let _color = widget_abi::push_style_color(ui, StyleColor::Button, [0.; 4]);
    let _hover = widget_abi::push_style_color(ui, StyleColor::ButtonHovered, [0.; 4]);
    let _active = widget_abi::push_style_color(ui, StyleColor::ButtonActive, [0.; 4]);
    widget_abi::window_geometry(ui, origin, [SIDE; 2], Condition::Always);
    ui.window(WINDOW)
        .flags(
            WindowFlags::NO_DECORATION
                | WindowFlags::NO_BACKGROUND
                | WindowFlags::NO_SAVED_SETTINGS
                | WindowFlags::NO_DOCKING
                | WindowFlags::NO_NAV
                | WindowFlags::NO_FOCUS_ON_APPEARING
                | WindowFlags::NO_BRING_TO_FRONT_ON_FOCUS,
        )
        .build(draw)
}

fn clear_properties_panel(
    points: [[f32; 2]; 2],
    area: WorkspaceArea,
    [left, top]: [f32; 2],
    [width, height]: [f32; 2],
) -> Option<[[f32; 2]; 2]> {
    let [x, y] = points[1];
    let clearance = SIDE / 2. + 4.;
    let right = left + width;
    let bottom = top + height;
    if !(left - clearance..right + clearance).contains(&x)
        || !(top - clearance..bottom + clearance).contains(&y)
    {
        return Some(points);
    }
    // Translate the whole projected axis so each pixel still represents the same depth.
    let target = [
        [right + clearance, y],
        [left - clearance, y],
        [x, bottom + clearance],
        [x, top - clearance],
    ]
    .into_iter()
    .filter(|point| in_workspace(*point, area))
    .min_by(|a, b| {
        let shift = |point| {
            let along = delta(point, [x, y]);
            dot(along, along)
        };
        shift(*a).total_cmp(&shift(*b))
    })?;
    let shift = delta(target, [x, y]);
    Some(points.map(|point| [point[0] + shift[0], point[1] + shift[1]]))
}

fn in_workspace([x, y]: [f32; 2], area: WorkspaceArea) -> bool {
    (area.left + SIDE..area.right - SIDE).contains(&x)
        && (area.top + SIDE..area.bottom - SIDE).contains(&y)
}

fn delta(a: [f32; 2], b: [f32; 2]) -> [f32; 2] {
    [a[0] - b[0], a[1] - b[1]]
}
fn dot(a: [f32; 2], b: [f32; 2]) -> f32 {
    a[0] * b[0] + a[1] * b[1]
}
