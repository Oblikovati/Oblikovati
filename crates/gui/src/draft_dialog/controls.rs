// SPDX-License-Identifier: GPL-2.0-only
//! ImGui controls and failure modal for the Part Modify Draft Properties panel (#3918).
use super::{
    DraftDialog, GuiDraftAction, GuiDraftPickTarget, GuiDraftPreview, parse_angle_deg,
    parting_tool_label,
};
use crate::{GuiCommandState, GuiShellAction};
use imgui::{StyleColor, StyleVar, Ui};
use oblikovati_model::feature::draft::{DraftAngleMode, DraftMode};

const ACTIVE_BLUE: [f32; 4] = [0.24, 0.45, 0.78, 1.0];
const PICK_TEAL: [f32; 4] = [0.18, 0.56, 0.62, 1.0];
const ERROR_RED: [f32; 4] = [0.92, 0.24, 0.24, 1.0];

impl DraftDialog {
    pub(super) fn draw_mode_buttons(&mut self, ui: &Ui) {
        ui.text("Draft Type");
        let modes = [
            (DraftMode::FixedEdge, "Fixed Edge"),
            (DraftMode::FixedPlane, "Fixed Plane"),
            (DraftMode::PartingTool, "Parting Tool"),
        ];
        for (idx, (mode, label)) in modes.into_iter().enumerate() {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 4.0);
            }
            let active = self.mode == mode;
            let _bg = active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
            if ui.button_with_size(format!("{label}##draft-mode-{idx}"), [82.0, 0.0]) && !active {
                self.mode = mode;
                self.sync_pick_target_after_mode_change();
            }
        }
    }

    pub(super) fn draw_selectors(&mut self, ui: &Ui) {
        let pull_label = match self.mode {
            DraftMode::FixedPlane => "Fixed Plane",
            DraftMode::FixedEdge | DraftMode::PartingTool => "Pull Direction",
        };
        crate::theme::property_label(ui, pull_label);
        let pull_text = if self.pull_direction.is_some() {
            "1 Face"
        } else if self.pick_target == GuiDraftPickTarget::PullDirection {
            "Pick Face..."
        } else {
            "Select Face"
        };
        let pull_active = self.pick_target == GuiDraftPickTarget::PullDirection;
        let _pull_bg = pull_active.then(|| ui.push_style_color(StyleColor::Button, PICK_TEAL));
        if ui.button(format!("{pull_text}##draft-pull-pick")) {
            self.pick_target = GuiDraftPickTarget::PullDirection;
        }
        drop(_pull_bg);
        ui.same_line_with_spacing(0.0, 6.0);
        if ui.button("Flip##draft-flip") {
            self.direction_flipped = !self.direction_flipped;
        }

        if self.mode == DraftMode::PartingTool {
            crate::theme::property_label(ui, "Parting Tool");
            let parting_text = match &self.parting_tool {
                Some(tool) => parting_tool_label(tool),
                None if self.pick_target == GuiDraftPickTarget::PartingTool => "Pick Plane...",
                None => "Select Plane",
            };
            let active = self.pick_target == GuiDraftPickTarget::PartingTool;
            let _bg = active.then(|| ui.push_style_color(StyleColor::Button, PICK_TEAL));
            if ui.button(format!("{parting_text}##draft-parting-pick")) {
                self.pick_target = GuiDraftPickTarget::PartingTool;
            }
        }

        let faces_ready = self.faces_selector_enabled();
        crate::theme::property_label(ui, "Faces");
        let faces_text = match self.faces.len() {
            0 if faces_ready && self.pick_target == GuiDraftPickTarget::Faces => {
                "Pick Faces...".to_owned()
            }
            0 => "Select Faces".to_owned(),
            1 => "1 Face".to_owned(),
            n => format!("{n} Faces"),
        };
        let _disabled = ui.begin_disabled(!faces_ready);
        let faces_active = faces_ready && self.pick_target == GuiDraftPickTarget::Faces;
        let _bg = faces_active.then(|| ui.push_style_color(StyleColor::Button, PICK_TEAL));
        if ui.button(format!("{faces_text}##draft-faces-pick")) {
            self.pick_target = GuiDraftPickTarget::Faces;
        }
        drop(_bg);
        if !self.faces.is_empty() {
            ui.same_line_with_spacing(0.0, 6.0);
            if ui.button("Clear##draft-faces-clear") {
                self.faces.clear();
                self.pick_target = GuiDraftPickTarget::Faces;
            }
        }
        drop(_disabled);
    }

    pub(super) fn draw_angle_and_options(&mut self, ui: &Ui) -> bool {
        let invalid = parse_angle_deg(&self.angle).is_none_or(|a| a <= 0.0 || a >= 90.0);
        crate::theme::property_label(ui, "Draft Angle");
        let _numeric = crate::ui_scale::numeric_font(ui);
        let _border_w = ui.push_style_var(StyleVar::FrameBorderSize(1.0));
        let _text_c = invalid.then(|| ui.push_style_color(StyleColor::Text, ERROR_RED));
        let _border_c = invalid.then(|| ui.push_style_color(StyleColor::Border, ERROR_RED));
        let submitted = ui
            .input_text("##draft-angle", &mut self.angle)
            .enter_returns_true(true)
            .build();
        drop(_border_c);
        drop(_text_c);
        drop(_border_w);
        drop(_numeric);

        ui.separator();
        let opt_label = if self.options_expanded {
            "Options <<##draft-options-toggle"
        } else {
            "Options >>##draft-options-toggle"
        };
        if ui.button(opt_label) {
            self.options_expanded = !self.options_expanded;
        }
        if self.options_expanded {
            crate::theme::property_label(ui, "Angle Mode");
            let abs_active = self.angle_mode == DraftAngleMode::Absolute;
            let _b1 = abs_active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
            if ui.button("Absolute##draft-angle-abs") {
                self.angle_mode = DraftAngleMode::Absolute;
            }
            drop(_b1);
            ui.same_line_with_spacing(0.0, 6.0);
            let rel_active = self.angle_mode == DraftAngleMode::Relative;
            let _b2 = rel_active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
            if ui.button("Relative##draft-angle-rel") {
                self.angle_mode = DraftAngleMode::Relative;
            }
            drop(_b2);
            ui.checkbox(
                "Automatic Blending##draft-auto-blend",
                &mut self.automatic_blending,
            );
        }
        submitted
    }

    pub(super) fn draw_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        enter_commit: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let req = self.request(state);
        let geometry_complete = self.geometry_complete();
        let preview_ready = req.as_ref().is_some_and(|r| {
            GuiShellAction::Draft(GuiDraftAction::Commit(Box::new(r.clone()))).dispatchable(state)
        });
        let preview_failed = state.draft_preview.as_ref().and_then(|p| match &p.result {
            GuiDraftPreview::Failed(msg) => Some(msg.clone()),
            _ => None,
        });
        let local_angle_error = parse_angle_deg(&self.angle)
            .filter(|a| *a > 0.0 && *a < 90.0)
            .is_none()
            .then(|| {
                format!(
                    "draft angle {:?}: expected finite angle in (0, 90) degrees; errors during the part update",
                    self.angle.trim()
                )
            });
        // Allow clicking OK whenever selections are complete so invalid edits/creations open the
        // Create/Edit Face Draft failure modal with Edit / Cancel recovery (draft-01, draft-06).
        let ok_enabled = geometry_complete
            && (preview_ready || preview_failed.is_some() || local_angle_error.is_some());

        let _disabled = ui.begin_disabled(!ok_enabled);
        let ok_clicked = ui.button_with_size("OK##draft-ok", [80.0, 0.0])
            || (enter_commit && ok_enabled && self.failure_modal.is_none());
        drop(_disabled);
        if ok_clicked {
            if preview_ready && let Some(r) = req {
                self.submitted = Some(r.clone());
                actions.push(GuiShellAction::Draft(GuiDraftAction::Commit(Box::new(r))));
            } else {
                let detail = preview_failed
                    .or(local_angle_error)
                    .unwrap_or_else(|| "errors during the part update".into());
                self.failure_modal = Some(detail);
                self.failure_detail_expanded = false;
            }
        }
        ui.same_line_with_spacing(0.0, 8.0);
        if ui.button_with_size("Cancel##draft-cancel", [80.0, 0.0]) {
            self.close(actions);
        }
    }

    pub(super) fn draw_failure_modal(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let Some(detail) = self.failure_modal.clone() else {
            return;
        };
        let is_edit = self.editing.is_some();
        let title = if is_edit {
            "Edit Face Draft"
        } else {
            "Create Face Draft"
        };
        let headline = if is_edit {
            "Editing the face draft failed."
        } else {
            "Face draft creation failed."
        };
        let mut keep_open = true;
        let mut edit_clicked = false;
        let mut cancel_clicked = false;
        crate::widget_abi::fit_dialog(ui, title, [360.0, 220.0], [380.0, 230.0]);
        ui.window(title)
            .opened(&mut keep_open)
            .collapsible(false)
            .build(|| {
                ui.text_colored(ERROR_RED, headline);
                ui.separator();
                if ui.button("Edit##draft-fail-edit") {
                    edit_clicked = true;
                }
                ui.same_line_with_spacing(0.0, 6.0);
                if ui.button("Cancel##draft-fail-cancel") {
                    cancel_clicked = true;
                }
                ui.same_line_with_spacing(0.0, 10.0);
                let toggle = if self.failure_detail_expanded {
                    "Hide Details <<##draft-fail-details"
                } else {
                    "Details >>##draft-fail-details"
                };
                if ui.button(toggle) {
                    self.failure_detail_expanded = !self.failure_detail_expanded;
                }
                if self.failure_detail_expanded {
                    ui.separator();
                    ui.text_wrapped("There were errors during the part update.");
                    ui.text_wrapped(&detail);
                }
            });
        if edit_clicked {
            self.failure_modal = None;
            self.failure_detail_expanded = false;
        } else if cancel_clicked || !keep_open {
            self.failure_modal = None;
            self.failure_detail_expanded = false;
            self.close(actions);
        }
    }
}
