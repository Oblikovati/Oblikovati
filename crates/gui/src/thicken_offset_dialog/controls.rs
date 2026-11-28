// SPDX-License-Identifier: GPL-2.0-only
//! Controls for Part Modify Thicken/Offset: Output, Faces, Distance, Direction, Boolean, Blending, and Approximation (#3918).
use super::types::{GuiThickenOffsetAction, GuiThickenOffsetPreview};
use super::{EDIT_ERROR_TITLE, ThickenOffsetDialog};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::thicken_offset::{
    ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean, ThickenDirection,
    ThickenOutputMode,
};

impl ThickenOffsetDialog {
    pub(super) fn draw_input_geometry(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
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
        crate::theme::property_label(ui, "Output");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(54.0);
        if toggle_button(
            ui,
            "Solid##thicken-output-solid",
            self.output == ThickenOutputMode::Solid,
            [half, 22.0],
        ) {
            self.output = ThickenOutputMode::Solid;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Surface##thicken-output-surface",
            self.output == ThickenOutputMode::Surface,
            [half, 22.0],
        ) {
            self.output = ThickenOutputMode::Surface;
        }

        crate::theme::property_label(ui, "Faces");
        let count = self.faces.len();
        let has_faces = count > 0;
        let face_label = match count {
            0 => "Select Faces##thicken-faces-pill".to_string(),
            1 => "1 Face##thicken-faces-pill".to_string(),
            n => format!("{n} Faces##thicken-faces-pill"),
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_faces { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_faces { 4.0 } else { 0.0 }).max(80.0);
        let _ = selector_pill(ui, &face_label, has_faces, true, [pill_w, 22.0]);
        if has_faces {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##thicken-faces-clear", [clear_w, 22.0]) {
                self.faces.clear();
                self.target_body = None;
            }
        }
        let prev_chain = self.automatic_face_chain;
        ui.checkbox("Automatic Face Chain", &mut self.automatic_face_chain);
        if self.automatic_face_chain != prev_chain {
            actions.push(GuiShellAction::ThickenOffset(
                GuiThickenOffsetAction::SetAutomaticFaceChain(self.automatic_face_chain),
            ));
        }
    }

    pub(super) fn draw_behavior_controls(&mut self, ui: &Ui, _actions: &mut Vec<GuiShellAction>) {
        ui.text("Behavior");
        crate::theme::property_label(ui, "Distance");
        let dist_valid = self.parsed_distance_cm().is_some();
        let field_w =
            (ui.content_region_avail()[0] - if dist_valid { 0.0 } else { 20.0 }).max(80.0);
        let border_color = (!dist_valid).then(|| {
            widget_abi::push_style_color(
                ui,
                imgui::StyleColor::Border,
                crate::theme::error_accent(ui),
            )
        });
        let border_w =
            (!dist_valid).then(|| ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.5)));
        ui.set_next_item_width(field_w);
        ui.input_text("##thicken-distance", &mut self.distance_text)
            .build();
        drop(border_w);
        drop(border_color);
        if !dist_valid {
            crate::feature_properties::draw_error_badge(ui, "Invalid distance");
        }

        ui.text("Direction");
        let show_third = !(self.output == ThickenOutputMode::Solid && self.automatic_blending);
        let choices: &[(ThickenDirection, &str)] = if show_third {
            &[
                (ThickenDirection::Negative, "Dir 1"),
                (ThickenDirection::Positive, "Dir 2"),
                (ThickenDirection::Symmetric, "Symmetric"),
            ]
        } else {
            &[
                (ThickenDirection::Negative, "Dir 1"),
                (ThickenDirection::Positive, "Dir 2"),
            ]
        };
        let avail = ui.content_region_avail()[0].max(160.0);
        let btn_w = ((avail - 4.0 * (choices.len() - 1) as f32) / choices.len() as f32).max(56.0);
        for (idx, &(dir, label)) in choices.iter().enumerate() {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 4.0);
            }
            if toggle_button(
                ui,
                &format!("{label}##thicken-dir-{idx}"),
                self.direction == dir,
                [btn_w, 22.0],
            ) {
                self.direction = dir;
            }
        }
        if self.output == ThickenOutputMode::Solid
            && ui.checkbox("Automatic Blending", &mut self.automatic_blending)
            && self.automatic_blending
            && self.direction == ThickenDirection::Symmetric
        {
            self.direction = ThickenDirection::Positive;
        }
    }

    pub(super) fn draw_output_controls(&mut self, ui: &Ui, state: &GuiCommandState) {
        if self.output == ThickenOutputMode::Surface {
            crate::theme::property_label(ui, "Body Name");
            let next_idx = state
                .browser
                .as_ref()
                .map_or(1, |b| b.surface_bodies.len() + 1);
            ui.align_text_to_frame_padding();
            ui.text(format!("Srf{next_idx}"));
            return;
        }
        if self.automatic_blending {
            return;
        }
        ui.text("Boolean");
        let avail = ui.content_region_avail()[0].max(210.0);
        let btn_w = ((avail - 4.0) * 0.5).max(80.0);
        let options = [
            (ThickenBoolean::Join, "Join"),
            (ThickenBoolean::Cut, "Cut"),
            (ThickenBoolean::Intersect, "Intersect"),
            (ThickenBoolean::NewSolid, "New Solid"),
        ];
        for (idx, &(op, label)) in options.iter().enumerate() {
            if idx % 2 == 1 {
                ui.same_line_with_spacing(0.0, 4.0);
            }
            if toggle_button(
                ui,
                &format!("{label}##thicken-bool-{idx}"),
                self.boolean == op,
                [btn_w, 22.0],
            ) {
                self.boolean = op;
            }
        }
    }

    pub(super) fn draw_advanced_properties(&mut self, ui: &Ui) {
        let header_label = if self.advanced_open {
            "v Advanced Properties"
        } else {
            "> Advanced Properties"
        };
        if widget_abi::selectable(ui, header_label, false, false) {
            self.advanced_open = !self.advanced_open;
        }
        if !self.advanced_open {
            return;
        }
        ui.checkbox("Allow Approximation", &mut self.allow_approximation);
        if !self.allow_approximation {
            return;
        }
        crate::theme::property_label(ui, "Type");
        ui.set_next_item_width(ui.content_region_avail()[0].max(100.0));
        if let Some(_combo) =
            ui.begin_combo("##thicken-approx-type", self.approximation_type.label())
        {
            for option in [
                ThickenApproximationType::NeverTooThin,
                ThickenApproximationType::NeverTooThick,
                ThickenApproximationType::Mean,
            ] {
                if widget_abi::selectable(
                    ui,
                    option.label(),
                    self.approximation_type == option,
                    false,
                ) {
                    self.approximation_type = option;
                }
            }
        }
        crate::theme::property_label(ui, "Tolerance");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(60.0);
        if toggle_button(
            ui,
            "Optimized##thicken-tol-opt",
            self.approximation_tolerance == ThickenApproximationTolerance::Optimized,
            [half, 22.0],
        ) {
            self.approximation_tolerance = ThickenApproximationTolerance::Optimized;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Specified##thicken-tol-spec",
            self.approximation_tolerance == ThickenApproximationTolerance::Specified,
            [half, 22.0],
        ) && self.approximation_tolerance != ThickenApproximationTolerance::Specified
        {
            self.approximation_tolerance = ThickenApproximationTolerance::Specified;
            if self.approximation_percent_text.trim().is_empty() {
                self.approximation_percent_text = "0,0 %".to_string();
            }
        }
        if self.approximation_tolerance == ThickenApproximationTolerance::Specified {
            crate::theme::property_label(ui, "Percent");
            let pct_valid = self.approximation_valid();
            let field_w =
                (ui.content_region_avail()[0] - if pct_valid { 0.0 } else { 20.0 }).max(80.0);
            let border_color = (!pct_valid).then(|| {
                widget_abi::push_style_color(
                    ui,
                    imgui::StyleColor::Border,
                    crate::theme::error_accent(ui),
                )
            });
            let border_w =
                (!pct_valid).then(|| ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.5)));
            ui.set_next_item_width(field_w);
            ui.input_text("##thicken-approx-pct", &mut self.approximation_percent_text)
                .build();
            drop(border_w);
            drop(border_color);
            if !pct_valid {
                crate::feature_properties::draw_error_badge(ui, "Invalid tolerance percentage");
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
        let local_valid = !self.faces.is_empty()
            && self.parsed_distance_cm().is_some()
            && self.approximation_valid();
        let ready_request = local_valid
            .then(|| self.request(state))
            .flatten()
            .filter(|req| {
                GuiShellAction::ThickenOffset(GuiThickenOffsetAction::Commit(Box::new(req.clone())))
                    .dispatchable(state)
            });
        let failed_edit_cause = (self.editing.is_some() && local_valid)
            .then(|| {
                state.thicken_offset_preview.as_ref().and_then(|preview| {
                    let req = self.request(state)?;
                    if preview.request != req {
                        return None;
                    }
                    match &preview.result {
                        GuiThickenOffsetPreview::Failed(msg) => Some(msg.clone()),
                        _ => None,
                    }
                })
            })
            .flatten();
        let can_ok = ready_request.is_some() || failed_edit_cause.is_some();
        if crate::theme::property_accept(ui, can_ok) {
            if let Some(req) = ready_request.clone() {
                self.submitted = Some(req.clone());
                self.reopen_after_commit = false;
                actions.push(GuiShellAction::ThickenOffset(
                    GuiThickenOffsetAction::Commit(Box::new(req)),
                ));
            } else if let Some(cause) = failed_edit_cause {
                self.edit_error = Some(cause);
            }
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
            actions.push(GuiShellAction::ThickenOffset(
                GuiThickenOffsetAction::Commit(Box::new(req)),
            ));
        }
    }

    pub(super) fn draw_edit_error_modal(&mut self, ui: &Ui) -> bool {
        if self.edit_error.is_some() && !self.edit_error_open {
            ui.open_popup(EDIT_ERROR_TITLE);
        }
        self.edit_error_open = self.edit_error.is_some();
        let Some(_popup) = ui
            .modal_popup_config(EDIT_ERROR_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return false;
        };
        let Some(cause) = self.edit_error.clone() else {
            ui.close_current_popup();
            return false;
        };
        ui.text("Problems occurred while executing the Edit Thicken feature command.");
        if ui.collapsing_header("Details", imgui::TreeNodeFlags::empty()) {
            ui.text_wrapped(&cause);
        }
        ui.spacing();
        if widget_abi::button(ui, "Edit", [64.0, 28.0]) {
            self.edit_error = None;
            self.edit_error_open = false;
            ui.close_current_popup();
            return false;
        }
        ui.same_line_with_spacing(0.0, 6.0);
        let cancel = widget_abi::button(ui, "Cancel", [68.0, 28.0])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if cancel {
            self.edit_error = None;
            self.edit_error_open = false;
            ui.close_current_popup();
            return true;
        }
        false
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

fn toggle_button(ui: &Ui, label: &str, selected: bool, size: [f32; 2]) -> bool {
    let color = selected.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Button, [0.26, 0.36, 0.68, 1.0])
    });
    let clicked = widget_abi::button(ui, label, size);
    drop(color);
    clicked
}
