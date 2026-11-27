// SPDX-License-Identifier: GPL-2.0-only

/// Consecutive indices are spaced by the golden ratio in hue, matching the Go mesh palette.
pub(super) fn index_color(index: usize) -> [f32; 4] {
    let hue = (index as f64 * 0.61803398875).fract() * 6.;
    let sector = hue.floor();
    let fraction = hue - sector;
    let saturation = 0.62;
    let value = 0.92;
    let low = value * (1. - saturation);
    let falling = value * (1. - fraction * saturation);
    let rising = value * (1. - (1. - fraction) * saturation);
    let rgb = match sector as u8 {
        0 => [value, rising, low],
        1 => [falling, value, low],
        2 => [low, value, rising],
        3 => [low, falling, value],
        4 => [rising, low, value],
        _ => [value, low, falling],
    };
    let [r, g, b] = rgb.map(|component| component as f32);
    [r, g, b, 1.]
}
