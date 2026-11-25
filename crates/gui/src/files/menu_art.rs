// SPDX-License-Identifier: GPL-2.0-only
//! Keep menu item navigation native while reserving room for a leading icon.
use crate::icons::{GuiRibbonIcons, IconGlyph, IconSize};
use imgui::{StyleVar, TextureId, Ui};

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
        34. - ui.text_line_height(),
    ]));
    draw();
    paint_glyph(ui, icons, glyph, left, if enabled { 1. } else { 0.4 });
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
