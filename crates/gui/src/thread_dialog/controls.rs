// SPDX-License-Identifier: GPL-2.0-only
//! Location, Specification, Direction, and Footer controls for Part Modify Thread (#3918).
use super::ThreadDialog;
use super::catalog::{
    THREAD_TYPES, classes_for, designations_for, format_mm_3dp, on_size_changed,
    on_thread_type_changed, sizes_for_type, validate_fields,
};
use super::types::{GuiThreadAction, GuiThreadPreview};
use crate::feature_properties::draw_pick_pill;
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::thread::ThreadHandedness;

impl ThreadDialog {
    pub(super) fn draw_location_section(&mut self, ui: &Ui, local_error: Option<&str>) -> bool {
        let mut enter_commit = false;
        ui.text("Location");
        let has_face = self.face.is_some();
        crate::theme::property_label(ui, "Face");
        draw_pick_pill(
            ui,
            if has_face { "1 Surface" } else { "Select Face" },
            has_face,
        );

        let was_full = self.full_depth;
        if ui.checkbox("Full Depth", &mut self.full_depth)
            && self.full_depth
            && !was_full
            && self.measured_depth_mm > 0.0
        {
            self.depth = format_mm_3dp(self.measured_depth_mm);
        }

        let bad_depth = local_error.is_some_and(|e| e.contains("Depth"));
        if self.full_depth {
            let mut shown = if self.measured_depth_mm > 0.0 {
                format_mm_3dp(self.measured_depth_mm)
            } else {
                self.depth.clone()
            };
            let _disabled = ui.begin_disabled(true);
            numeric_row(ui, "Depth", &mut shown, false);
        } else {
            enter_commit |= numeric_row(ui, "Depth", &mut self.depth, bad_depth);
            let bad_offset = local_error.is_some_and(|e| e.contains("Offset"));
            enter_commit |= numeric_row(ui, "Offset", &mut self.offset, bad_offset);
        }

        ui.checkbox("Display thread in model", &mut self.display_in_model);
        enter_commit
    }

    pub(super) fn draw_specification_section(&mut self, ui: &Ui) {
        ui.separator();
        ui.text("Specification");
        let internal = self.face_internal;

        if draw_combo_row(
            ui,
            "Thread Type",
            "##thread-type",
            &mut self.spec.thread_type,
            THREAD_TYPES,
        ) {
            on_thread_type_changed(&mut self.spec, internal);
        }

        let sizes = sizes_for_type(&self.spec.thread_type);
        if draw_combo_row(ui, "Size", "##thread-size", &mut self.spec.size, sizes) {
            on_size_changed(&mut self.spec, internal);
        }

        let designations = designations_for(&self.spec.thread_type, &self.spec.size);
        draw_combo_row(
            ui,
            "Designation",
            "##thread-designation",
            &mut self.spec.designation,
            designations,
        );

        let classes = classes_for(&self.spec.thread_type, internal);
        draw_combo_row(ui, "Class", "##thread-class", &mut self.spec.class, classes);

        crate::theme::property_label(ui, "Direction");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(40.0);
        if toggle_button(
            ui,
            "Right Hand",
            self.spec.handedness == ThreadHandedness::RightHand,
            [half, 20.0],
        ) {
            self.spec.handedness = ThreadHandedness::RightHand;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Left Hand",
            self.spec.handedness == ThreadHandedness::LeftHand,
            [half, 20.0],
        ) {
            self.spec.handedness = ThreadHandedness::LeftHand;
        }
    }

    pub(super) fn draw_footer_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        enter_commit: bool,
        popup_active: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let local_ok = validate_fields(self.full_depth, &self.depth, &self.offset).is_none();
        let req = self.request(state);
        let preview_result = req.as_ref().and_then(|current| {
            state
                .thread_preview
                .as_ref()
                .filter(|preview| preview.request == *current)
                .map(|preview| &preview.result)
        });
        let can_commit = local_ok
            && req.as_ref().is_some_and(|current| {
                GuiShellAction::Thread(GuiThreadAction::Commit(Box::new(current.clone())))
                    .dispatchable(state)
            });
        let failed_message = match preview_result {
            Some(GuiThreadPreview::Failed(message)) if local_ok && self.face.is_some() => {
                Some(message.clone())
            }
            _ => None,
        };
        let can_ok = can_commit || failed_message.is_some();
        let ok_clicked = crate::theme::property_accept(ui, can_ok);
        if ok_clicked || (enter_commit && can_ok) {
            if let Some(message) = failed_message {
                self.modals.show_edit_failed(message);
                return;
            }
            if let Some(current) = req.clone() {
                self.submitted = Some(current.clone());
                self.reopen_after_commit = false;
                actions.push(GuiShellAction::Thread(GuiThreadAction::Commit(Box::new(
                    current,
                ))));
            }
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if crate::theme::property_cancel(ui)
            || (!popup_active
                && !self.modals.any_active()
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
        let _disabled = ui.begin_disabled(!can_commit);
        if widget_abi::button(ui, "+", [plus_w, 0.0])
            && let Some(current) = req
        {
            self.submitted = Some(current.clone());
            self.reopen_after_commit = true;
            actions.push(GuiShellAction::Thread(GuiThreadAction::Commit(Box::new(
                current,
            ))));
        }
    }
}

fn toggle_button(ui: &Ui, label: &str, selected: bool, size: [f32; 2]) -> bool {
    let color = selected.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Button, [0.26, 0.36, 0.68, 1.0])
    });
    let clicked = widget_abi::button(ui, label, size);
    drop(color);
    clicked
}

fn draw_combo_row(ui: &Ui, label: &str, id: &str, current: &mut String, items: &[&str]) -> bool {
    crate::theme::property_label(ui, label);
    let mut changed = false;
    if let Some(_combo) = ui.begin_combo(id, current.as_str()) {
        for &item in items {
            if widget_abi::selectable(ui, item, current == item, false) {
                *current = item.to_owned();
                changed = true;
            }
        }
    }
    changed
}

fn numeric_row(ui: &Ui, label: &str, value: &mut String, invalid: bool) -> bool {
    crate::theme::property_label(ui, label);
    let _id = ui.push_id(label);
    let _numeric = crate::ui_scale::numeric_font(ui);
    let _outline = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.0));
    let _border = invalid.then(|| {
        widget_abi::push_style_color(ui, imgui::StyleColor::Border, [0.88, 0.25, 0.25, 1.0])
    });
    ui.input_text("##value", value)
        .enter_returns_true(true)
        .build()
}
