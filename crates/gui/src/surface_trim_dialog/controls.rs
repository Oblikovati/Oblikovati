// SPDX-License-Identifier: GPL-2.0-only
//! Selector pills, removal-region flip control, and OK/Apply/Cancel footer for Trim Surface (#3918).
use super::SurfaceTrimDialog;
use super::types::{GuiSurfaceTrimAction, GuiSurfaceTrimPickMode};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{hole::HoleOriginPlane, surface_trim::SurfaceTrimTool};

impl SurfaceTrimDialog {
    pub(super) fn draw_cutting_tool_row(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Cutting Tool");
        let has_tool = self.tool.is_some();
        let active = self.pick_mode == GuiSurfaceTrimPickMode::CuttingTool;
        let label = match &self.tool {
            None => "Select Cutting Tool##trim-tool-pill",
            Some(SurfaceTrimTool::OriginPlane(HoleOriginPlane::Xy)) => "XY Plane##trim-tool-pill",
            Some(SurfaceTrimTool::OriginPlane(HoleOriginPlane::Xz)) => "XZ Plane##trim-tool-pill",
            Some(SurfaceTrimTool::OriginPlane(HoleOriginPlane::Yz)) => "YZ Plane##trim-tool-pill",
            Some(SurfaceTrimTool::WorkPlane(_)) => "1 Work Plane##trim-tool-pill",
            Some(SurfaceTrimTool::SurfaceFace { .. }) => "1 Surface##trim-tool-pill",
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_tool { 44.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_tool { 4.0 } else { 0.0 }).max(80.0);
        if selector_pill(ui, label, has_tool, active, [pill_w, 22.0]) {
            self.set_pick_mode(GuiSurfaceTrimPickMode::CuttingTool, actions);
        }
        if has_tool {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##trim-tool-clear", [clear_w, 22.0]) {
                self.tool = None;
                self.target_body = None;
                self.remove_side = None;
                self.set_pick_mode(GuiSurfaceTrimPickMode::CuttingTool, actions);
            }
        }
    }

    pub(super) fn draw_remove_row(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Remove");
        let has_region = self.target_body.is_some() && self.remove_side.is_some();
        let active = self.pick_mode == GuiSurfaceTrimPickMode::Remove;
        let label = if has_region {
            "1 Region##trim-remove-pill"
        } else {
            "Select Region##trim-remove-pill"
        };
        let avail = ui.content_region_avail()[0];
        let flip_w = 48.0;
        let pill_w = (avail - flip_w - 4.0).max(76.0);
        if selector_pill(ui, label, has_region, active, [pill_w, 22.0]) {
            self.set_pick_mode(GuiSurfaceTrimPickMode::Remove, actions);
        }
        ui.same_line_with_spacing(0.0, 4.0);
        let _disabled = ui.begin_disabled(!has_region);
        if widget_abi::button(ui, "Flip##trim-switch-region", [flip_w, 22.0])
            && let Some(side) = self.remove_side
        {
            self.remove_side = Some(side.opposite());
        }
        drop(_disabled);
        if ui.is_item_hovered() {
            ui.tooltip_text("Switch removal region");
        }
    }

    pub(super) fn draw_footer_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let ready_request = self
            .is_complete()
            .then(|| self.request(state))
            .flatten()
            .filter(|req| {
                GuiShellAction::SurfaceTrim(GuiSurfaceTrimAction::Commit(Box::new(req.clone())))
                    .dispatchable(state)
            });
        let can_commit = ready_request.is_some();
        if crate::theme::property_accept(ui, can_commit)
            && let Some(req) = ready_request.clone()
        {
            self.submitted = Some(req.clone());
            self.reopen_after_commit = false;
            actions.push(GuiShellAction::SurfaceTrim(GuiSurfaceTrimAction::Commit(
                Box::new(req),
            )));
        }
        ui.same_line_with_spacing(0.0, 6.0);
        {
            let _disabled = ui.begin_disabled(!can_commit);
            if widget_abi::button(ui, "Apply", [58.0, 0.0])
                && let Some(req) = ready_request
            {
                self.submitted = Some(req.clone());
                self.reopen_after_commit = true;
                actions.push(GuiShellAction::SurfaceTrim(GuiSurfaceTrimAction::Commit(
                    Box::new(req),
                )));
            }
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if crate::theme::property_cancel(ui)
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
    }
}

fn selector_pill(ui: &Ui, label: &str, selected: bool, active: bool, size: [f32; 2]) -> bool {
    let color = if active {
        [0.24, 0.44, 0.78, 1.0]
    } else if selected {
        [0.20, 0.34, 0.56, 1.0]
    } else {
        [0.28, 0.28, 0.31, 1.0]
    };
    let _bg = widget_abi::push_style_color(ui, imgui::StyleColor::Button, color);
    widget_abi::button(ui, label, size)
}

pub(super) fn parse_browser_plane_tool(reference: &str) -> Option<SurfaceTrimTool> {
    let trimmed = reference.trim();
    if let Some(raw_id) = trimmed.strip_prefix("datum/v1/")
        && let Ok(id) = raw_id.parse::<u64>()
    {
        return Some(SurfaceTrimTool::WorkPlane(
            oblikovati_model::feature::FeatureId(id),
        ));
    }
    let lower = trimmed.to_ascii_lowercase();
    if lower.contains("xy") {
        Some(SurfaceTrimTool::OriginPlane(HoleOriginPlane::Xy))
    } else if lower.contains("xz") {
        Some(SurfaceTrimTool::OriginPlane(HoleOriginPlane::Xz))
    } else if lower.contains("yz") {
        Some(SurfaceTrimTool::OriginPlane(HoleOriginPlane::Yz))
    } else {
        None
    }
}
