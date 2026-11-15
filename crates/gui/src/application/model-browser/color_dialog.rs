// SPDX-License-Identifier: GPL-2.0-only
//! The color dialog behind Select in a line color list: swatches, slots and defined colors.
use imgui::{ColorStackToken, StyleColor, Ui};

pub(super) const TITLE: &str = "Color";
const BASIC_SIZE: [f32; 2] = [300., 360.];
const DEFINING_WIDTH: f32 = 640.;
const CHANNEL_AND_ACTION_ROWS: f32 = 8.;
const SWATCH: [f32; 2] = [22., 20.];
const SWATCHES_PER_ROW: usize = 8;
const CUSTOM_SLOTS: usize = 16;
const FIELD_SIDE: f32 = 180.;
const FIELD_CELLS: usize = 24;
const BAR_WIDTH: f32 = 16.;
const HUE_LIMIT: i32 = 359;
const SHARE_LIMIT: i32 = 100;
const CHANNEL_LIMIT: i32 = 255;
const FIELD_WIDTH: f32 = 70.;
const BASIC_COLORS: [[u8; 3]; 24] = [
    [255, 128, 128],
    [255, 255, 128],
    [128, 255, 128],
    [0, 255, 128],
    [128, 255, 255],
    [0, 128, 255],
    [255, 128, 192],
    [255, 128, 255],
    [255, 0, 0],
    [255, 255, 0],
    [128, 255, 0],
    [0, 255, 64],
    [0, 255, 255],
    [0, 128, 192],
    [128, 128, 192],
    [255, 0, 255],
    [128, 64, 64],
    [255, 128, 64],
    [0, 255, 0],
    [0, 128, 128],
    [0, 64, 128],
    [128, 128, 255],
    [128, 0, 64],
    [255, 0, 128],
];

/// Hue in degrees; saturation and luminance as whole percentages.
pub(crate) fn hsl_of_rgb([red, green, blue]: [u8; 3]) -> [i32; 3] {
    let [r, g, b] = [red, green, blue].map(|channel| f32::from(channel) / 255.);
    let high = r.max(g).max(b);
    let low = r.min(g).min(b);
    let spread = high - low;
    let luminance = (high + low) / 2.;
    if spread == 0. {
        return [0, 0, (luminance * 100.).round() as i32];
    }
    let saturation = spread / (1. - (2. * luminance - 1.).abs());
    let sector = if high == r {
        ((g - b) / spread).rem_euclid(6.)
    } else if high == g {
        (b - r) / spread + 2.
    } else {
        (r - g) / spread + 4.
    };
    [
        (sector * 60.).round() as i32 % 360,
        (saturation * 100.).round() as i32,
        (luminance * 100.).round() as i32,
    ]
}
pub(crate) fn rgb_of_hsl([hue, saturation, luminance]: [i32; 3]) -> [u8; 3] {
    let share = saturation as f32 / 100.;
    let level = luminance as f32 / 100.;
    let chroma = (1. - (2. * level - 1.).abs()) * share;
    let sector = hue.rem_euclid(360) as f32 / 60.;
    let second = chroma * (1. - (sector.rem_euclid(2.) - 1.).abs());
    let [r, g, b] = match sector as i32 {
        0 => [chroma, second, 0.],
        1 => [second, chroma, 0.],
        2 => [0., chroma, second],
        3 => [0., second, chroma],
        4 => [second, 0., chroma],
        _ => [chroma, 0., second],
    };
    let lift = level - chroma / 2.;
    [r, g, b].map(|channel| ((channel + lift) * 255.).round() as u8)
}
fn as_floats([red, green, blue]: [u8; 3]) -> [f32; 4] {
    [red, green, blue]
        .map(|channel| f32::from(channel) / 255.)
        .into_iter()
        .chain([1.])
        .collect::<Vec<f32>>()
        .try_into()
        .unwrap_or([0., 0., 0., 1.])
}
/// What a press on OK or Cancel means; while neither is pressed the dialog stays up.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum ColorAnswer {
    Chosen([u8; 3]),
    Cancelled,
}
#[derive(Debug)]
pub(super) struct ColorDialog {
    color: [u8; 3],
    custom: [[u8; 3]; CUSTOM_SLOTS],
    next_slot: usize,
    defining: bool,
}
impl Default for ColorDialog {
    fn default() -> Self {
        Self {
            color: BASIC_COLORS[0],
            custom: [[255, 255, 255]; CUSTOM_SLOTS],
            next_slot: 0,
            defining: false,
        }
    }
}
impl ColorDialog {
    /// Start over without defining; the custom slots stay for the session.
    pub(super) fn begin(&mut self) {
        self.defining = false;
    }
    /// A frame's worth of the dialog, once its popup has been asked for.
    pub(super) fn panel(&mut self, ui: &Ui) -> Option<ColorAnswer> {
        let size = self.fitted_size(ui);
        // Defining grows the dialog while it is open, so its size cannot wait for its appearing.
        let [width, height] = ui.io().display_size;
        let centered = [(width - size[0]) / 2., (height - size[1]) / 2.];
        crate::widget_abi::window_geometry(ui, centered, size, imgui::Condition::Always);
        let _dialog = ui.begin_modal_popup(TITLE)?;
        self.contents(ui);
        let answer = answer_buttons(ui, self.color);
        if answer.is_some() {
            ui.close_current_popup();
        }
        answer
    }
    fn fitted_size(&self, ui: &Ui) -> [f32; 2] {
        let style = ui.clone_style();
        // Six channel rows, Add, and the answer row must fit below the color field.
        let height = FIELD_SIDE
            + CHANNEL_AND_ACTION_ROWS * ui.frame_height_with_spacing()
            + ui.frame_height()
            + 2. * (style.window_padding[1] + style.item_spacing[1]);
        let wanted = if self.defining {
            [DEFINING_WIDTH, height]
        } else {
            BASIC_SIZE
        };
        std::array::from_fn(|axis| wanted[axis].min((ui.io().display_size[axis] - 24.).max(1.)))
    }
    fn contents(&mut self, ui: &Ui) {
        // On a short display only the palette scrolls; cancellation stays reachable.
        let height = (ui.content_region_avail()[1] - ui.frame_height_with_spacing()).max(1.);
        let region = crate::widget_abi::scroll_region(ui, "##color-palette", [0., height]);
        if !region.visible {
            return;
        }
        ui.group(|| self.swatches(ui));
        if self.defining {
            ui.same_line();
            ui.group(|| self.definition(ui));
        }
    }
    fn swatches(&mut self, ui: &Ui) {
        ui.text("Basic colors");
        let basic = BASIC_COLORS;
        if let Some(chosen) = swatch_grid(ui, "basic", &basic) {
            self.color = chosen;
        }
        ui.text("Custom colors");
        let custom = self.custom;
        if let Some(chosen) = swatch_grid(ui, "custom", &custom) {
            self.color = chosen;
        }
        if !self.defining && crate::widget_abi::button(ui, "Define Custom Colors", [0.; 2]) {
            self.defining = true;
        }
    }
    fn definition(&mut self, ui: &Ui) {
        self.color_field(ui);
        let mut hsl = hsl_of_rgb(self.color);
        let mut rgb = self.color.map(i32::from);
        let mut hsl_changed = number_row(ui, "Hue", &mut hsl[0], HUE_LIMIT);
        hsl_changed |= number_row(ui, "Saturation", &mut hsl[1], SHARE_LIMIT);
        hsl_changed |= number_row(ui, "Luminance", &mut hsl[2], SHARE_LIMIT);
        let mut rgb_changed = number_row(ui, "Red", &mut rgb[0], CHANNEL_LIMIT);
        rgb_changed |= number_row(ui, "Green", &mut rgb[1], CHANNEL_LIMIT);
        rgb_changed |= number_row(ui, "Blue", &mut rgb[2], CHANNEL_LIMIT);
        if hsl_changed {
            self.color = rgb_of_hsl(hsl);
        }
        if rgb_changed {
            self.color = rgb.map(|channel| channel as u8);
        }
        if crate::widget_abi::button(ui, "Add to Custom Colors", [0.; 2]) {
            self.custom[self.next_slot] = self.color;
            self.next_slot = (self.next_slot + 1) % CUSTOM_SLOTS;
        }
    }
    /// A hue and saturation field, a brightness bar beside it and the color so far.
    fn color_field(&mut self, ui: &Ui) {
        let origin = ui.cursor_screen_pos();
        let [_, _, luminance] = hsl_of_rgb(self.color);
        ui.invisible_button("##color-field", [FIELD_SIDE, FIELD_SIDE]);
        let list = ui.get_window_draw_list();
        let cell = FIELD_SIDE / FIELD_CELLS as f32;
        for row in 0..FIELD_CELLS {
            for column in 0..FIELD_CELLS {
                let hue = (column * 360 / FIELD_CELLS) as i32;
                let saturation = 100 - (row * 100 / FIELD_CELLS) as i32;
                let color = as_floats(rgb_of_hsl([hue, saturation, 50]));
                let at = [
                    origin[0] + column as f32 * cell,
                    origin[1] + row as f32 * cell,
                ];
                list.add_rect(at, [at[0] + cell, at[1] + cell], color)
                    .filled(true)
                    .build();
            }
        }
        if ui.is_item_active() {
            let [x, y] = ui.io().mouse_pos;
            let hue = ((x - origin[0]) / FIELD_SIDE * 360.).clamp(0., 359.) as i32;
            let share = (100. - (y - origin[1]) / FIELD_SIDE * 100.).clamp(0., 100.) as i32;
            self.color = rgb_of_hsl([hue, share, luminance]);
        }
        drop(list);
        ui.same_line();
        self.brightness_bar(ui);
    }
    fn brightness_bar(&mut self, ui: &Ui) {
        let origin = ui.cursor_screen_pos();
        let list = ui.get_window_draw_list();
        let [hue, saturation, _] = hsl_of_rgb(self.color);
        ui.invisible_button("##color-brightness", [BAR_WIDTH, FIELD_SIDE]);
        let cell = FIELD_SIDE / FIELD_CELLS as f32;
        for row in 0..FIELD_CELLS {
            let level = 100 - (row * 100 / FIELD_CELLS) as i32;
            let color = as_floats(rgb_of_hsl([hue, saturation, level]));
            let top = origin[1] + row as f32 * cell;
            list.add_rect([origin[0], top], [origin[0] + BAR_WIDTH, top + cell], color)
                .filled(true)
                .build();
        }
        if ui.is_item_active() {
            let level = 100. - (ui.io().mouse_pos[1] - origin[1]) / FIELD_SIDE * 100.;
            self.color = rgb_of_hsl([hue, saturation, level.clamp(0., 100.) as i32]);
        }
        ui.same_line();
        let preview = ui.cursor_screen_pos();
        ui.dummy([FIELD_WIDTH, FIELD_SIDE]);
        list.add_rect(
            preview,
            [preview[0] + FIELD_WIDTH, preview[1] + FIELD_SIDE / 2.],
            as_floats(self.color),
        )
        .filled(true)
        .build();
        list.add_text(
            [preview[0], preview[1] + FIELD_SIDE / 2.],
            [1., 1., 1., 1.],
            "Preview",
        );
    }
}
fn number_row(ui: &Ui, label: &str, value: &mut i32, limit: i32) -> bool {
    ui.set_next_item_width(FIELD_WIDTH);
    let changed = ui.input_int(label, value).step(0).build();
    *value = (*value).clamp(0, limit);
    changed
}
fn swatch_grid(ui: &Ui, group: &str, colors: &[[u8; 3]]) -> Option<[u8; 3]> {
    let mut chosen = None;
    for (index, color) in colors.iter().enumerate() {
        let _tinted: [ColorStackToken<'_>; 2] = [
            ui.push_style_color(StyleColor::Button, as_floats(*color)),
            ui.push_style_color(StyleColor::ButtonHovered, as_floats(*color)),
        ];
        if crate::widget_abi::button(ui, &format!("##{group}-{index}"), SWATCH) {
            chosen = Some(*color);
        }
        if (index + 1) % SWATCHES_PER_ROW != 0 {
            ui.same_line();
        }
    }
    ui.new_line();
    chosen
}
fn answer_buttons(ui: &Ui, color: [u8; 3]) -> Option<ColorAnswer> {
    let chosen = crate::widget_abi::button(ui, "OK##color", [80., 0.]);
    ui.same_line();
    if crate::widget_abi::button(ui, "Cancel##color", [80., 0.]) {
        return Some(ColorAnswer::Cancelled);
    }
    chosen.then_some(ColorAnswer::Chosen(color))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn expanded_palette_keeps_the_answer_row_visible_at_both_pixel_densities() {
        for density in [1., 2.] {
            let mut context = crate::test_context::create();
            context.set_ini_filename(None);
            context.fonts().build_rgba32_texture();
            context.io_mut().display_size = [1440. / density, 900. / density];
            context.io_mut().display_framebuffer_scale = [density; 2];
            context.io_mut().delta_time = 1. / 60.;
            check_expanded_palette_frames(&mut context);
        }
    }
    fn check_expanded_palette_frames(context: &mut imgui::Context) {
        let mut dialog = ColorDialog {
            defining: true,
            ..Default::default()
        };
        for _ in 0..2 {
            let ui = context.frame();
            let size = dialog.fitted_size(ui);
            assert!(size[1] <= ui.io().display_size[1]);
            crate::widget_abi::window_geometry(ui, [0.; 2], size, imgui::Condition::Always);
            ui.window("Color layout test").build(|| {
                dialog.contents(ui);
                assert!(answer_buttons(ui, dialog.color).is_none());
                assert!(
                    ui.is_item_visible(),
                    "expected Cancel inside the fitted dialog"
                );
            });
            context.render();
        }
    }
    #[test]
    fn hsl_and_rgb_describe_the_same_color() {
        assert_eq!(hsl_of_rgb([255, 0, 0]), [0, 100, 50]);
        assert_eq!(hsl_of_rgb([0, 0, 255]), [240, 100, 50]);
        assert_eq!(hsl_of_rgb([128, 128, 128]), [0, 0, 50]);
        for color in [
            [255, 0, 0],
            [0, 255, 0],
            [0, 0, 255],
            [255, 255, 0],
            [255, 255, 255],
        ] {
            assert_eq!(rgb_of_hsl(hsl_of_rgb(color)), color);
        }
    }
    #[test]
    fn a_new_dialog_is_basic_and_keeps_its_slots_when_it_starts_over() {
        let mut dialog = ColorDialog::default();
        assert!(!dialog.defining);
        dialog.custom[0] = [1, 2, 3];
        dialog.defining = true;
        dialog.begin();
        assert!(!dialog.defining);
        assert_eq!(dialog.custom[0], [1, 2, 3]);
    }
}
