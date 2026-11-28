// SPDX-License-Identifier: GPL-2.0-only
//! Selector pills, Boolean operation controls, footer, and Edit Combine Feature error modal (#3918).
use super::types::{GuiCombineAction, GuiCombinePickMode, GuiCombinePreview};
use super::{CombineDialog, EDIT_ERROR_TITLE};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::combine::CombineOperation;

impl CombineDialog {
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
        self.draw_base_row(ui, actions);
        self.draw_tool_row(ui, actions);
    }

    fn draw_base_row(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Basebody");
        let has_base = self.base_body.is_some();
        let active = self.pick_mode == GuiCombinePickMode::Base;
        let label = if has_base {
            "1 Solid##combine-base-pill"
        } else {
            "Select Solid##combine-base-pill"
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_base { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_base { 4.0 } else { 0.0 }).max(80.0);
        if selector_pill(ui, label, has_base, active, [pill_w, 22.0]) {
            self.pick_mode = GuiCombinePickMode::Base;
            actions.push(GuiShellAction::Combine(GuiCombineAction::SetPickMode(
                self.pick_mode,
            )));
        }
        if has_base {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##combine-base-clear", [clear_w, 22.0]) {
                self.base_body = None;
                self.pick_mode = GuiCombinePickMode::Base;
                actions.push(GuiShellAction::Combine(GuiCombineAction::SetPickMode(
                    self.pick_mode,
                )));
            }
        }
    }

    fn draw_tool_row(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Toolbodies");
        let count = self.tool_bodies.len();
        let has_tools = count > 0;
        let active = self.pick_mode == GuiCombinePickMode::Toolbodies;
        let label = match count {
            0 => "Select Solids##combine-tool-pill".to_string(),
            1 => "1 Solid##combine-tool-pill".to_string(),
            n => format!("{n} Solids##combine-tool-pill"),
        };
        let avail = ui.content_region_avail()[0];
        let clear_w = if has_tools { 46.0 } else { 0.0 };
        let pill_w = (avail - clear_w - if has_tools { 4.0 } else { 0.0 }).max(80.0);
        if selector_pill(ui, &label, has_tools, active, [pill_w, 22.0]) {
            self.pick_mode = GuiCombinePickMode::Toolbodies;
            actions.push(GuiShellAction::Combine(GuiCombineAction::SetPickMode(
                self.pick_mode,
            )));
        }
        if has_tools {
            ui.same_line_with_spacing(0.0, 4.0);
            if widget_abi::button(ui, "Clear##combine-tool-clear", [clear_w, 22.0]) {
                self.tool_bodies.clear();
                self.pick_mode = GuiCombinePickMode::Toolbodies;
                actions.push(GuiShellAction::Combine(GuiCombineAction::SetPickMode(
                    self.pick_mode,
                )));
            }
        }
    }

    pub(super) fn draw_output_controls(&mut self, ui: &Ui) {
        ui.text("Output");
        ui.text("Operation");
        let avail = ui.content_region_avail()[0].max(210.0);
        let btn_w = ((avail - 8.0) / 3.0).max(66.0);
        for (idx, (op, label)) in [
            (CombineOperation::Join, "Join"),
            (CombineOperation::Cut, "Cut"),
            (CombineOperation::Intersect, "Intersect"),
        ]
        .into_iter()
        .enumerate()
        {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 4.0);
            }
            if op_toggle_button(ui, label, self.operation == op, [btn_w, 22.0]) {
                self.operation = op;
            }
        }
        ui.spacing();
        ui.checkbox("Keep Toolbodies", &mut self.keep_toolbodies);
    }

    pub(super) fn draw_footer_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        popup_active: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let has_selections = self.base_body.is_some() && !self.tool_bodies.is_empty();
        let ready_request = has_selections
            .then(|| self.request(state))
            .flatten()
            .filter(|req| {
                GuiShellAction::Combine(GuiCombineAction::Commit(Box::new(req.clone())))
                    .dispatchable(state)
            });
        let failed_edit_cause = (self.editing.is_some() && has_selections)
            .then(|| {
                state.combine_preview.as_ref().and_then(|preview| {
                    let req = self.request(state)?;
                    if preview.request != req {
                        return None;
                    }
                    match &preview.result {
                        GuiCombinePreview::Failed(msg) => Some(msg.clone()),
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
                actions.push(GuiShellAction::Combine(GuiCombineAction::Commit(Box::new(
                    req,
                ))));
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
            actions.push(GuiShellAction::Combine(GuiCombineAction::Commit(Box::new(
                req,
            ))));
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
        ui.text("Problems occurred while executing the Edit Combine Feature command.");
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

fn op_toggle_button(ui: &Ui, label: &str, selected: bool, size: [f32; 2]) -> bool {
    let color = selected.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Button, [0.26, 0.36, 0.68, 1.0])
    });
    let clicked = widget_abi::button(ui, label, size);
    drop(color);
    clicked
}
