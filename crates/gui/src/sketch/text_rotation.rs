// SPDX-License-Identifier: GPL-2.0-only
//! Apply common rotations through the same authored angle as typed input.
use imgui::Ui;

const QUARTER_TURNS: [(u16, &str); 4] = [(0, "0°"), (90, "90°"), (180, "180°"), (270, "270°")];

/// Edit the shared angle value, for example by selecting 90 degrees in the list.
pub(super) fn draw(ui: &Ui, degrees: &mut String) -> bool {
    ui.set_next_item_width(145.);
    let typed = ui.input_text("Rotation (deg)", degrees).build();
    ui.same_line();
    ui.set_next_item_width(145.);
    let Some(_choices) = ui.begin_combo("##text-rotation-options", "Choose rotation") else {
        return typed;
    };
    let current = displayed_quarter_turn(degrees);
    for (value, label) in QUARTER_TURNS {
        if ui
            .selectable_config(label)
            .selected(current == Some(value))
            .build()
        {
            *degrees = value.to_string();
            return true;
        }
    }
    typed
}

fn displayed_quarter_turn(degrees: &str) -> Option<u16> {
    let angle = degrees.parse::<f64>().ok()?.rem_euclid(360.);
    QUARTER_TURNS
        .iter()
        .find_map(|(value, _)| (f64::from(*value) == angle).then_some(*value))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn quarter_turn_selection_preserves_equivalent_typed_angles() {
        for (typed, selected) in [("0", 0), ("90.0", 90), ("540", 180), ("-90", 270)] {
            assert_eq!(displayed_quarter_turn(typed), Some(selected));
        }
        for typed in ["37.5", "", "NaN", "inf"] {
            assert_eq!(displayed_quarter_turn(typed), None);
        }
    }
}
