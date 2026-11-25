// SPDX-License-Identifier: GPL-2.0-only
//! Keep menu item navigation native while reserving room for a leading icon.
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use imgui::{StyleColor, StyleVar, TextureId, Ui};

const COMMAND_HEIGHT: f32 = 32.;

pub(super) fn command(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    glyph: IconGlyph,
    enabled: bool,
    draw: impl FnOnce(),
) {
    let left = ui.cursor_screen_pos()[0] + 6.;
    let spacing = ui.clone_style().item_spacing[0];
    let _spacing = ui.push_style_var(StyleVar::ItemSpacing([
        spacing,
        COMMAND_HEIGHT - ui.text_line_height(),
    ]));
    draw();
    paint_glyph(ui, icons, glyph, left, if enabled { 1. } else { 0.4 });
}

pub(super) fn current_value_button(ui: &Ui, label: &str) -> bool {
    let clicked = ui.button(label);
    crate::theme::active_tab_underline(ui);
    clicked
}

pub(super) fn shadow(ui: &Ui) {
    let [left, top] = ui.window_pos();
    let [width, height] = ui.window_size();
    let [right, bottom] = [left + width, top + height];
    let draw = ui.get_window_draw_list();
    let (ink, layers) = if ui.style_color(StyleColor::WindowBg)[0] > 0.5 {
        ([17. / 255., 19. / 255., 24. / 255.], [0.012, 0.025, 0.045])
    } else {
        ([0.; 3], [0.08, 0.14, 0.30])
    };
    let [r, g, b] = ink;
    for (radius, alpha) in [12., 8., 4.].into_iter().zip(layers) {
        let start = [left - radius, top];
        let end = [right + radius, bottom + radius + 4.];
        for (min, max) in [
            (start, [left, bottom]),
            ([right, top], [end[0], bottom]),
            ([start[0], bottom], end),
        ] {
            // Only draw outside the popup so its opaque surface and border stay intact.
            draw.with_clip_rect(min, max, || {
                draw.add_rect([start[0], top - radius + 4.], end, [r, g, b, alpha])
                    .rounding(radius + 3.)
                    .filled(true)
                    .build();
            });
        }
    }
}

pub(super) fn label(label: &str) -> String {
    format!("        {label}")
}

pub(super) fn paint_glyph(
    ui: &Ui,
    icons: &GuiRibbonIcons,
    glyph: IconGlyph,
    left: f32,
    alpha: f32,
) {
    let texture = TextureId::new(icons.key(glyph, IconSize::Small).raw());
    let center = (ui.item_rect_min()[1] + ui.item_rect_max()[1]) / 2.;
    ui.get_window_draw_list()
        .add_image(texture, [left, center - 8.], [left + 16., center + 8.])
        .col([1., 1., 1., alpha])
        .build();
}
