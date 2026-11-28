// SPDX-License-Identifier: GPL-2.0-only
//! Selector pills, mode switch, All Faces, Keep Side controls, and footer for Split (#3918).
use super::SplitDialog;
use super::types::{GuiSplitAction, GuiSplitPickMode};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::split::{SplitKeepSide, SplitMode, SplitTool};

impl SplitDialog {
    pub(super) fn draw_input_geometry(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let header_label = if self.input_geometry_open {
            "v Input Geometry"
        } else {
            "> Input Geometry"
        };
        if widget_abi::selectable(ui, header_label, false, false) {
            self.input_geometry_open = !self.input_geometry_open;
        }
        if !self.input_geometry_open {
            return;
        }
        self.draw_tool_row(ui, actions);
        match self.mode {
            SplitMode::Faces => self.draw_faces_rows(ui, state, actions),
            SplitMode::Solid => self.draw_solid_rows(ui, actions),
        }
    }

    fn draw_tool_row(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Tool");
        let has_tool = self.tool.is_some();
        let active = self.pick_mode == GuiSplitPickMode::Tool;
        let label = match &self.tool {
            None => "Select Tool##split-tool-pill",
            Some(SplitTool::SketchCurve { .. }) => "1 Curve##split-tool-pill",
            Some(SplitTool::OriginPlane(_) | SplitTool::WorkPlane(_)) => {
                "1 Work Plane##split-tool-pill"
            }
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_tool { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_tool { 4.0 } else { 0.0 }).max(80.0);
        if selector_pill(ui, label, has_tool, active, [pill_w, 22.0]) {
            self.pick_mode = GuiSplitPickMode::Tool;
            actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
                self.pick_mode,
                self.mode,
            )));
        }
        if has_tool {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##split-tool-clear", [clear_w, 22.0]) {
                self.tool = None;
                self.pick_mode = GuiSplitPickMode::Tool;
                actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
                    self.pick_mode,
                    self.mode,
                )));
            }
        }
    }

    fn draw_faces_rows(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let mut all_faces = self.all_faces;
        if ui.checkbox("All Faces##split-all-faces", &mut all_faces) {
            self.set_all_faces(all_faces, state, actions);
        }
        // Label + mode-switch icon button beside the Faces selector (split-04).
        let row_x = ui.cursor_pos()[0];
        ui.align_text_to_frame_padding();
        ui.text("Faces");
        ui.same_line_with_spacing(0.0, 6.0);
        if widget_abi::button(ui, "Solid##split-mode-switch", [42.0, 22.0]) {
            self.switch_mode(SplitMode::Solid, actions);
            return;
        }
        let control_x = row_x + (ui.window_content_region_max()[0] - row_x) * 0.42;
        if ui.cursor_pos()[0] < control_x {
            ui.same_line_with_spacing(0.0, 0.0);
            ui.set_cursor_pos([control_x, ui.cursor_pos()[1]]);
        } else {
            ui.same_line_with_spacing(0.0, 4.0);
        }
        let count = self.displayed_face_count();
        let has_faces = count > 0;
        let active = self.pick_mode == GuiSplitPickMode::Target;
        let label = match count {
            0 => "Select Faces##split-faces-pill".to_string(),
            1 => "1 Face##split-faces-pill".to_string(),
            n => format!("{n} Faces##split-faces-pill"),
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_faces { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_faces { 4.0 } else { 0.0 }).max(74.0);
        if selector_pill(ui, &label, has_faces, active, [pill_w, 22.0]) {
            self.pick_mode = GuiSplitPickMode::Target;
            actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
                self.pick_mode,
                self.mode,
            )));
        }
        if has_faces {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##split-faces-clear", [clear_w, 22.0]) {
                self.all_faces = false;
                self.all_faces_count = 0;
                self.faces.clear();
                self.pick_mode = GuiSplitPickMode::Target;
                actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
                    self.pick_mode,
                    self.mode,
                )));
            }
        }
    }

    fn draw_solid_rows(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let row_x = ui.cursor_pos()[0];
        ui.align_text_to_frame_padding();
        ui.text("Solid");
        ui.same_line_with_spacing(0.0, 6.0);
        if widget_abi::button(ui, "Faces##split-mode-switch", [42.0, 22.0]) {
            self.switch_mode(SplitMode::Faces, actions);
            return;
        }
        let control_x = row_x + (ui.window_content_region_max()[0] - row_x) * 0.42;
        if ui.cursor_pos()[0] < control_x {
            ui.same_line_with_spacing(0.0, 0.0);
            ui.set_cursor_pos([control_x, ui.cursor_pos()[1]]);
        } else {
            ui.same_line_with_spacing(0.0, 4.0);
        }
        let has_solid = self.body.is_some();
        let active = self.pick_mode == GuiSplitPickMode::Target;
        let label = if has_solid {
            "1 Solid##split-solid-pill"
        } else {
            "Select Solid##split-solid-pill"
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_solid { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_solid { 4.0 } else { 0.0 }).max(74.0);
        if selector_pill(ui, label, has_solid, active, [pill_w, 22.0]) {
            self.pick_mode = GuiSplitPickMode::Target;
            actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
                self.pick_mode,
                self.mode,
            )));
        }
        if has_solid {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##split-solid-clear", [clear_w, 22.0]) {
                self.body = None;
                self.pick_mode = GuiSplitPickMode::Target;
                actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
                    self.pick_mode,
                    self.mode,
                )));
            }
        }
        ui.spacing();
        ui.text("Keep Side");
        let avail = ui.content_region_avail()[0].max(210.0);
        let btn_w = ((avail - 8.0) / 3.0).max(66.0);
        for (idx, (side, title)) in [
            (SplitKeepSide::Both, "Keep Both"),
            (SplitKeepSide::SideA, "Keep Side A"),
            (SplitKeepSide::SideB, "Keep Side B"),
        ]
        .into_iter()
        .enumerate()
        {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 4.0);
            }
            if side_toggle_button(ui, title, self.keep_side == side, [btn_w, 22.0]) {
                self.keep_side = side;
            }
        }
    }

    pub(super) fn draw_footer_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        popup_active: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let ready_request = self
            .is_complete()
            .then(|| self.request(state))
            .flatten()
            .filter(|req| {
                GuiShellAction::Split(GuiSplitAction::Commit(Box::new(req.clone())))
                    .dispatchable(state)
            });
        let can_ok = ready_request.is_some();
        if crate::theme::property_accept(ui, can_ok)
            && let Some(req) = ready_request.clone()
        {
            self.submitted = Some(req.clone());
            self.reopen_after_commit = false;
            actions.push(GuiShellAction::Split(GuiSplitAction::Commit(Box::new(req))));
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if crate::theme::property_cancel(ui)
            || (!popup_active
                && ui.is_window_focused()
                && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
            return;
        }
        let plus_w = 28.0;
        let right = ui.window_content_region_max()[0] - plus_w;
        if right > ui.cursor_pos()[0] + 6.0 {
            ui.same_line_with_spacing(0.0, 0.0);
            ui.set_cursor_pos([right, ui.cursor_pos()[1]]);
        } else {
            ui.same_line_with_spacing(0.0, 6.0);
        }
        let _disabled = ui.begin_disabled(ready_request.is_none());
        if widget_abi::button(ui, "+", [plus_w, 0.0])
            && let Some(req) = ready_request
        {
            self.submitted = Some(req.clone());
            self.reopen_after_commit = true;
            actions.push(GuiShellAction::Split(GuiSplitAction::Commit(Box::new(req))));
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

fn side_toggle_button(ui: &Ui, label: &str, selected: bool, size: [f32; 2]) -> bool {
    let color = selected.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Button, [0.26, 0.36, 0.68, 1.0])
    });
    let clicked = widget_abi::button(ui, label, size);
    drop(color);
    clicked
}

pub(super) fn parse_browser_plane_tool(reference: &str) -> Option<SplitTool> {
    let trimmed = reference.trim();
    if let Some(raw_id) = trimmed.strip_prefix("datum/v1/")
        && let Ok(id) = raw_id.parse::<u64>()
    {
        return Some(SplitTool::WorkPlane(oblikovati_model::feature::FeatureId(
            id,
        )));
    }
    let lower = trimmed.to_ascii_lowercase();
    if lower.contains("xy") {
        Some(SplitTool::OriginPlane(
            oblikovati_model::feature::hole::HoleOriginPlane::Xy,
        ))
    } else if lower.contains("xz") {
        Some(SplitTool::OriginPlane(
            oblikovati_model::feature::hole::HoleOriginPlane::Xz,
        ))
    } else if lower.contains("yz") {
        Some(SplitTool::OriginPlane(
            oblikovati_model::feature::hole::HoleOriginPlane::Yz,
        ))
    } else {
        None
    }
}

pub(super) fn decode_hex_key(raw: &str) -> Result<Vec<u8>, String> {
    let trimmed = raw.trim();
    if !trimmed.len().is_multiple_of(2) {
        return Err("split body key: expected even-length hex string".into());
    }
    let mut bytes = Vec::with_capacity(trimmed.len() / 2);
    for pair in trimmed.as_bytes().as_chunks::<2>().0 {
        let text = std::str::from_utf8(pair).map_err(|_| "split body key: expected ASCII hex")?;
        let byte = u8::from_str_radix(text, 16).map_err(|_| "split body key: expected hex")?;
        bytes.push(byte);
    }
    Ok(bytes)
}
