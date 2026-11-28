// SPDX-License-Identifier: GPL-2.0-only
//! ImGui controls and failure recovery modal for the Part Surface Replace Face panel (#3918).
use super::{
    GuiReplaceFaceAction, GuiReplaceFacePickMode, GuiReplaceFacePreview, ReplaceFaceDialog,
};
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{StyleColor, Ui};

const PICK_ACTIVE: [f32; 4] = [0.18, 0.56, 0.62, 1.0];
const PICK_FILLED: [f32; 4] = [0.20, 0.34, 0.56, 1.0];
const PICK_IDLE: [f32; 4] = [0.28, 0.28, 0.31, 1.0];
const ERROR_RED: [f32; 4] = [0.92, 0.24, 0.24, 1.0];
const FAILURE_WINDOW_TITLE: &str = "Create Replace Face feature failed";

impl ReplaceFaceDialog {
    pub(super) fn draw_input_geometry(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let header = if self.input_geometry_open {
            "v Input Geometry##replace-face-input-hdr"
        } else {
            "> Input Geometry##replace-face-input-hdr"
        };
        if widget_abi::button(ui, header, [-1.0, 0.0]) {
            self.input_geometry_open = !self.input_geometry_open;
        }
        if !self.input_geometry_open {
            return;
        }
        self.draw_existing_faces_selector(ui, actions);
        self.draw_new_faces_selector(ui, actions);
        let prev_chain = self.automatic_face_chain;
        ui.checkbox(
            "Automatic Face Chain##replace-face-chain",
            &mut self.automatic_face_chain,
        );
        if self.automatic_face_chain != prev_chain {
            actions.push(GuiShellAction::ReplaceFace(
                GuiReplaceFaceAction::SetAutomaticFaceChain(self.automatic_face_chain),
            ));
        }
    }

    fn draw_existing_faces_selector(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "Existing Faces");
        let count = self.existing_faces.len();
        let pill = match count {
            0 => "Select Existing Faces##replace-face-existing-pick".to_owned(),
            1 => "1 Face##replace-face-existing-pick".to_owned(),
            n => format!("{n} Faces##replace-face-existing-pick"),
        };
        let active = self.pick_mode == GuiReplaceFacePickMode::ExistingFaces;
        let color = selector_color(active, count > 0);
        let _bg = widget_abi::push_style_color(ui, StyleColor::Button, color);
        let clicked = widget_abi::button(ui, &pill, [0.0, 22.0]);
        drop(_bg);
        if clicked && !active {
            self.pick_mode = GuiReplaceFacePickMode::ExistingFaces;
            actions.push(GuiShellAction::ReplaceFace(
                GuiReplaceFaceAction::SetPickMode(self.pick_mode),
            ));
        }
        if count > 0 {
            ui.same_line_with_spacing(0.0, 6.0);
            if widget_abi::button(ui, "x##replace-face-existing-clear", [22.0, 22.0]) {
                self.existing_faces.clear();
                self.body = None;
            }
        }
    }

    fn draw_new_faces_selector(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        crate::theme::property_label(ui, "New Faces");
        let count = self.new_faces.len();
        let pill = match count {
            0 => "Select New Faces##replace-face-new-pick".to_owned(),
            1 => "1 Face##replace-face-new-pick".to_owned(),
            n => format!("{n} Faces##replace-face-new-pick"),
        };
        let active = self.pick_mode == GuiReplaceFacePickMode::NewFaces;
        let color = selector_color(active, count > 0);
        let _bg = widget_abi::push_style_color(ui, StyleColor::Button, color);
        let clicked = widget_abi::button(ui, &pill, [0.0, 22.0]);
        drop(_bg);
        if clicked && !active {
            self.pick_mode = GuiReplaceFacePickMode::NewFaces;
            actions.push(GuiShellAction::ReplaceFace(
                GuiReplaceFaceAction::SetPickMode(self.pick_mode),
            ));
        }
        if count > 0 {
            ui.same_line_with_spacing(0.0, 6.0);
            if widget_abi::button(ui, "x##replace-face-new-clear", [22.0, 22.0]) {
                self.new_faces.clear();
            }
        }
    }

    pub(super) fn draw_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let req = self.request(state);
        let has_both = !self.existing_faces.is_empty() && !self.new_faces.is_empty();
        let preview_ready = req.as_ref().is_some_and(|r| {
            GuiShellAction::ReplaceFace(GuiReplaceFaceAction::Commit(Box::new(r.clone())))
                .dispatchable(state)
        });
        let preview_failed = state
            .replace_face_preview
            .as_ref()
            .and_then(|p| match &p.result {
                GuiReplaceFacePreview::Failed(msg) => Some(msg.clone()),
                _ => None,
            });
        // Enable OK once both Existing Faces and New Faces are populated so failed replacements
        // display the recovery modal ("Create Replace Face feature failed") with Edit / Cancel.
        let ok_enabled = has_both;
        let _disabled = ui.begin_disabled(!ok_enabled);
        let ok_clicked = widget_abi::button(ui, "OK##replace-face-ok", [72.0, 28.0]);
        drop(_disabled);
        if ok_clicked {
            if preview_ready && let Some(r) = req {
                self.submitted = Some(r.clone());
                actions.push(GuiShellAction::ReplaceFace(GuiReplaceFaceAction::Commit(
                    Box::new(r),
                )));
            } else {
                let detail =
                    preview_failed.unwrap_or_else(|| "Create Replace Face feature failed".into());
                self.failure_modal = Some(detail);
            }
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if widget_abi::button(ui, "Cancel##replace-face-cancel", [68.0, 28.0]) {
            self.close(actions);
        }
    }

    pub(super) fn draw_failure_modal(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        if self.failure_modal.is_some() && !self.failure_modal_open {
            ui.open_popup(FAILURE_WINDOW_TITLE);
        }
        self.failure_modal_open = self.failure_modal.is_some();
        let Some(_popup) = ui
            .modal_popup_config(FAILURE_WINDOW_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        let Some(detail) = self.failure_modal.clone() else {
            ui.close_current_popup();
            return;
        };
        let summary = detail
            .split(';')
            .next()
            .unwrap_or(&detail)
            .trim()
            .to_owned();
        ui.text_colored(ERROR_RED, FAILURE_WINDOW_TITLE);
        if !summary.is_empty() && summary != FAILURE_WINDOW_TITLE {
            ui.separator();
            ui.text_wrapped(&summary);
        }
        ui.spacing();
        let edit_clicked = widget_abi::button(ui, "Edit", [64.0, 28.0]);
        ui.same_line_with_spacing(0.0, 8.0);
        let cancel_clicked = widget_abi::button(ui, "Cancel", [72.0, 28.0])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if edit_clicked || cancel_clicked {
            self.failure_modal = None;
            self.failure_modal_open = false;
            ui.close_current_popup();
            if cancel_clicked {
                self.close(actions);
            }
        }
    }
}

fn selector_color(active: bool, filled: bool) -> [f32; 4] {
    if active {
        PICK_ACTIVE
    } else if filled {
        PICK_FILLED
    } else {
        PICK_IDLE
    }
}
