// SPDX-License-Identifier: GPL-2.0-only
//! The Precise Input toolbar: exact coordinates for the running command's next point.
use crate::{GuiCommandState, GuiShellAction};
use imgui::{StyleColor, Ui, WindowFlags};

/// Typed coordinates in the document's length units; the owner evaluates them.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiPrecisePoint {
    pub x: String,
    pub y: String,
}
const OPTIONS: &str = "##precise-input-options";
const FIELD_WIDTH: f32 = 80.;
const INVALID_TEXT: [f32; 4] = [0.95, 0.2, 0.2, 1.];

/// Whether the toolbar is up and what its fields hold.
#[derive(Default)]
pub(crate) struct PreciseInputBar {
    pub open: bool,
    x: String,
    y: String,
}
impl PreciseInputBar {
    /// The ribbon's way in; example: `bar.toggle(ui)` inside the Format panel.
    pub(crate) fn toggle(&mut self, ui: &Ui) {
        if crate::widget_abi::toggle_button(ui, "Precise Input", [108., 18.], self.open) {
            self.open = !self.open;
        }
    }
    /// Draw while open; example: `bar.panel(ui, state, actions)` once per frame.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.open || state.authoring != crate::GuiAuthoringContext::Sketch {
            return;
        }
        let [width, _] = ui.io().display_size;
        ui.window("Precise Input")
            .position([width * 0.5, 200.], imgui::Condition::FirstUseEver)
            .flags(WindowFlags::ALWAYS_AUTO_RESIZE | WindowFlags::NO_COLLAPSE)
            .build(|| {
                let accepted = self.fields(ui);
                self.options(ui);
                if accepted && valid_number(&self.x) && valid_number(&self.y) {
                    actions.push(GuiShellAction::PrecisePoint(GuiPrecisePoint {
                        x: std::mem::take(&mut self.x),
                        y: std::mem::take(&mut self.y),
                    }));
                }
            });
    }
    fn fields(&mut self, ui: &Ui) -> bool {
        let first = coordinate_field(ui, "X", &mut self.x);
        ui.same_line();
        let second = coordinate_field(ui, "Y", &mut self.y);
        ui.same_line();
        first || second
    }
    fn options(&mut self, ui: &Ui) {
        if ui.button("...##precise-input-options-button") {
            ui.open_popup(OPTIONS);
        }
        if let Some(_menu) = ui.begin_popup(OPTIONS)
            && ui.menu_item("Close")
        {
            self.open = false;
        }
    }
}
fn coordinate_field(ui: &Ui, label: &str, text: &mut String) -> bool {
    let invalid = !text.is_empty() && !valid_number(text);
    let _red = invalid.then(|| ui.push_style_color(StyleColor::Text, INVALID_TEXT));
    ui.set_next_item_width(FIELD_WIDTH);
    ui.input_text(label, text).enter_returns_true(true).build()
}
/// A coordinate is a plain number here; an empty field is not yet an answer.
pub(crate) fn valid_number(text: &str) -> bool {
    text.trim().parse::<f64>().is_ok_and(f64::is_finite)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn only_a_finite_number_is_a_coordinate() {
        assert!(valid_number("5"));
        assert!(valid_number(" -6.25 "));
        assert!(!valid_number("a"));
        assert!(!valid_number(""));
        assert!(!valid_number("inf"));
    }
}
