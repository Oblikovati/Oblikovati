// SPDX-License-Identifier: GPL-2.0-only
//! ImGui controls for the Part Modify Finish Properties panel (#3918).
use super::{
    FinishDialog, GuiFinishAction, GuiFinishPreview, GuiFinishSelectionMode, GuiFinishType, catalog,
};
use crate::{GuiCommandState, GuiShellAction};
use imgui::{StyleColor, Ui};

const PICK_TEAL: [f32; 4] = [0.18, 0.56, 0.62, 1.0];
const INACTIVE_PILL: [f32; 4] = [0.24, 0.27, 0.32, 1.0];
const ACTIVE_TYPE_BG: [f32; 4] = [0.24, 0.34, 0.72, 1.0];

pub(super) fn format_area_mm2(area_mm2: f64) -> String {
    if !area_mm2.is_finite() || area_mm2 <= 1e-6 {
        return "0.000 mm^2".to_owned();
    }
    let rounded = area_mm2.round();
    if (area_mm2 - rounded).abs() < 1e-3 {
        format!("{} mm^2", rounded as i64)
    } else {
        format!("{area_mm2:.3} mm^2")
    }
}

fn draw_text_row(ui: &Ui, label: &str, id: &str, value: &mut String) {
    crate::theme::property_label(ui, label);
    ui.set_next_item_width(-1.0);
    let _ = ui.input_text(id, value).build();
}

fn entity_pill_label(count: usize) -> String {
    match count {
        0 => "Select Faces".to_owned(),
        1 => "1 Entity".to_owned(),
        n => format!("{n} Entities"),
    }
}

impl FinishDialog {
    pub(super) fn draw_input_geometry(&mut self, ui: &Ui, state: &GuiCommandState) {
        let header = if self.input_geometry_open {
            "v Input Geometry##finish-input-hdr"
        } else {
            "> Input Geometry##finish-input-hdr"
        };
        if ui.button_with_size(header, [-1.0, 0.0]) {
            self.input_geometry_open = !self.input_geometry_open;
        }
        if !self.input_geometry_open {
            return;
        }
        self.draw_selector_row(
            ui,
            "Include",
            GuiFinishSelectionMode::Include,
            self.included_faces.len(),
            "finish-include",
        );
        self.draw_selector_row(
            ui,
            "Exclude",
            GuiFinishSelectionMode::Exclude,
            self.excluded_faces.len(),
            "finish-exclude",
        );
        let area_mm2 = self.displayed_area_mm2(state);
        crate::theme::property_label(ui, "Area");
        ui.text(format_area_mm2(area_mm2));
    }

    fn draw_selector_row(
        &mut self,
        ui: &Ui,
        label: &str,
        mode: GuiFinishSelectionMode,
        count: usize,
        id_suffix: &str,
    ) {
        crate::theme::property_label(ui, label);
        let pill = entity_pill_label(count);
        let bg = if self.selection_mode == mode {
            PICK_TEAL
        } else {
            INACTIVE_PILL
        };
        let _color = ui.push_style_color(StyleColor::Button, bg);
        if ui.button(format!("{pill}##{id_suffix}-pick")) {
            self.selection_mode = mode;
        }
        drop(_color);
        if count > 0 {
            ui.same_line_with_spacing(0.0, 6.0);
            if ui.button(format!("x##{id_suffix}-clear")) {
                match mode {
                    GuiFinishSelectionMode::Include => {
                        self.included_faces.clear();
                        self.excluded_faces.clear();
                        self.body = None;
                        self.last_area_mm2 = 0.0;
                    }
                    GuiFinishSelectionMode::Exclude => {
                        self.excluded_faces.clear();
                    }
                }
            }
        }
    }

    fn displayed_area_mm2(&self, state: &GuiCommandState) -> f64 {
        if self.included_faces.is_empty() {
            return 0.0;
        }
        if let Some(preview) = &state.finish_preview
            && let GuiFinishPreview::Ready { area_mm2 } = preview.result
            && preview.request.included_faces == self.included_faces
            && preview.request.excluded_faces == self.excluded_faces
        {
            return area_mm2;
        }
        self.last_area_mm2
    }

    pub(super) fn draw_behavior(&mut self, ui: &Ui) {
        let header = if self.behavior_open {
            "v Behavior##finish-behavior-hdr"
        } else {
            "> Behavior##finish-behavior-hdr"
        };
        if ui.button_with_size(header, [-1.0, 0.0]) {
            self.behavior_open = !self.behavior_open;
        }
        if !self.behavior_open {
            return;
        }
        self.draw_type_selector(ui);
        self.draw_process_combo(ui);
        match self.finish_type {
            GuiFinishType::HeatTreatment => {
                draw_text_row(ui, "Hardness", "##finish-hardness", &mut self.hardness);
                draw_text_row(ui, "Depth", "##finish-depth", &mut self.depth);
            }
            GuiFinishType::Appearance | GuiFinishType::MaterialCoating | GuiFinishType::Paint => {
                self.draw_appearance_combo(ui);
                draw_text_row(ui, "Thickness", "##finish-thickness", &mut self.thickness);
            }
            GuiFinishType::SurfaceTexture => {
                self.draw_appearance_combo(ui);
            }
        }
        draw_text_row(
            ui,
            "Description",
            "##finish-description",
            &mut self.description,
        );
        draw_text_row(
            ui,
            "Short Description",
            "##finish-short-description",
            &mut self.short_description,
        );
        draw_text_row(ui, "Comment", "##finish-comment", &mut self.comment);
    }

    fn draw_type_selector(&mut self, ui: &Ui) {
        ui.text_disabled(format!("Type: {}", self.finish_type.label()));
        let avail = ui.content_region_avail()[0];
        let gap = 4.0;
        let width = ((avail - gap * 4.0) / 5.0).max(24.0);
        for (idx, kind) in GuiFinishType::ALL.iter().enumerate() {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, gap);
            }
            let active = self.finish_type == *kind;
            let color = active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_TYPE_BG));
            let label = format!("T{}##finish-type-{}", kind.ordinal(), kind.ordinal());
            if ui.button_with_size(&label, [width, 22.0]) && !active {
                self.select_finish_type(*kind);
            }
            drop(color);
            if ui.is_item_hovered() {
                ui.tooltip_text(kind.label());
            }
        }
    }

    fn draw_process_combo(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Process");
        let presets = catalog::presets_for(self.finish_type);
        ui.set_next_item_width(-1.0);
        if let Some(_combo) = ui.begin_combo("##finish-process", &self.process) {
            for preset in presets {
                let selected = self.process == preset.process;
                if ui
                    .selectable_config(preset.process)
                    .selected(selected)
                    .build()
                {
                    self.apply_preset(preset);
                }
            }
        }
    }

    fn draw_appearance_combo(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Appearance");
        ui.set_next_item_width(-1.0);
        if let Some(_combo) = ui.begin_combo("##finish-appearance", &self.appearance) {
            for &choice in catalog::APPEARANCE_CHOICES {
                let selected = self.appearance == choice;
                if ui.selectable_config(choice).selected(selected).build() {
                    self.appearance = choice.to_owned();
                }
            }
        }
    }

    pub(super) fn draw_advanced_properties(&mut self, ui: &Ui) {
        let header = if self.advanced_open {
            "v Advanced Properties##finish-adv-hdr"
        } else {
            "> Advanced Properties##finish-adv-hdr"
        };
        if ui.button_with_size(header, [-1.0, 0.0]) {
            self.advanced_open = !self.advanced_open;
        }
        if !self.advanced_open {
            return;
        }
        ui.checkbox(
            "Export all Finish Parameters##finish-export-all",
            &mut self.export_all_parameters,
        );
    }

    pub(super) fn draw_footer(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let req = self.request(state);
        let ok_enabled = !self.included_faces.is_empty()
            && req.as_ref().is_some_and(|r| {
                GuiShellAction::Finish(GuiFinishAction::Commit(Box::new(r.clone())))
                    .dispatchable(state)
            });
        let _disabled = ui.begin_disabled(!ok_enabled);
        let ok_clicked = ui.button_with_size("OK##finish-ok", [72.0, 28.0]);
        drop(_disabled);
        if ok_clicked && let Some(r) = req {
            self.submitted = Some(r.clone());
            actions.push(GuiShellAction::Finish(GuiFinishAction::Commit(Box::new(r))));
        }
        ui.same_line_with_spacing(0.0, 6.0);
        if ui.button_with_size("Cancel##finish-cancel", [68.0, 28.0]) {
            self.close(actions);
        }
    }
}
