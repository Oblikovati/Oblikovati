// SPDX-License-Identifier: GPL-2.0-only
//! Signed Taper A draft for an existing single-sided extrusion.
use imgui::Ui;

pub(super) struct TaperDraft {
    original: String,
    value: String,
}

impl TaperDraft {
    pub(super) fn new(value: String) -> Self {
        Self {
            original: value.clone(),
            value,
        }
    }

    pub(super) fn change(&self) -> Option<String> {
        (self.value.trim() != self.original.trim()).then(|| self.value.trim().to_owned())
    }

    pub(super) fn is_empty(&self) -> bool {
        self.value.trim().is_empty()
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> bool {
        crate::theme::property_label(ui, "Taper A");
        let sign_width = 28.;
        let spacing = ui.clone_style().item_spacing[0];
        ui.set_next_item_width((ui.content_region_avail()[0] - spacing - sign_width).max(1.));
        let numeric = crate::ui_scale::numeric_font(ui);
        let changed = ui
            .input_text("##taper-a", &mut self.value)
            .auto_select_all(true)
            .build();
        drop(numeric);
        ui.same_line();
        let value = self.value.trim();
        let literal = value
            .strip_suffix("deg")
            .or_else(|| value.strip_suffix("rad"))
            .unwrap_or(value)
            .trim();
        let reversible = literal.parse::<f64>().is_ok_and(f64::is_finite);
        let _disabled = ui.begin_disabled(!reversible);
        let reverse = crate::widget_abi::button(ui, "±", [sign_width, ui.frame_height()]);
        if ui.is_item_hovered() {
            ui.tooltip_text("Reverse Taper A");
        }
        if reverse {
            self.value = match value.strip_prefix('-') {
                Some(positive) => positive.trim_start().to_owned(),
                None => format!("-{}", value.trim_start_matches('+')),
            };
        }
        changed || reverse
    }
}
