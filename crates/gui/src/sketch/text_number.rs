// SPDX-License-Identifier: GPL-2.0-only
//! Keep rejected formatting edits local and preserve the valid live draft.
use imgui::{Key, Ui};

pub(super) struct TextNumberSpec {
    pub label: &'static str,
    pub title: &'static str,
    pub message: &'static str,
    pub accepts: fn(f64) -> bool,
}

pub(super) const HEIGHT: TextNumberSpec = TextNumberSpec {
    label: "Height (mm)",
    title: "Text Height",
    message: "Enter a size greater than 0.01 mm.",
    accepts: |height| height.is_finite() && height > 0.01,
};

pub(super) const WIDTH: TextNumberSpec = TextNumberSpec {
    label: "Width (%)",
    title: "Text Width",
    message: "Width is out of the expected range. Enter a positive width.",
    accepts: |width| oblikovati_model::sketch::sketch_text_width_factor(width).is_some(),
};

#[derive(Default)]
pub(super) struct TextNumberInput {
    pending: Option<String>,
    warning: bool,
    request_warning: bool,
    refocus: bool,
}

impl TextNumberInput {
    pub(super) fn draw(&mut self, ui: &Ui, accepted: &mut String, spec: &TextNumberSpec) -> bool {
        let mut visible = self.pending.as_ref().unwrap_or(accepted).clone();
        ui.set_next_item_width(145.);
        if std::mem::take(&mut self.refocus) {
            ui.set_keyboard_focus_here();
        }
        if ui
            .input_text(spec.label, &mut visible)
            .auto_select_all(true)
            .build()
        {
            self.pending = Some(visible);
        }
        ui.is_item_deactivated_after_edit() && self.apply(accepted, spec)
    }

    pub(super) fn apply(&mut self, accepted: &mut String, spec: &TextNumberSpec) -> bool {
        let Some(pending) = self.pending.take() else {
            return false;
        };
        if pending.trim().parse::<f64>().is_ok_and(spec.accepts) {
            *accepted = pending;
            return true;
        }
        self.pending = Some(pending);
        self.request_warning |= !self.warning;
        self.warning = true;
        false
    }

    pub(super) fn blocked(&self) -> bool {
        self.warning
    }

    pub(super) fn choose_preset(&mut self, accepted: &mut String, preset: &str) {
        self.pending = None;
        self.warning = false;
        self.request_warning = false;
        *accepted = preset.into();
    }

    pub(super) fn warning(&mut self, ui: &Ui, spec: &TextNumberSpec) {
        if !self.warning {
            return;
        }
        if std::mem::take(&mut self.request_warning) {
            ui.open_popup(spec.title);
        }
        let Some(_popup) = ui
            .modal_popup_config(spec.title)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        ui.text(spec.message);
        if ui.button("OK") || ui.is_key_pressed_no_repeat(Key::Enter) {
            ui.close_current_popup();
            self.warning = false;
            self.refocus = true;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn valid_height_replaces_the_draft_only_after_the_edit() {
        let mut height = TextNumberInput {
            pending: Some("6.10".into()),
            ..Default::default()
        };
        let mut accepted = "3.05".into();
        assert!(height.apply(&mut accepted, &HEIGHT));
        assert_eq!(accepted, "6.10");
    }

    #[test]
    fn invalid_height_keeps_the_previous_value_and_the_unaccepted_text() {
        for candidate in ["0.01", "0", "-5", "", "NaN", "Inf", "5 mm"] {
            let mut height = TextNumberInput {
                pending: Some(candidate.into()),
                ..Default::default()
            };
            let mut accepted = "6.10".into();
            assert!(!height.apply(&mut accepted, &HEIGHT));
            assert!(height.blocked());
            assert!(!height.apply(&mut accepted, &HEIGHT));
            assert!(height.request_warning);
            assert_eq!(height.pending.as_deref(), Some(candidate));
            assert_eq!(accepted, "6.10");
        }
    }

    #[test]
    fn width_declines_zero_and_values_the_native_font_cannot_represent() {
        for value in ["0", "-1", "NaN", "1e300", "1e-300"] {
            let mut input = TextNumberInput {
                pending: Some(value.into()),
                ..Default::default()
            };
            let mut accepted = "50".into();
            assert!(!input.apply(&mut accepted, &WIDTH));
            assert_eq!(accepted, "50");
            assert!(input.blocked());
        }
    }
}
