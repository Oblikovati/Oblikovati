// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiExtrudeDirection;
use imgui::Ui;

pub(super) fn draw(ui: &Ui, direction: &mut GuiExtrudeDirection, read_only: bool) -> bool {
    use GuiExtrudeDirection::{Negative, Positive, Symmetric};
    let previous = *direction;
    ui.text("Direction");
    let _disabled = ui.begin_disabled(read_only);
    for (choice, label) in [
        (Positive, "Default"),
        (Negative, "Flipped"),
        (Symmetric, "Symmetric"),
    ] {
        ui.same_line();
        if ui.radio_button_bool(label, *direction == choice) {
            *direction = choice;
        }
    }
    *direction != previous
}
