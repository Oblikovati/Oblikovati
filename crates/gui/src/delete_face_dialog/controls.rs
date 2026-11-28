// SPDX-License-Identifier: GPL-2.0-only
//! ImGui controls and failure modal for the Part Modify Delete Face Properties panel (#3918).
use super::{DeleteFaceDialog, GuiDeleteFaceAction, GuiDeleteFacePreview};
use crate::{GuiCommandState, GuiShellAction};
use imgui::{StyleColor, Ui};

const PICK_TEAL: [f32; 4] = [0.18, 0.56, 0.62, 1.0];
const ERROR_RED: [f32; 4] = [0.92, 0.24, 0.24, 1.0];
const FAILURE_WINDOW_TITLE: &str = "Create DelFace feature failed";

impl DeleteFaceDialog {
    pub(super) fn draw_input_geometry(&mut self, ui: &Ui) {
        let header = if self.input_geometry_open {
            "v Input Geometry##delete-face-input-hdr"
        } else {
            "> Input Geometry##delete-face-input-hdr"
        };
        if ui.button_with_size(header, [-1.0, 0.0]) {
            self.input_geometry_open = !self.input_geometry_open;
        }
        if !self.input_geometry_open {
            return;
        }
        crate::theme::property_label(ui, "Faces");
        let pill = match self.faces.len() {
            0 => "Select Faces".to_owned(),
            1 => "1 Face".to_owned(),
            n => format!("{n} Faces"),
        };
        let _bg = ui.push_style_color(StyleColor::Button, PICK_TEAL);
        let _ = ui.button(format!("{pill}##delete-face-faces-pick"));
        drop(_bg);
        if !self.faces.is_empty() {
            ui.same_line_with_spacing(0.0, 6.0);
            if ui.button("x##delete-face-faces-clear") {
                self.faces.clear();
                self.body = None;
            }
        }
    }

    pub(super) fn draw_behavior(&mut self, ui: &Ui) {
        let header = if self.behavior_open {
            "v Behavior##delete-face-behavior-hdr"
        } else {
            "> Behavior##delete-face-behavior-hdr"
        };
        if ui.button_with_size(header, [-1.0, 0.0]) {
            self.behavior_open = !self.behavior_open;
        }
        if !self.behavior_open {
            return;
        }
        ui.checkbox(
            "Heal Remaining Faces##delete-face-heal",
            &mut self.heal_remaining_faces,
        );
        ui.checkbox(
            "Select Lump or Void##delete-face-lump",
            &mut self.select_lump_or_void,
        );
    }

    pub(super) fn draw_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let req = self.request(state);
        let has_faces = !self.faces.is_empty();
        let preview_ready = req.as_ref().is_some_and(|r| {
            GuiShellAction::DeleteFace(GuiDeleteFaceAction::Commit(Box::new(r.clone())))
                .dispatchable(state)
        });
        let preview_failed = state
            .delete_face_preview
            .as_ref()
            .and_then(|p| match &p.result {
                GuiDeleteFacePreview::Failed(msg) => Some(msg.clone()),
                _ => None,
            });
        // Keep OK enabled whenever at least one face is selected so failed healed deletions
        // can report "Create DelFace feature failed" with Edit / Cancel recovery (delete-face-02).
        let ok_enabled = has_faces;

        let _disabled = ui.begin_disabled(!ok_enabled);
        let ok_clicked = ui.button_with_size("OK##delete-face-ok", [72.0, 28.0]);
        drop(_disabled);
        if ok_clicked {
            if preview_ready && let Some(r) = req {
                self.submitted = Some(r.clone());
                actions.push(GuiShellAction::DeleteFace(GuiDeleteFaceAction::Commit(
                    Box::new(r),
                )));
            } else {
                let detail =
                    preview_failed.unwrap_or_else(|| "Create DelFace feature failed".into());
                self.failure_modal = Some(detail);
            }
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if ui.button_with_size("Cancel##delete-face-cancel", [68.0, 28.0]) {
            self.close(actions);
        }
    }

    pub(super) fn draw_failure_modal(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let Some(detail) = self.failure_modal.clone() else {
            return;
        };
        let mut keep_open = true;
        let mut edit_clicked = false;
        let mut cancel_clicked = false;
        crate::widget_abi::fit_dialog(ui, FAILURE_WINDOW_TITLE, [360.0, 190.0], [380.0, 210.0]);
        let summary = detail
            .split(';')
            .next()
            .unwrap_or(&detail)
            .trim()
            .to_owned();
        ui.window(FAILURE_WINDOW_TITLE)
            .opened(&mut keep_open)
            .collapsible(false)
            .build(|| {
                ui.text_colored(ERROR_RED, "Create DelFace feature failed");
                ui.separator();
                if ui.button_with_size("Edit##delete-face-fail-edit", [72.0, 26.0]) {
                    edit_clicked = true;
                }
                ui.same_line_with_spacing(0.0, 6.0);
                if ui.button_with_size("Cancel##delete-face-fail-cancel", [72.0, 26.0]) {
                    cancel_clicked = true;
                }
                if !summary.is_empty() && summary != "Create DelFace feature failed" {
                    ui.separator();
                    ui.text_wrapped(&summary);
                }
            });
        if edit_clicked {
            self.failure_modal = None;
        } else if cancel_clicked || !keep_open {
            self.failure_modal = None;
            self.close(actions);
        }
    }
}
