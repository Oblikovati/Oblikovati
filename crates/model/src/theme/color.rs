// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::Rgba;
pub(super) fn parse_hex(value: &str) -> Result<Rgba, String> {
    let digits = value.strip_prefix('#').unwrap_or("");
    if !matches!(digits.len(), 6 | 8) || !digits.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        return Err(format!(
            "theme color {value:?}: expected #RRGGBB or #RRGGBBAA"
        ));
    }
    let mut channels = [255u8; 4];
    for (channel, pair) in channels
        .iter_mut()
        .zip(digits.as_bytes().as_chunks::<2>().0)
    {
        *channel = u8::from_str_radix(std::str::from_utf8(pair).expect("validated ASCII"), 16)
            .expect("validated hex");
    }
    Ok(Rgba {
        r: channels[0] as f32 / 255.,
        g: channels[1] as f32 / 255.,
        b: channels[2] as f32 / 255.,
        a: channels[3] as f32 / 255.,
    })
}
pub(super) fn format_hex(color: &Rgba) -> String {
    let bytes = [color.r, color.g, color.b, color.a]
        .map(|channel| (channel.clamp(0., 1.) * 255. + 0.5) as u8);
    format!(
        "#{:02x}{:02x}{:02x}{:02x}",
        bytes[0], bytes[1], bytes[2], bytes[3]
    )
}
pub(super) fn validate_color(color: &Rgba) -> Result<(), String> {
    if [color.r, color.g, color.b, color.a]
        .iter()
        .all(|value| value.is_finite() && (0. ..=1.).contains(value))
    {
        return Ok(());
    }
    Err(format!(
        "theme color {color:?}: expected four finite channels in 0..=1"
    ))
}
pub(super) fn mix_rgb(base: &Rgba, text: &Rgba, amount: f32) -> Rgba {
    let mix = |left: f32, right: f32| left + (right - left) * amount;
    Rgba {
        r: mix(base.r, text.r),
        g: mix(base.g, text.g),
        b: mix(base.b, text.b),
        a: base.a,
    }
}
pub(super) fn sketch_green(background: &Rgba) -> Rgba {
    if relative_luminance(background) > 0.2 {
        return Rgba {
            r: 0.02,
            g: 0.36,
            b: 0.13,
            a: 1.,
        };
    }
    Rgba {
        r: 0.24,
        g: 0.85,
        b: 0.40,
        a: 1.,
    }
}
fn relative_luminance(color: &Rgba) -> f64 {
    let linear = |value: f32| {
        let value = value as f64;
        if value <= 0.03928 {
            value / 12.92
        } else {
            ((value + 0.055) / 1.055).powf(2.4)
        }
    };
    0.2126 * linear(color.r) + 0.7152 * linear(color.g) + 0.0722 * linear(color.b)
}
