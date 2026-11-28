// SPDX-License-Identifier: GPL-2.0-only
//! Analytic finish face area integration and appearance color mapping (#3918).
use super::super::solid::NamedSolid;
use oblikovati_model::{history::DocumentEditHistory, identity::DocumentGuid};
use std::collections::BTreeSet;

/// Internal model length units are centimeters (`1 cm² = 100 mm²`).
const CM2_TO_MM2: f64 = 100.0;

/// Map a Finish appearance name to linear display RGBA color for viewport rendering.
/// Example: `finish_appearance_rgba("Red")` returns bright crimson RGBA.
pub fn finish_appearance_rgba(appearance: &str) -> [f32; 4] {
    match appearance.trim().to_ascii_lowercase().as_str() {
        "red" | "ral 3018" | "prime/paint ral 3018" => [0.85, 0.14, 0.14, 1.0],
        "turquoise" => [0.14, 0.78, 0.76, 1.0],
        "teal" | "anodized - teal" => [0.12, 0.68, 0.66, 1.0],
        "blue" | "ral 5010" => [0.16, 0.38, 0.82, 1.0],
        "green" | "ral 6018" => [0.18, 0.68, 0.28, 1.0],
        "yellow" | "ral 1023" => [0.92, 0.76, 0.14, 1.0],
        "black" | "black oxide" => [0.16, 0.17, 0.20, 1.0],
        "chrome" | "chrome - polished" | "silver" => [0.82, 0.85, 0.88, 1.0],
        "gold" | "brass" => [0.84, 0.68, 0.24, 1.0],
        "barrel finished" | "satin" | "natural" => [0.70, 0.73, 0.77, 1.0],
        _ => [0.58, 0.64, 0.72, 1.0],
    }
}

/// Map a Finish appearance name to 8-bit RGB channels for live preview face tinting.
/// Example: `finish_appearance_rgb_u8("Turquoise")`.
pub fn finish_appearance_rgb_u8(appearance: &str) -> [u8; 3] {
    let [r, g, b, _] = finish_appearance_rgba(appearance);
    [
        (r.clamp(0.0, 1.0) * 255.0).round() as u8,
        (g.clamp(0.0, 1.0) * 255.0).round() as u8,
        (b.clamp(0.0, 1.0) * 255.0).round() as u8,
    ]
}

/// Compute exact analytic total surface area in `mm²` for `included \ excluded` faces using `BRepGProp`.
/// Example: `compute_finish_area_mm2(&solid, &history, doc, &included, &excluded)?`.
pub fn compute_finish_area_mm2(
    solid: &NamedSolid,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    included: &[Vec<u8>],
    excluded: &[Vec<u8>],
) -> Result<f64, String> {
    if included.is_empty() {
        return Ok(0.0);
    }
    let excluded_set: BTreeSet<&[u8]> = excluded.iter().map(Vec::as_slice).collect();
    let mut seen = BTreeSet::new();
    let mut area_cm2 = 0.0_f64;
    for face in included {
        let key = face.as_slice();
        if excluded_set.contains(key) || !seen.insert(key) {
            continue;
        }
        let face_cm2 = solid.face_area(history, document, key)?;
        if !face_cm2.is_finite() || face_cm2 <= 0.0 {
            return Err(format!(
                "finish face {key:?}, area {face_cm2}: expected finite positive area"
            ));
        }
        area_cm2 += face_cm2;
    }
    Ok(area_cm2 * CM2_TO_MM2)
}
