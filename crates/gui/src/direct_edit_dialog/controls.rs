// SPDX-License-Identifier: GPL-2.0-only
//! ImGui controls for the floating Direct Edit toolbar (#3918).
use super::{DirectEditDialog, GuiDirectEditAction, GuiDirectEditPreview};
use crate::{GuiCommandState, GuiShellAction, widget_abi};
use imgui::{StyleColor, StyleVar, Ui};
use oblikovati_model::feature::direct_edit::{
    DirectEditCoordinateMode, DirectEditMode, DirectEditScaleMode, DirectEditSelectionFilter,
};

const ACTIVE_BLUE: [f32; 4] = [0.24, 0.45, 0.78, 1.0];
const PICK_TEAL: [f32; 4] = [0.18, 0.56, 0.62, 1.0];
const APPLY_GREEN: [f32; 4] = [0.16, 0.62, 0.30, 1.0];
const CANCEL_RED: [f32; 4] = [0.78, 0.22, 0.22, 1.0];
const ERROR_RED: [f32; 4] = [0.92, 0.24, 0.24, 1.0];

const FILTER_POPUP: &str = "##direct-edit-filter-popup";
const COORD_POPUP: &str = "##direct-edit-coord-popup";

impl DirectEditDialog {
    pub(super) fn draw_mode_bar(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let modes = [
            DirectEditMode::Move,
            DirectEditMode::Size,
            DirectEditMode::Scale,
            DirectEditMode::Rotate,
            DirectEditMode::Delete,
        ];
        for (idx, mode) in modes.into_iter().enumerate() {
            if idx > 0 {
                ui.same_line_with_spacing(0.0, 4.0);
            }
            let active = self.mode == mode;
            let _bg = active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
            let _border = active.then(|| ui.push_style_var(StyleVar::FrameBorderSize(1.5)));
            if widget_abi::button(
                ui,
                &format!("{}##direct-mode-{idx}", mode.label()),
                [60.0, 26.0],
            ) && !active
            {
                self.switch_mode(mode, actions);
            }
        }
    }

    pub(super) fn draw_selection_row(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let filter_label = format!("{} v##direct-filter-btn", self.filter.label());
        if widget_abi::button(ui, &filter_label, [84.0, 24.0]) {
            ui.open_popup(FILTER_POPUP);
        }
        if let Some(_popup) = ui.begin_popup(FILTER_POPUP) {
            for option in [
                DirectEditSelectionFilter::Faces,
                DirectEditSelectionFilter::Solids,
            ] {
                if ui
                    .menu_item_config(option.label())
                    .selected(self.filter == option)
                    .build()
                    && self.filter != option
                {
                    self.set_filter(option, actions);
                }
            }
        }
        ui.same_line_with_spacing(0.0, 8.0);
        let has_sel = self.has_geometry_selection();
        let pill = match (self.filter, self.faces.len(), self.body.is_some()) {
            (DirectEditSelectionFilter::Faces, 0, _) => "Select Face".to_owned(),
            (DirectEditSelectionFilter::Faces, 1, _) => "1 Face".to_owned(),
            (DirectEditSelectionFilter::Faces, n, _) => format!("{n} Faces"),
            (DirectEditSelectionFilter::Solids, _, true) => "1 Solid".to_owned(),
            (DirectEditSelectionFilter::Solids, _, false) => "Select Solid".to_owned(),
        };
        let _bg = has_sel.then(|| ui.push_style_color(StyleColor::Button, PICK_TEAL));
        widget_abi::button(ui, &format!("{pill}##direct-sel-pill"), [120.0, 24.0]);
    }

    pub(super) fn draw_mode_controls(&mut self, ui: &Ui) {
        match self.mode {
            DirectEditMode::Scale => self.draw_scale_controls(ui),
            DirectEditMode::Rotate => self.draw_rotate_controls(ui),
            DirectEditMode::Move => {
                ui.text_disabled(if self.has_geometry_selection() {
                    "Drag manipulator or switch selection filter"
                } else {
                    "Select a face or solid to move"
                });
            }
            DirectEditMode::Size => {
                ui.text_disabled(if self.has_geometry_selection() {
                    "Drag size manipulator to resize face"
                } else {
                    "Select a face to size"
                });
            }
            DirectEditMode::Delete => {
                ui.text_disabled(if self.has_geometry_selection() {
                    "Selected geometry ready for Delete"
                } else {
                    "Select faces to delete"
                });
            }
        }
    }

    fn draw_scale_controls(&mut self, ui: &Ui) {
        let uniform_active = self.scale_mode == DirectEditScaleMode::Uniform;
        let _u_bg = uniform_active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
        if widget_abi::button(ui, "Uniform##direct-scale-uniform", [96.0, 24.0]) && !uniform_active
        {
            self.scale_mode = DirectEditScaleMode::Uniform;
            self.coupled_from_uniform = false;
        }
        drop(_u_bg);
        ui.same_line_with_spacing(0.0, 6.0);
        let non_uniform_active = self.scale_mode == DirectEditScaleMode::NonUniform;
        let _nu_bg =
            non_uniform_active.then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
        if widget_abi::button(ui, "Non Uniform##direct-scale-nonuniform", [112.0, 24.0])
            && !non_uniform_active
        {
            self.scale_mode = DirectEditScaleMode::NonUniform;
            let seed = self.uniform_factor.clone();
            self.scale_factors = [seed.clone(), seed.clone(), seed];
            self.coupled_from_uniform = true;
        }
        drop(_nu_bg);

        ui.spacing();
        let _numeric = crate::ui_scale::numeric_font(ui);
        let _border = ui.push_style_var(StyleVar::FrameBorderSize(1.0));
        match self.scale_mode {
            DirectEditScaleMode::Uniform => {
                ui.text("Factor");
                ui.same_line_with_spacing(0.0, 8.0);
                let _w = ui.push_item_width(120.0);
                ui.input_text("##direct-scale-factor", &mut self.uniform_factor)
                    .build();
            }
            DirectEditScaleMode::NonUniform => {
                let labels = [
                    ("X", "##direct-scale-x"),
                    ("Y", "##direct-scale-y"),
                    ("Z", "##direct-scale-z"),
                ];
                for (axis, (label, id)) in labels.into_iter().enumerate() {
                    if axis > 0 {
                        ui.same_line_with_spacing(0.0, 8.0);
                    }
                    ui.text(label);
                    ui.same_line_with_spacing(0.0, 4.0);
                    let _w = ui.push_item_width(68.0);
                    let before = self.scale_factors[axis].clone();
                    if ui.input_text(id, &mut self.scale_factors[axis]).build()
                        && axis > 0
                        && self.scale_factors[axis] != before
                    {
                        self.coupled_from_uniform = false;
                    }
                }
            }
        }
    }

    fn draw_rotate_controls(&mut self, ui: &Ui) {
        let coord_label = format!("{} v##direct-coord-btn", self.coordinate_mode.label());
        if widget_abi::button(ui, &coord_label, [88.0, 24.0]) {
            ui.open_popup(COORD_POPUP);
        }
        if let Some(_popup) = ui.begin_popup(COORD_POPUP) {
            for option in [
                DirectEditCoordinateMode::World,
                DirectEditCoordinateMode::Local,
            ] {
                if ui
                    .menu_item_config(option.label())
                    .selected(self.coordinate_mode == option)
                    .build()
                {
                    self.coordinate_mode = option;
                }
            }
        }
        ui.same_line_with_spacing(0.0, 8.0);
        let _snap_bg = self
            .snap_parallel
            .then(|| ui.push_style_color(StyleColor::Button, ACTIVE_BLUE));
        if widget_abi::button(ui, "Snap Parallel##direct-snap-parallel", [112.0, 24.0]) {
            self.snap_parallel = true;
        }
    }

    pub(super) fn draw_action_bar(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let Some(preview) = &state.direct_edit_preview
            && let GuiDirectEditPreview::Failed(msg) = &preview.result
        {
            ui.text_colored(ERROR_RED, format!("! {msg}"));
        }
        let req = self.commit_request(state);
        let can_apply = req.as_ref().is_some_and(|r| {
            GuiShellAction::DirectEdit(GuiDirectEditAction::Commit(Box::new(r.clone())))
                .dispatchable(state)
        });
        let _dis = ui.begin_disabled(!can_apply);
        let _green = ui.push_style_color(StyleColor::Button, APPLY_GREEN);
        if widget_abi::button(ui, "+##direct-edit-apply", [36.0, 26.0])
            && let Some(r) = req
        {
            self.submitted_generation = Some(r.generation);
            actions.push(GuiShellAction::DirectEdit(GuiDirectEditAction::Commit(
                Box::new(r),
            )));
        }
        drop(_green);
        drop(_dis);
        if ui.is_item_hovered() {
            ui.tooltip_text("Apply Direct Edit");
        }
        ui.same_line_with_spacing(0.0, 8.0);
        let _red = ui.push_style_color(StyleColor::Button, CANCEL_RED);
        if widget_abi::button(ui, "X##direct-edit-cancel", [36.0, 26.0]) {
            self.handle_red_x(actions);
        }
        drop(_red);
        if ui.is_item_hovered() {
            ui.tooltip_text(if self.has_active_selection() {
                "Clear selection"
            } else {
                "Close Direct Edit"
            });
        }
    }
}
