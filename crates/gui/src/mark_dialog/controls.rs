// SPDX-License-Identifier: GPL-2.0-only
//! Selection Sets, Behavior, and Footer controls for Part Modify Mark (#3918).
use super::MarkDialog;
use super::types::{GuiMarkAction, GuiMarkGeometrySet, GuiMarkPreview};
use crate::feature_properties::draw_pick_pill;
use crate::widget_abi;
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::mark::{MarkDirection, MarkMethod, MarkType};

const MARK_TYPES: [MarkType; 2] = [MarkType::MarkSurface, MarkType::MarkThrough];

impl MarkDialog {
    pub(super) fn draw_selection_sets_section(&mut self, ui: &Ui) {
        ui.text("Selection Sets");
        ui.text("Mark Geometry Sets");
        ui.same_line_with_spacing(0.0, 6.0);
        let add_color =
            widget_abi::push_style_color(ui, imgui::StyleColor::Button, [0.18, 0.58, 0.30, 1.0]);
        let add_hovered = widget_abi::push_style_color(
            ui,
            imgui::StyleColor::ButtonHovered,
            [0.22, 0.68, 0.36, 1.0],
        );
        let add_clicked = widget_abi::button(ui, "+##add-mark-set", [24.0, 20.0]);
        drop(add_hovered);
        drop(add_color);
        if add_clicked {
            self.geometry_sets.push(GuiMarkGeometrySet::default());
            self.active_set = self.geometry_sets.len().saturating_sub(1);
        }

        let mut remove_idx = None;
        let set_count = self.geometry_sets.len();
        for idx in 0..set_count {
            let _row_id = ui.push_id_usize(idx);
            let active = self.active_set == idx;
            let count = self.geometry_sets[idx].curves.len();
            let geom_label = match count {
                0 => "Select Geometry".to_owned(),
                1 => "1 Curve".to_owned(),
                n => format!("{n} Curves"),
            };
            crate::theme::property_label(ui, &format!("Set {}", idx + 1));
            draw_pick_pill(ui, &geom_label, count > 0 || active);
            if ui.is_item_clicked() {
                self.active_set = idx;
            }
            if set_count > 1 {
                ui.same_line_with_spacing(0.0, 4.0);
                if widget_abi::button(ui, "x##rm-set", [20.0, 20.0]) {
                    remove_idx = Some(idx);
                }
            }
            crate::theme::property_label(ui, "Mark Type");
            let current_label = self.geometry_sets[idx].mark_type.label();
            let combo_id = format!("##mark-type-{idx}");
            if let Some(_combo) = ui.begin_combo(&combo_id, current_label) {
                for kind in MARK_TYPES {
                    let selected = self.geometry_sets[idx].mark_type == kind;
                    if widget_abi::selectable(ui, kind.label(), selected, false) {
                        self.geometry_sets[idx].mark_type = kind;
                    }
                }
            }
        }
        if let Some(idx) = remove_idx
            && self.geometry_sets.len() > 1
        {
            self.geometry_sets.remove(idx);
            if self.active_set >= self.geometry_sets.len() {
                self.active_set = self.geometry_sets.len() - 1;
            }
        }
    }

    pub(super) fn draw_behavior_section(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        ui.separator();
        ui.text("Behavior");
        crate::theme::property_label(ui, "Method");
        let half = ((ui.content_region_avail()[0] - 4.0) * 0.5).max(40.0);
        if toggle_button(
            ui,
            "Project",
            self.method == MarkMethod::ProjectToClosestFace,
            [half, 22.0],
        ) {
            self.method = MarkMethod::ProjectToClosestFace;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Wrap to Face",
            self.method == MarkMethod::WrapToFace,
            [half, 22.0],
        ) {
            self.method = MarkMethod::WrapToFace;
        }

        if self.method == MarkMethod::WrapToFace {
            let face_count = self.faces.len();
            let face_label = match face_count {
                0 => "Select Face".to_owned(),
                1 => "1 Face".to_owned(),
                n => format!("{n} Faces"),
            };
            crate::theme::property_label(ui, "Faces");
            draw_pick_pill(ui, &face_label, face_count > 0);
            if face_count > 0 {
                ui.same_line_with_spacing(0.0, 4.0);
                if widget_abi::button(ui, "Clear##mark-faces", [44.0, 20.0]) {
                    self.faces.clear();
                    self.seed_face = None;
                }
            }
            if ui.checkbox("Automatic Face Chain", &mut self.automatic_face_chain)
                && let Some(seed) = self.seed_face.clone()
                && let (Some(document), Some(generation)) =
                    (self.document, state.document_generation)
            {
                actions.push(GuiShellAction::Mark(GuiMarkAction::ExpandFaceChain {
                    document,
                    generation,
                    seed_face: seed,
                    automatic_face_chain: self.automatic_face_chain,
                }));
            }
        }

        crate::theme::property_label(ui, "Direction");
        let third = ((ui.content_region_avail()[0] - 8.0) / 3.0).max(32.0);
        if toggle_button(
            ui,
            "Pos",
            self.direction == MarkDirection::Positive,
            [third, 20.0],
        ) {
            self.direction = MarkDirection::Positive;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Neg",
            self.direction == MarkDirection::Negative,
            [third, 20.0],
        ) {
            self.direction = MarkDirection::Negative;
        }
        ui.same_line_with_spacing(0.0, 4.0);
        if toggle_button(
            ui,
            "Sym",
            self.direction == MarkDirection::Symmetric,
            [third, 20.0],
        ) {
            self.direction = MarkDirection::Symmetric;
        }
    }

    pub(super) fn draw_footer_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        popup_active: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let req = self.request(state);
        let preview_result = req.as_ref().and_then(|current| {
            state
                .mark_preview
                .as_ref()
                .filter(|preview| preview.request == *current)
                .map(|preview| &preview.result)
        });
        let has_geometry = self.has_any_geometry();
        let can_commit = has_geometry
            && req.as_ref().is_some_and(|current| {
                GuiShellAction::Mark(GuiMarkAction::Commit(Box::new(current.clone())))
                    .dispatchable(state)
            });
        let failed_message = match preview_result {
            Some(GuiMarkPreview::Failed(message)) if has_geometry => Some(message.clone()),
            _ => None,
        };
        let can_ok = can_commit || failed_message.is_some();
        if crate::theme::property_accept(ui, can_ok) {
            if let Some(message) = failed_message {
                self.modals.show_edit_failed(message);
                return;
            }
            if let Some(current) = req.clone() {
                self.submitted = Some(current.clone());
                self.reopen_after_commit = false;
                actions.push(GuiShellAction::Mark(GuiMarkAction::Commit(Box::new(
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
        if widget_abi::button(ui, "+##mark-apply", [plus_w, 0.0])
            && let Some(current) = req
        {
            self.submitted = Some(current.clone());
            self.reopen_after_commit = true;
            actions.push(GuiShellAction::Mark(GuiMarkAction::Commit(Box::new(
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
