// SPDX-License-Identifier: GPL-2.0-only
//! Work Axis interactive command actions, display overlays, and status prompts (#3918).
use crate::{
    GuiAuthoringContext, GuiCommandState, GuiFrameRequest, GuiOriginPlaneKind, GuiShellAction,
};
use imgui::Ui;
use oblikovati_model::feature::{FeatureId, work_axis::WorkAxisReference};

const WORK_AXIS_COLOR: [f32; 4] = [0.85, 0.65, 0.25, 0.9];
const WORK_AXIS_SELECTED_COLOR: [f32; 4] = [0.25, 0.6, 1.0, 1.0];
const WORK_AXIS_THICKNESS: f32 = 1.5;
const WORK_AXIS_SELECTED_THICKNESS: f32 = 3.0;

/// Construction mode chosen from 3D Model > Work Features > Axis.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiWorkAxisMode {
    General,
    OnLineOrEdge,
    ParallelToLineThroughPoint,
    ThroughTwoPoints,
    IntersectionOfTwoPlanes,
    NormalToPlaneThroughPoint,
    ThroughCircularEdge,
    ThroughRevolvedFace,
}

impl GuiWorkAxisMode {
    /// Menu caption matching the 3D Model > Work Features > Axis dropdown.
    /// Example: `GuiWorkAxisMode::ThroughTwoPoints.label()`.
    pub const fn label(self) -> &'static str {
        match self {
            Self::General => "Axis",
            Self::OnLineOrEdge => "On Line or Edge",
            Self::ParallelToLineThroughPoint => "Parallel to Line through Point",
            Self::ThroughTwoPoints => "Through Two Points",
            Self::IntersectionOfTwoPlanes => "Intersection of Two Planes",
            Self::NormalToPlaneThroughPoint => "Normal to Plane through Point",
            Self::ThroughCircularEdge => "Through Center of Circular or Elliptical Edge",
            Self::ThroughRevolvedFace => "Through Revolved Face or Feature",
        }
    }
}

/// Interactive Work Axis command actions dispatched from the ribbon and browser.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiWorkAxisAction {
    Start(GuiWorkAxisMode),
    PickOriginPlane(GuiOriginPlaneKind),
    PickOriginAxis(WorkAxisReference),
    PickOriginPoint,
    Cancel,
}

impl GuiWorkAxisAction {
    /// True when the active document is an editable Part outside a transaction.
    /// Example: `GuiWorkAxisAction::Start(GuiWorkAxisMode::General).enabled(&state)`.
    pub(crate) fn enabled(self, state: &GuiCommandState) -> bool {
        if matches!(self, Self::Cancel) {
            return state.work_axis_prompt.is_some();
        }
        state.authoring == GuiAuthoringContext::Part
            && state.document_type == Some(oblikovati_api::types::DOCUMENT_PART)
            && !state.transaction_open
    }
}

/// One projected user work axis line rendered over the viewport.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiWorkAxisDisplay {
    pub id: FeatureId,
    pub name: String,
    pub ends: [[i32; 2]; 2],
    pub selected: bool,
}

/// Draw visible user work axes and highlight whichever work axis is currently selected.
pub(crate) fn draw_work_axes(ui: &Ui, request: &GuiFrameRequest) {
    if request.commands.work_axes.is_empty() {
        return;
    }
    let scale = request.layout_scale();
    let logical = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
    let list = ui.get_background_draw_list();
    let selected_color = request
        .commands
        .color_scheme
        .as_ref()
        .map_or(WORK_AXIS_SELECTED_COLOR, |scheme| {
            scheme.geometry_selection(false)
        });
    for axis in &request.commands.work_axes {
        let (color, thickness) = if axis.selected {
            (selected_color, WORK_AXIS_SELECTED_THICKNESS)
        } else {
            (WORK_AXIS_COLOR, WORK_AXIS_THICKNESS)
        };
        list.add_line(logical(axis.ends[0]), logical(axis.ends[1]), color)
            .thickness(thickness)
            .build();
    }
}

/// Cancel an active Work Axis prompt when the user presses Escape outside any popup.
pub(crate) fn cancel_on_escape(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    if state.work_axis_prompt.is_some()
        && !crate::widget_abi::any_popup_open(ui)
        && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
    {
        actions.push(GuiShellAction::WorkAxis(GuiWorkAxisAction::Cancel));
    }
}
