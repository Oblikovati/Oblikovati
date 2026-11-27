// SPDX-License-Identifier: GPL-2.0-only
//! Coil Size, Rotation, and Coil Ends controls for the Coil Properties panel.
use super::CoilDialog;
use crate::{GuiCoilPreview, feature_properties::draw_error_badge, widget_abi};
use imgui::Ui;
use oblikovati_model::feature::sketch_coil::{CoilHandedness, CoilMethod};

const METHODS: [(CoilMethod, &str); 4] = [
    (CoilMethod::PitchAndRevolution, "Pitch and Revolution"),
    (CoilMethod::RevolutionAndHeight, "Revolution and Height"),
    (CoilMethod::PitchAndHeight, "Pitch and Height"),
    (CoilMethod::Spiral, "Spiral"),
];

pub(super) fn method_label(method: CoilMethod) -> &'static str {
    match method {
        CoilMethod::PitchAndRevolution => "Pitch and Revolution",
        CoilMethod::RevolutionAndHeight => "Revolution and Height",
        CoilMethod::PitchAndHeight => "Pitch and Height",
        CoilMethod::Spiral => "Spiral",
    }
}

impl CoilDialog {
    pub(super) fn draw_input_geometry(&mut self, ui: &Ui) {
        if !ui.collapsing_header("Input Geometry", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        crate::theme::property_label(ui, "Profiles");
        ui.text(if self.profile_cleared {
            "Select Profiles"
        } else {
            "1 Profile"
        });
        if !self.profile_cleared {
            ui.same_line_with_spacing(0., 12.);
            if widget_abi::button(ui, "ClearProfile", [0.; 2]) {
                self.profile_cleared = true;
            }
        }
        crate::theme::property_label(ui, "Axis");
        ui.text(if self.axis.is_some() {
            "1 Axis"
        } else {
            "Select Axis"
        });
        if self.axis.is_some() {
            ui.same_line_with_spacing(0., 8.);
            if widget_abi::button(ui, "FlipAxis", [0.; 2]) {
                self.axis_reversed = !self.axis_reversed;
            }
            ui.same_line_with_spacing(0., 8.);
            if widget_abi::button(ui, "ClearAxis", [0.; 2]) {
                self.axis = None;
            }
        }
    }

    pub(super) fn draw_coil_size(&mut self, ui: &Ui, result: Option<&GuiCoilPreview>) {
        if !ui.collapsing_header("Coil Size", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        crate::theme::property_label(ui, "Method");
        ui.set_next_item_width(-1.);
        if let Some(_combo) = ui.begin_combo("##coil-method", method_label(self.method)) {
            for (method, label) in METHODS {
                if ui
                    .selectable_config(label)
                    .selected(self.method == method)
                    .build()
                {
                    self.method = method;
                    self.presets.clear_selection();
                }
            }
        }
        let error = match result {
            Some(GuiCoilPreview::Failed(msg)) => Some(msg.as_str()),
            _ => None,
        };
        let show_pitch = matches!(
            self.method,
            CoilMethod::PitchAndRevolution | CoilMethod::PitchAndHeight | CoilMethod::Spiral
        );
        let show_rev = matches!(
            self.method,
            CoilMethod::PitchAndRevolution | CoilMethod::RevolutionAndHeight | CoilMethod::Spiral
        );
        let show_height = matches!(
            self.method,
            CoilMethod::RevolutionAndHeight | CoilMethod::PitchAndHeight
        );
        if show_pitch {
            draw_scalar_field(
                ui,
                "Pitch",
                "##coil-pitch",
                &mut self.pitch,
                error.filter(|msg| msg.contains("pitch")),
                &mut self.presets,
            );
        }
        if show_rev {
            draw_scalar_field(
                ui,
                "Revolution",
                "##coil-revolution",
                &mut self.revolutions,
                error.filter(|msg| msg.contains("revolution")),
                &mut self.presets,
            );
        }
        if show_height {
            draw_scalar_field(
                ui,
                "Height",
                "##coil-height",
                &mut self.height,
                error.filter(|msg| msg.contains("height")),
                &mut self.presets,
            );
        }
        if self.method != CoilMethod::Spiral {
            draw_scalar_field(
                ui,
                "Taper",
                "##coil-taper",
                &mut self.taper,
                error.filter(|msg| msg.contains("taper")),
                &mut self.presets,
            );
        }
        crate::theme::property_label(ui, "Rotation");
        if widget_abi::button(ui, "Left-hand", [0.; 2]) {
            self.handedness = CoilHandedness::Left;
            self.presets.clear_selection();
        }
        ui.same_line();
        if widget_abi::button(ui, "Right-hand", [0.; 2]) {
            self.handedness = CoilHandedness::Right;
            self.presets.clear_selection();
        }
        if let Some(msg) = error
            && !msg.contains("pitch")
            && !msg.contains("revolution")
            && !msg.contains("height")
            && !msg.contains("taper")
        {
            draw_error_badge(ui, msg);
        }
    }

    pub(super) fn draw_coil_ends(&mut self, ui: &Ui) {
        if !ui.collapsing_header("Coil Ends", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        if ui.checkbox("Close Start", &mut self.close_start) {
            self.presets.clear_selection();
        }
        if self.close_start {
            draw_end_field(
                ui,
                "Start Flat Angle",
                "##coil-start-flat",
                &mut self.start_flat,
            );
            draw_end_field(
                ui,
                "Start Transition Angle",
                "##coil-start-trans",
                &mut self.start_transition,
            );
        }
        if ui.checkbox("Close End", &mut self.close_end) {
            self.presets.clear_selection();
        }
        if self.close_end {
            draw_end_field(ui, "End Flat Angle", "##coil-end-flat", &mut self.end_flat);
            draw_end_field(
                ui,
                "End Transition Angle",
                "##coil-end-trans",
                &mut self.end_transition,
            );
        }
    }
}

fn draw_scalar_field<F: Clone>(
    ui: &Ui,
    label: &str,
    id: &str,
    value: &mut String,
    error: Option<&str>,
    presets: &mut crate::feature_properties::SessionPresets<F>,
) {
    crate::theme::property_label(ui, label);
    let badge = if error.is_some() { 26. } else { 0. };
    ui.set_next_item_width((ui.content_region_avail()[0] - badge).max(60.));
    if ui.input_text(id, value).build() {
        presets.clear_selection();
    }
    if let Some(message) = error {
        draw_error_badge(ui, message);
    }
}

fn draw_end_field(ui: &Ui, label: &str, id: &str, value: &mut String) {
    crate::theme::property_label(ui, label);
    ui.set_next_item_width(-1.);
    ui.input_text(id, value).build();
}
