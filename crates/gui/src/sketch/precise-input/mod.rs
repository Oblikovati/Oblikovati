// SPDX-License-Identifier: GPL-2.0-only
//! The Precise Input toolbar: exact coordinates for the running command's next point.
use crate::{GuiCommandState, GuiShellAction};
use imgui::{StyleColor, Ui, WindowFlags};

/// How the two typed values name a point.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiPreciseMode {
    #[default]
    XAndY,
    XAndAngle,
    YAndAngle,
    DistanceAndAngle,
}
impl GuiPreciseMode {
    const ALL: [Self; 4] = [
        Self::XAndY,
        Self::XAndAngle,
        Self::YAndAngle,
        Self::DistanceAndAngle,
    ];
    fn label(self) -> &'static str {
        match self {
            Self::XAndY => "X and Y coordinate",
            Self::XAndAngle => "X coordinate and X axis angle",
            Self::YAndAngle => "Y coordinate and X axis angle",
            Self::DistanceAndAngle => "Distance and X axis angle",
        }
    }
    fn field_labels(self) -> [&'static str; 2] {
        match self {
            Self::XAndY => ["X", "Y"],
            Self::XAndAngle => ["X", "Angle"],
            Self::YAndAngle => ["Y", "Angle"],
            Self::DistanceAndAngle => ["Distance", "Angle"],
        }
    }
}
/// Typed values with what they mean; the owner evaluates and places them.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiPrecisePoint {
    pub first: String,
    pub second: String,
    pub mode: GuiPreciseMode,
    /// Measured from the triad rather than from the sketch origin.
    pub relative: bool,
}
/// What the toolbar asks of the triad.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiTriadCommand {
    Reposition,
    ResetToOrigin,
}
/// What a running command offers the toolbar.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiPreciseTarget {
    /// The pointer's position in the display unit, shown until a value is typed.
    pub pointer: Option<[String; 2]>,
    /// The triad's origin and its axis tips, in window pixels.
    pub triad: Option<[[i32; 2]; 3]>,
}
const OPTIONS: &str = "##precise-input-options";
const REFERENCE: &str = "##precise-input-reference";
const MODES: &str = "##precise-input-modes";
const FIELD_WIDTH: f32 = 80.;
const INVALID_TEXT: [f32; 4] = [0.95, 0.2, 0.2, 1.];

#[derive(Default)]
struct PreciseField {
    text: String,
    typed: bool,
    active: bool,
}
impl PreciseField {
    /// Follow the pointer until a person types; example: once per frame before drawing.
    fn follow(&mut self, pointer: Option<&String>) {
        if self.typed || self.active {
            return;
        }
        self.text.clear();
        if let Some(pointer) = pointer {
            self.text.push_str(pointer);
        }
    }
}
/// Whether the toolbar is up and what its fields hold.
#[derive(Default)]
pub(crate) struct PreciseInputBar {
    pub open: bool,
    first: PreciseField,
    second: PreciseField,
    mode: GuiPreciseMode,
    relative: bool,
}
impl PreciseInputBar {
    /// The ribbon's way in; example: `bar.toggle(ui)` inside the Format panel.
    pub(crate) fn toggle(&mut self, ui: &Ui) {
        if crate::widget_abi::toggle_button(ui, "Precise Input", [108., 18.], self.open) {
            self.open = !self.open;
        }
    }
    /// Draw while open and a command takes points; a command that ends drops what was typed.
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(target) = state.precise_target.as_ref() else {
            self.first = PreciseField::default();
            self.second = PreciseField::default();
            return;
        };
        if !self.open {
            return;
        }
        draw_triad(ui, target);
        let [width, _] = ui.io().display_size;
        ui.window("Precise Input")
            .position([width * 0.5, 200.], imgui::Condition::FirstUseEver)
            .flags(
                WindowFlags::ALWAYS_AUTO_RESIZE
                    | WindowFlags::NO_COLLAPSE
                    | WindowFlags::NO_FOCUS_ON_APPEARING,
            )
            .build(|| {
                self.reference_control(ui);
                ui.same_line();
                self.mode_control(ui);
                ui.same_line();
                self.fields(ui, target, actions);
                triad_controls(ui, actions);
                ui.same_line();
                self.options(ui);
            });
    }
    fn fields(
        &mut self,
        ui: &Ui,
        target: &crate::GuiPreciseTarget,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let pointer = target
            .pointer
            .as_ref()
            .filter(|_| self.mode == GuiPreciseMode::XAndY);
        self.first.follow(pointer.map(|values| &values[0]));
        self.second.follow(pointer.map(|values| &values[1]));
        let [first_label, second_label] = self.mode.field_labels();
        let first = coordinate_field(ui, first_label, &mut self.first);
        ui.same_line();
        let second = coordinate_field(ui, second_label, &mut self.second);
        ui.same_line();
        if (first || second) && valid_number(&self.first.text) && valid_number(&self.second.text) {
            actions.push(GuiShellAction::PrecisePoint(GuiPrecisePoint {
                first: std::mem::take(&mut self.first.text),
                second: std::mem::take(&mut self.second.text),
                mode: self.mode,
                relative: self.relative,
            }));
            self.first.typed = false;
            self.second.typed = false;
        }
    }
    fn reference_control(&mut self, ui: &Ui) {
        if ui.button(if self.relative { "Rel" } else { "Abs" }) {
            ui.open_popup(REFERENCE);
        }
        let Some(_list) = ui.begin_popup(REFERENCE) else {
            return;
        };
        if ui
            .menu_item_config("Relative")
            .selected(self.relative)
            .build()
        {
            self.relative = true;
        }
        if ui
            .menu_item_config("Absolute")
            .selected(!self.relative)
            .build()
        {
            self.relative = false;
        }
    }
    fn mode_control(&mut self, ui: &Ui) {
        if ui.button("Mode") {
            ui.open_popup(MODES);
        }
        let Some(_list) = ui.begin_popup(MODES) else {
            return;
        };
        for mode in GuiPreciseMode::ALL {
            if ui
                .menu_item_config(mode.label())
                .selected(self.mode == mode)
                .build()
            {
                self.mode = mode;
                self.first = PreciseField::default();
                self.second = PreciseField::default();
            }
        }
    }
    fn options(&mut self, ui: &Ui) {
        if ui.button("...##precise-input-options-button") {
            ui.open_popup(OPTIONS);
        }
        let Some(_menu) = ui.begin_popup(OPTIONS) else {
            return;
        };
        // Listed so the toolbar's menu is complete; neither has anything to change here yet.
        ui.menu_item_config("Pin Mini-Toolbar Position")
            .enabled(false)
            .build();
        ui.menu_item_config("Auto Fade").enabled(false).build();
        if ui.menu_item("Close") {
            self.open = false;
        }
    }
}
fn triad_controls(ui: &Ui, actions: &mut Vec<GuiShellAction>) {
    if ui.button("Reposition Triad") {
        actions.push(GuiShellAction::PreciseTriad(GuiTriadCommand::Reposition));
    }
    ui.same_line();
    if ui.button("Reset to Origin") {
        actions.push(GuiShellAction::PreciseTriad(GuiTriadCommand::ResetToOrigin));
    }
}
fn draw_triad(ui: &Ui, target: &GuiPreciseTarget) {
    let Some([origin, x_tip, y_tip]) = target.triad else {
        return;
    };
    let scale = ui.io().display_framebuffer_scale;
    let at = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
    let color = ui.style_color(StyleColor::Text);
    let list = ui.get_background_draw_list();
    for tip in [x_tip, y_tip] {
        list.add_line(at(origin), at(tip), color).build();
    }
}
fn coordinate_field(ui: &Ui, label: &str, field: &mut PreciseField) -> bool {
    let invalid = !field.text.is_empty() && !valid_number(&field.text);
    let _red = invalid.then(|| ui.push_style_color(StyleColor::Text, INVALID_TEXT));
    ui.set_next_item_width(FIELD_WIDTH);
    let accepted = ui
        .input_text(label, &mut field.text)
        .auto_select_all(true)
        .enter_returns_true(true)
        .build();
    field.active = ui.is_item_active();
    field.typed |= ui.is_item_edited();
    accepted
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
    #[test]
    fn a_field_follows_the_pointer_until_a_value_is_typed() {
        let mut field = PreciseField::default();
        field.follow(Some(&"5.000".to_owned()));
        assert_eq!(field.text, "5.000");
        field.typed = true;
        field.follow(Some(&"9.000".to_owned()));
        assert_eq!(field.text, "5.000");
    }
    #[test]
    fn each_mode_names_its_two_values() {
        assert_eq!(GuiPreciseMode::XAndY.field_labels(), ["X", "Y"]);
        assert_eq!(
            GuiPreciseMode::DistanceAndAngle.field_labels(),
            ["Distance", "Angle"]
        );
    }
}
