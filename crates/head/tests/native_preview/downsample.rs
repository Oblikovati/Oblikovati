// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn capture(
    extent: [u32; 2],
    pixel: impl Fn(u32, u32) -> [u8; 4],
) -> oblikovati_viewport::ViewportCapture {
    let mut rgba = Vec::with_capacity((extent[0] * extent[1] * 4) as usize);
    for y in 0..extent[1] {
        for x in 0..extent[0] {
            rgba.extend_from_slice(&pixel(x, y));
        }
    }
    oblikovati_viewport::ViewportCapture {
        view_id: 0,
        scene_generation: 0,
        camera_generation: 0,
        surface_generation: 0,
        overlay_generation: 0,
        extent,
        rgba,
    }
}
#[test]
fn a_preview_has_a_fixed_size_whatever_the_window() {
    for extent in [[64, 36], [1440, 900], [100, 2000]] {
        let image = downsampled_preview(&capture(extent, |_, _| [1, 2, 3, 255]));
        assert_eq!(image.extent, PREVIEW_CELLS);
        assert_eq!(
            image.rgb.len(),
            (PREVIEW_CELLS[0] * PREVIEW_CELLS[1]) as usize
        );
        assert!(image.rgb.iter().all(|cell| *cell == [1, 2, 3]));
    }
}
#[test]
fn cells_read_from_their_own_part_of_the_window() {
    let image = downsampled_preview(&capture([640, 360], |x, y| {
        [(x / 10) as u8, (y / 10) as u8, 0, 255]
    }));
    assert_eq!(image.rgb[0], [0, 0, 0]);
    assert_eq!(image.rgb[63], [63, 0, 0]);
    assert_eq!(image.rgb[64 * 35], [0, 35, 0]);
}
