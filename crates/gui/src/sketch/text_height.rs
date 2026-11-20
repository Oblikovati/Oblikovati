// SPDX-License-Identifier: GPL-2.0-only
//! Keep an invalid text height local to the control and keep the last valid draft visible.
use imgui::{Key, Ui};

pub(super) const LABEL: &str = "Height (mm)";
const TITLE: &str = "Text Height";
const MINIMUM_MM: f64 = 0.01;

#[derive(Default)]
pub(super) struct TextHeightInput {
    pending: Option<String>,
    warning: bool,
    request_warning: bool,
    refocus: bool,
}

impl TextHeightInput {
    pub(super) fn draw(&mut self, ui: &Ui, accepted: &mut String) -> bool {
        let mut visible = self.pending.as_ref().unwrap_or(accepted).clone();
        ui.set_next_item_width(145.);
        if std::mem::take(&mut self.refocus) {
            ui.set_keyboard_focus_here();
        }
        if ui
            .input_text(LABEL, &mut visible)
            .auto_select_all(true)
            .build()
        {
            self.pending = Some(visible);
        }
        ui.is_item_deactivated_after_edit() && self.apply(accepted)
    }

    pub(super) fn apply(&mut self, accepted: &mut String) -> bool {
        let Some(pending) = self.pending.take() else {
            return false;
        };
        if valid_height(&pending) {
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

    pub(super) fn warning(&mut self, ui: &Ui) {
        if !self.warning {
            return;
        }
        if std::mem::take(&mut self.request_warning) {
            ui.open_popup(TITLE);
        }
        let Some(_popup) = ui
            .modal_popup_config(TITLE)
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        ui.text("Enter a size greater than 0.01 mm.");
        if ui.button("OK") || ui.is_key_pressed_no_repeat(Key::Enter) {
            ui.close_current_popup();
            self.warning = false;
            self.refocus = true;
        }
    }
}

fn valid_height(value: &str) -> bool {
    value
        .trim()
        .parse::<f64>()
        .is_ok_and(|height| height.is_finite() && height > MINIMUM_MM)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn valid_height_replaces_the_draft_only_after_the_edit() {
        let mut height = TextHeightInput {
            pending: Some("6.10".into()),
            ..Default::default()
        };
        let mut accepted = "3.05".into();
        assert!(height.apply(&mut accepted));
        assert_eq!(accepted, "6.10");
    }

    #[test]
    fn invalid_height_keeps_the_previous_value_and_the_unaccepted_text() {
        for candidate in ["0.01", "0", "-5", "", "NaN", "Inf", "5 mm"] {
            let mut height = TextHeightInput {
                pending: Some(candidate.into()),
                ..Default::default()
            };
            let mut accepted = "6.10".into();
            assert!(!height.apply(&mut accepted));
            assert!(height.blocked());
            assert!(!height.apply(&mut accepted));
            assert!(height.request_warning);
            assert_eq!(height.pending.as_deref(), Some(candidate));
            assert_eq!(accepted, "6.10");
        }
    }
}
