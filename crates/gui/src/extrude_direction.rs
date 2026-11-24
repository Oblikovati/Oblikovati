// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiExtrudeDirection;
use imgui::Ui;

pub(super) fn draw(ui: &Ui, direction: &mut GuiExtrudeDirection, read_only: bool) -> bool {
    use GuiExtrudeDirection::{Asymmetric, Negative, Positive, Symmetric};
    let previous = *direction;
    ui.text("Direction");
    let _disabled = ui.begin_disabled(read_only);
    for (choice, label) in [
        (Positive, "Default"),
        (Negative, "Flipped"),
        (Symmetric, "Symmetric"),
        (Asymmetric, "Asymmetric"),
    ] {
        if choice != Symmetric {
            ui.same_line();
        }
        if ui.radio_button_bool(label, *direction == choice) {
            *direction = choice;
        }
    }
    *direction != previous
}

pub(super) fn draw_second_distance(ui: &Ui, first: &mut String, second: &mut String) -> bool {
    let style = ui.clone_style();
    let label = "Distance B";
    let flip = "Flip A/B";
    let reserved = ui.calc_text_size(label)[0]
        + ui.calc_text_size(flip)[0]
        + style.item_inner_spacing[0]
        + style.item_spacing[0]
        + 2. * style.frame_padding[0];
    ui.set_next_item_width((ui.content_region_avail()[0] - reserved).max(1.));
    let edited = ui.input_text(label, second).build();
    ui.same_line();
    let flipped = crate::widget_abi::button(ui, flip, [0.; 2]);
    if flipped {
        std::mem::swap(first, second);
    }
    let _disabled = ui.begin_disabled(true);
    ui.input_text("Taper B", &mut String::from("0 deg")).build();
    edited || flipped
}
