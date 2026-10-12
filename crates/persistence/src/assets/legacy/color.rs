// SPDX-License-Identifier: GPL-2.0-only
use super::gamma::GAMMA_BITS;
use oblikovati_api::types::Color3;
pub(super) fn linear_hex_color(hex: &str) -> Result<Color3, String> {
    let body = hex.strip_prefix('#').unwrap_or_default();
    if !matches!(body.len(), 6 | 8) || !body.bytes().all(|value| value.is_ascii_hexdigit()) {
        return Err(format!(
            "legacy color {hex:?}: expected #RRGGBB or #RRGGBBAA"
        ));
    }
    let channels = (0..3)
        .map(|axis| {
            u8::from_str_radix(&body[axis * 2..axis * 2 + 2], 16)
                .map(|channel| f32::from_bits(GAMMA_BITS[channel as usize]))
        })
        .collect::<Result<Vec<_>, _>>()
        .map_err(|cause| {
            format!("legacy color {hex:?}: expected hexadecimal byte channels: {cause}")
        })?;
    Ok(Color3 {
        r: channels[0],
        g: channels[1],
        b: channels[2],
    })
}
pub(super) fn emission_split(color: Color3) -> (f32, Color3) {
    let luminance = color.r.max(color.g).max(color.b);
    if luminance <= 0. {
        return (0., Color3::default());
    }
    (
        luminance,
        Color3 {
            r: color.r / luminance,
            g: color.g / luminance,
            b: color.b / luminance,
        },
    )
}
