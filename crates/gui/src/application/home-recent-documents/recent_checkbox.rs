// SPDX-License-Identifier: GPL-2.0-only
//! Keep a standard keyboard and pointer checkbox with a fixed visual footprint.
use imgui::{StyleColor, StyleVar, Ui};

const HALF_BOX: f32 = 7.;

pub(super) fn checkbox(ui: &Ui, label: &str, checked: &mut bool, partial: bool) -> bool {
    let changed = {
        let _border = ui.push_style_var(StyleVar::FrameBorderSize(0.));
        let _hidden = [
            StyleColor::FrameBg,
            StyleColor::FrameBgHovered,
            StyleColor::FrameBgActive,
            StyleColor::CheckMark,
        ]
        .map(|color| ui.push_style_color(color, [0.; 4]));
        ui.checkbox(label, checked)
    };
    if ui.is_item_visible() {
        paint(ui, *checked, partial && !changed);
    }
    changed
}

fn paint(ui: &Ui, checked: bool, partial: bool) {
    let [min, max] = outline(ui.item_rect_min(), ui.item_rect_max());
    let [x, y] = min.map(|edge| edge + HALF_BOX);
    let background = if ui.is_item_hovered() {
        StyleColor::FrameBgHovered
    } else {
        StyleColor::FrameBg
    };
    let draw = ui.get_window_draw_list();
    draw.add_rect(min, max, ui.style_color(background))
        .rounding(3.)
        .filled(true)
        .build();
    draw.add_rect(min, max, ui.style_color(StyleColor::Border))
        .rounding(3.)
        .build();
    let mark = ui.style_color(StyleColor::CheckMark);
    if partial {
        draw.add_line([x - 3., y], [x + 3., y], mark)
            .thickness(2.)
            .build();
    } else if checked {
        draw.add_polyline(
            vec![[x - 3.5, y], [x - 1., y + 2.5], [x + 4., y - 3.]],
            mark,
        )
        .thickness(1.8)
        .build();
    }
}

fn outline(low: [f32; 2], high: [f32; 2]) -> [[f32; 2]; 2] {
    let center: [f32; 2] = std::array::from_fn(|axis| ((low[axis] + high[axis]) * 0.5).floor());
    [
        center.map(|edge| edge - HALF_BOX),
        center.map(|edge| edge + HALF_BOX),
    ]
}

#[cfg(test)]
#[test]
fn the_studio_outline_stays_centered_as_the_native_checkbox_grows() {
    assert_eq!(outline([10., 20.], [28., 38.]), [[12., 22.], [26., 36.]]);
    assert_eq!(outline([10., 20.], [46., 56.]), [[21., 31.], [35., 45.]]);
}
