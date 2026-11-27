// SPDX-License-Identifier: GPL-2.0-only
//! Transient STL Import Options modal launched from the Import file picker.
use crate::widget_abi;
use imgui::Ui;
use oblikovati_model::feature::imported_model::{StlImportOptions, StlImportUnits};

pub(super) const STL_OPTIONS_TITLE: &str = "STL Import Options";

pub(super) enum StlOptionsOutcome {
    Pending,
    Accepted(StlImportOptions),
    Cancelled,
}

#[derive(Default)]
pub(super) struct StlOptionsModal {
    open: bool,
    opening: bool,
    draft: StlImportOptions,
    rgb_float: [f32; 3],
}

impl StlOptionsModal {
    pub(super) fn is_open(&self) -> bool {
        self.open
    }

    pub(super) fn open(&mut self, options: &StlImportOptions) {
        self.draft = options.clone();
        self.rgb_float = [
            f32::from(options.color_rgb[0]) / 255.0,
            f32::from(options.color_rgb[1]) / 255.0,
            f32::from(options.color_rgb[2]) / 255.0,
        ];
        self.open = true;
        self.opening = true;
    }

    pub(super) fn update_inspection(&mut self, bounding_box: [f64; 3], template_units: String) {
        self.draft.bounding_box = bounding_box;
        self.draft.template_units = template_units;
    }

    pub(super) fn close(&mut self) {
        self.open = false;
        self.opening = false;
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> StlOptionsOutcome {
        if !self.open {
            return StlOptionsOutcome::Pending;
        }
        if std::mem::take(&mut self.opening) {
            ui.open_popup(STL_OPTIONS_TITLE);
        }
        let Some(_popup) = ui
            .modal_popup_config(STL_OPTIONS_TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return StlOptionsOutcome::Pending;
        };

        crate::theme::property_label(ui, "Import Units");
        ui.set_next_item_width(180.);
        if let Some(_combo) = ui.begin_combo("##stl-import-units", self.draft.units.label()) {
            for unit in StlImportUnits::ALL {
                if ui
                    .selectable_config(unit.label())
                    .selected(self.draft.units == unit)
                    .build()
                {
                    self.draft.units = unit;
                }
            }
        }

        crate::theme::property_label(ui, "Template Units");
        ui.text(&self.draft.template_units);

        let [bx, by, bz] = self.draft.bounding_box;
        crate::theme::property_label(ui, "Bounding Box");
        ui.text(format!("{bx:.2} x {by:.2} x {bz:.2}"));

        ui.separator();
        ui.checkbox("Translation Report", &mut self.draft.translation_report);
        ui.checkbox("Color", &mut self.draft.color_enabled);

        {
            let _disabled = (!self.draft.color_enabled).then(|| ui.begin_disabled(true));
            crate::theme::property_label(ui, "RGB");
            ui.set_next_item_width(180.);
            if ui.color_edit3("##stl-color-rgb", &mut self.rgb_float) {
                self.draft.color_rgb = [
                    (self.rgb_float[0].clamp(0.0, 1.0) * 255.0).round() as u8,
                    (self.rgb_float[1].clamp(0.0, 1.0) * 255.0).round() as u8,
                    (self.rgb_float[2].clamp(0.0, 1.0) * 255.0).round() as u8,
                ];
            }
        }

        ui.separator();
        let ok = widget_abi::button(ui, "OK", [72., 28.]);
        ui.same_line_with_spacing(0., 8.);
        let cancel = widget_abi::button(ui, "Cancel", [72., 28.])
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape);
        if ok {
            ui.close_current_popup();
            self.close();
            return StlOptionsOutcome::Accepted(self.draft.clone());
        }
        if cancel {
            ui.close_current_popup();
            self.close();
            return StlOptionsOutcome::Cancelled;
        }
        StlOptionsOutcome::Pending
    }
}
