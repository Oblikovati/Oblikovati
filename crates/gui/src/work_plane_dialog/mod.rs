// SPDX-License-Identifier: GPL-2.0-only
//! Interactive Work Plane construction state machine and in-canvas angle input (#3918).

pub mod types;

use crate::{
    GuiCommandState, GuiShellAction, application::document_creation::workspaces::WorkspaceArea,
};
use imgui::Ui;
use types::{GuiWorkPlaneAction, GuiWorkPlaneMode, GuiWorkPlanePick, GuiWorkPlaneRequest};

const ANGLE_WINDOW: &str = "Work Plane";
const ANGLE_WIDTH: f32 = 240.;

#[derive(Default)]
pub(crate) struct WorkPlaneDialog {
    document: Option<u64>,
    mode: Option<GuiWorkPlaneMode>,
    picks: Vec<GuiWorkPlanePick>,
    angle: String,
    angle_focused: bool,
    last_pick_seq: u64,
    last_preview: Option<GuiWorkPlaneRequest>,
    error: Option<String>,
}

impl WorkPlaneDialog {
    pub(crate) fn active(&self) -> bool {
        self.document.is_some() && self.mode.is_some()
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, mode: GuiWorkPlaneMode) {
        self.document = state.document;
        self.mode = Some(mode);
        self.picks.clear();
        self.angle = "90 degrees".to_owned();
        self.angle_focused = false;
        self.last_pick_seq = state
            .work_plane_pick
            .as_ref()
            .map_or(0, |pick| pick.sequence);
        self.last_preview = None;
        self.error = None;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.active() {
            actions.push(GuiShellAction::WorkPlane(GuiWorkPlaneAction::Cancel));
        }
        self.reset();
    }

    fn reset(&mut self) {
        self.document = None;
        self.mode = None;
        self.picks.clear();
        self.angle_focused = false;
        self.last_preview = None;
        self.error = None;
    }

    pub(crate) fn status_prompt(&self) -> Option<&'static str> {
        let mode = self.mode?;
        Some(match mode {
            GuiWorkPlaneMode::General => "Define work plane by selecting geometry",
            GuiWorkPlaneMode::MidplaneTwoPlanes => match self.picks.len() {
                0 => "Select first plane or planar face",
                _ => "Select second plane or planar face",
            },
            GuiWorkPlaneMode::ThreePoints => match self.picks.len() {
                0 => "Select first vertex or point",
                1 => "Select second vertex or point",
                _ => "Select third vertex or point",
            },
            GuiWorkPlaneMode::ParallelThroughPoint => {
                if !self.has_plane() && !self.has_point() {
                    "Select reference plane or point"
                } else if !self.has_plane() {
                    "Select reference plane"
                } else {
                    "Select point or vertex"
                }
            }
            GuiWorkPlaneMode::NormalToAxisThroughPoint | GuiWorkPlaneMode::LineThroughPoint => {
                if !self.has_axis() && !self.has_point() {
                    "Select axis or point"
                } else if !self.has_axis() {
                    "Select axis or edge"
                } else {
                    "Select point or vertex"
                }
            }
            GuiWorkPlaneMode::AngleToPlaneAroundEdge => {
                if self.has_plane() && self.has_axis() {
                    "Enter angle and press Enter"
                } else if !self.has_plane() && !self.has_axis() {
                    "Select plane and straight edge or axis"
                } else if !self.has_plane() {
                    "Select reference plane"
                } else {
                    "Select straight edge or axis"
                }
            }
            GuiWorkPlaneMode::TwoCoplanarEdges => match self.picks.len() {
                0 => "Select first coplanar edge",
                _ => "Select second coplanar edge",
            },
            GuiWorkPlaneMode::MidplaneOfTorus => "Select a torus",
            GuiWorkPlaneMode::TangentThroughEdge => "Select a surface or edge",
            GuiWorkPlaneMode::TangentThroughPoint => "Select a surface or point",
            GuiWorkPlaneMode::TangentParallelToPlane => "Select a surface or plane",
        })
    }

    pub(crate) fn sync_and_draw(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: WorkspaceArea,
        browser_picks: (
            Option<crate::GuiOriginPlaneKind>,
            Option<oblikovati_model::feature::work_axis::WorkAxisReference>,
            Option<oblikovati_model::feature::FeatureId>,
        ),
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_some()
            && (self.document != state.document
                || state.authoring != crate::GuiAuthoringContext::Part)
        {
            self.close(actions);
            return;
        }
        let (Some(document), Some(mode)) = (self.document, self.mode) else {
            return;
        };
        if ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.close(actions);
            return;
        }
        self.ingest_browser_picks(mode, browser_picks);
        self.ingest_viewport_pick(state, document, mode);
        if self.should_auto_commit(mode) {
            let request = GuiWorkPlaneRequest {
                document,
                mode,
                picks: self.ordered_picks(mode),
                angle: self.angle.clone(),
            };
            actions.push(GuiShellAction::WorkPlane(GuiWorkPlaneAction::Commit(
                request,
            )));
            self.reset();
            return;
        }
        if mode == GuiWorkPlaneMode::AngleToPlaneAroundEdge && self.has_plane() && self.has_axis() {
            self.draw_angle_input(ui, area, document, actions);
        }
    }

    fn ingest_browser_picks(
        &mut self,
        mode: GuiWorkPlaneMode,
        (origin_plane, origin_axis, work_plane): (
            Option<crate::GuiOriginPlaneKind>,
            Option<oblikovati_model::feature::work_axis::WorkAxisReference>,
            Option<oblikovati_model::feature::FeatureId>,
        ),
    ) {
        if let Some(plane) = origin_plane {
            self.try_accept_pick(mode, GuiWorkPlanePick::OriginPlane(plane));
        }
        if let Some(axis) = origin_axis {
            self.try_accept_pick(mode, GuiWorkPlanePick::OriginAxis(axis));
        }
        if let Some(plane) = work_plane {
            self.try_accept_pick(mode, GuiWorkPlanePick::WorkPlane(plane));
        }
    }

    fn ingest_viewport_pick(
        &mut self,
        state: &GuiCommandState,
        document: u64,
        mode: GuiWorkPlaneMode,
    ) {
        let Some(event) = &state.work_plane_pick else {
            return;
        };
        if event.document != document || event.sequence <= self.last_pick_seq {
            return;
        }
        self.last_pick_seq = event.sequence;
        for pick in &event.candidates {
            if self.try_accept_pick(mode, pick.clone()) {
                break;
            }
        }
    }

    fn try_accept_pick(&mut self, mode: GuiWorkPlaneMode, pick: GuiWorkPlanePick) -> bool {
        if self.picks.contains(&pick) {
            return false;
        }
        let accepted = match mode {
            GuiWorkPlaneMode::General => true,
            GuiWorkPlaneMode::MidplaneTwoPlanes => pick.is_plane() && self.picks.len() < 2,
            GuiWorkPlaneMode::ThreePoints => pick.is_point() && self.picks.len() < 3,
            GuiWorkPlaneMode::ParallelThroughPoint => {
                (pick.is_plane() && !self.has_plane()) || (pick.is_point() && !self.has_point())
            }
            GuiWorkPlaneMode::NormalToAxisThroughPoint | GuiWorkPlaneMode::LineThroughPoint => {
                (pick.is_axis() && !self.has_axis()) || (pick.is_point() && !self.has_point())
            }
            GuiWorkPlaneMode::AngleToPlaneAroundEdge => {
                (pick.is_plane() && !self.has_plane()) || (pick.is_axis() && !self.has_axis())
            }
            GuiWorkPlaneMode::TwoCoplanarEdges => pick.is_axis() && self.picks.len() < 2,
            GuiWorkPlaneMode::MidplaneOfTorus => pick.is_surface() && self.picks.is_empty(),
            GuiWorkPlaneMode::TangentThroughEdge => {
                (pick.is_surface() && !self.has_surface()) || (pick.is_axis() && !self.has_axis())
            }
            GuiWorkPlaneMode::TangentThroughPoint => {
                (pick.is_surface() && !self.has_surface()) || (pick.is_point() && !self.has_point())
            }
            GuiWorkPlaneMode::TangentParallelToPlane => {
                (pick.is_surface() && !self.has_surface())
                    || (pick.is_plane() && !pick.is_surface() && self.picks.len() < 2)
            }
        };
        if accepted {
            self.picks.push(pick);
            self.error = None;
        }
        accepted
    }

    fn should_auto_commit(&self, mode: GuiWorkPlaneMode) -> bool {
        match mode {
            GuiWorkPlaneMode::General => false,
            GuiWorkPlaneMode::MidplaneTwoPlanes => self.picks.len() == 2,
            GuiWorkPlaneMode::ThreePoints => self.picks.len() == 3,
            GuiWorkPlaneMode::ParallelThroughPoint => self.has_plane() && self.has_point(),
            GuiWorkPlaneMode::NormalToAxisThroughPoint | GuiWorkPlaneMode::LineThroughPoint => {
                self.has_axis() && self.has_point()
            }
            GuiWorkPlaneMode::AngleToPlaneAroundEdge => false,
            GuiWorkPlaneMode::TwoCoplanarEdges => self.picks.len() == 2,
            GuiWorkPlaneMode::MidplaneOfTorus => self.picks.len() == 1,
            GuiWorkPlaneMode::TangentThroughEdge => self.has_surface() && self.has_axis(),
            GuiWorkPlaneMode::TangentThroughPoint => self.has_surface() && self.has_point(),
            GuiWorkPlaneMode::TangentParallelToPlane => self.picks.len() == 2,
        }
    }

    fn ordered_picks(&self, mode: GuiWorkPlaneMode) -> Vec<GuiWorkPlanePick> {
        match mode {
            GuiWorkPlaneMode::ParallelThroughPoint => ordered_pair(
                &self.picks,
                GuiWorkPlanePick::is_plane,
                GuiWorkPlanePick::is_point,
            ),
            GuiWorkPlaneMode::NormalToAxisThroughPoint | GuiWorkPlaneMode::LineThroughPoint => {
                ordered_pair(
                    &self.picks,
                    GuiWorkPlanePick::is_axis,
                    GuiWorkPlanePick::is_point,
                )
            }
            GuiWorkPlaneMode::AngleToPlaneAroundEdge => ordered_pair(
                &self.picks,
                GuiWorkPlanePick::is_axis,
                GuiWorkPlanePick::is_plane,
            ),
            GuiWorkPlaneMode::TangentThroughEdge => ordered_pair(
                &self.picks,
                GuiWorkPlanePick::is_surface,
                GuiWorkPlanePick::is_axis,
            ),
            GuiWorkPlaneMode::TangentThroughPoint => ordered_pair(
                &self.picks,
                GuiWorkPlanePick::is_surface,
                GuiWorkPlanePick::is_point,
            ),
            _ => self.picks.clone(),
        }
    }

    fn draw_angle_input(
        &mut self,
        ui: &Ui,
        area: WorkspaceArea,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let request = GuiWorkPlaneRequest {
            document,
            mode: GuiWorkPlaneMode::AngleToPlaneAroundEdge,
            picks: self.ordered_picks(GuiWorkPlaneMode::AngleToPlaneAroundEdge),
            angle: self.angle.clone(),
        };
        if self.last_preview.as_ref() != Some(&request) {
            self.last_preview = Some(request.clone());
            actions.push(GuiShellAction::WorkPlane(GuiWorkPlaneAction::Preview(
                request.clone(),
            )));
        }
        let height = 3. * ui.frame_height_with_spacing() + 34.;
        crate::widget_abi::fit_viewport_properties(ui, ANGLE_WINDOW, area, [ANGLE_WIDTH, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        let mut submitted = false;
        let mut cancelled = false;
        ui.window(ANGLE_WINDOW)
            .opened(&mut open)
            .collapsible(false)
            .resizable(false)
            .build(|| {
                crate::theme::floating_property_shadow(ui);
                crate::theme::property_label(ui, "Angle");
                if !self.angle_focused {
                    ui.set_keyboard_focus_here();
                    self.angle_focused = true;
                }
                let enter_pressed = {
                    let _numeric = crate::ui_scale::numeric_font(ui);
                    let _outline = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
                    ui.input_text("##Angle", &mut self.angle)
                        .auto_select_all(true)
                        .enter_returns_true(true)
                        .build()
                };
                if enter_pressed {
                    submitted = true;
                }
                ui.separator();
                if crate::theme::property_accept(ui, !self.angle.trim().is_empty()) {
                    submitted = true;
                }
                ui.same_line();
                if crate::widget_abi::button(ui, "Cancel", [72., 28.]) {
                    cancelled = true;
                }
            });
        if submitted && !self.angle.trim().is_empty() {
            let commit_req = GuiWorkPlaneRequest {
                document,
                mode: GuiWorkPlaneMode::AngleToPlaneAroundEdge,
                picks: self.ordered_picks(GuiWorkPlaneMode::AngleToPlaneAroundEdge),
                angle: self.angle.clone(),
            };
            actions.push(GuiShellAction::WorkPlane(GuiWorkPlaneAction::Commit(
                commit_req,
            )));
            self.reset();
        } else if !open || cancelled {
            self.close(actions);
        }
    }

    fn has_plane(&self) -> bool {
        self.picks.iter().any(GuiWorkPlanePick::is_plane)
    }
    fn has_point(&self) -> bool {
        self.picks.iter().any(GuiWorkPlanePick::is_point)
    }
    fn has_axis(&self) -> bool {
        self.picks.iter().any(GuiWorkPlanePick::is_axis)
    }
    fn has_surface(&self) -> bool {
        self.picks.iter().any(GuiWorkPlanePick::is_surface)
    }
}

fn ordered_pair(
    picks: &[GuiWorkPlanePick],
    first_pred: fn(&GuiWorkPlanePick) -> bool,
    second_pred: fn(&GuiWorkPlanePick) -> bool,
) -> Vec<GuiWorkPlanePick> {
    let mut result = Vec::with_capacity(2);
    if let Some(first) = picks.iter().find(|pick| first_pred(pick)) {
        result.push(first.clone());
    }
    if let Some(second) = picks
        .iter()
        .find(|pick| second_pred(pick) && result.first() != Some(*pick))
    {
        result.push(second.clone());
    }
    result
}
