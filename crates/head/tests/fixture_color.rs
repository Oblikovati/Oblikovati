// SPDX-License-Identifier: GPL-2.0-only
//! Colors for native fixture assertions come from the asset or the explicit fixture.
use oblikovati_viewport::ViewportCapture;

/// Resolve the shipped neutral appearance; example: `default_fixture_rgb()`.
pub(crate) fn default_fixture_rgb() -> [u8; 3] {
    use oblikovati_api::contract::Appearance;
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let color = catalog
        .appearances()
        .get(oblikovati_model::assets::DEFAULT_APPEARANCE_ID)
        .unwrap()
        .base()
        .color;
    framebuffer_rgb([color.r, color.g, color.b])
}
/// Convert linear RGB to framebuffer sRGB; example: `framebuffer_rgb([0., 0., 0.])`.
pub(crate) fn framebuffer_rgb(color: [f32; 3]) -> [u8; 3] {
    color.map(|value| {
        let srgb = if value <= 0.0031308 {
            value * 12.92
        } else {
            1.055 * value.powf(1. / 2.4) - 0.055
        };
        (srgb * 255.).round() as u8
    })
}
/// Count opaque fixture pixels; example: `count_fixture_color(&capture, color)`.
pub(crate) fn count_fixture_color(capture: &ViewportCapture, color: [u8; 3]) -> usize {
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| pixel[..3] == color && pixel[3] == 255)
        .count()
}
#[test]
fn fixture_conversion_matches_srgb_reference_values() {
    assert_eq!(framebuffer_rgb([0., 0.0031308, 1.]), [0, 10, 255]);
    assert_ne!(default_fixture_rgb(), [0, 255, 0]);
}

#[test]
fn pixel_counter_requires_the_exact_color_and_full_opacity() {
    let capture = ViewportCapture {
        view_id: 1,
        scene_generation: 1,
        camera_generation: 1,
        surface_generation: 1,
        overlay_generation: 1,
        extent: [3, 1],
        rgba: vec![1, 2, 3, 255, 1, 2, 3, 0, 0, 255, 0, 255],
    };
    assert_eq!(count_fixture_color(&capture, [1, 2, 3]), 1);
    assert_eq!(count_fixture_color(&capture, [0, 255, 0]), 1);
    assert_eq!(count_fixture_color(&capture, [4, 5, 6]), 0);
}
