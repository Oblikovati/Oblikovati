// SPDX-License-Identifier: GPL-2.0-only
//! Measure visible radial separators in a captured mini navigation wheel.
use oblikovati_viewport::ViewportCapture;
use std::f32::consts::TAU;

const SAMPLES: usize = 360;
const NEIGHBOR: usize = 10;
// Three RGB channels must differ enough to separate a line from antialiasing.
const MIN_CONTRAST: i32 = 30;
const RADII: [f32; 3] = [0.1875, 0.25, 0.3125];

fn sample(capture: &ViewportCapture, bounds: [f32; 4], angle: usize, radius: f32) -> i32 {
    let center = [(bounds[0] + bounds[2]) / 2., (bounds[1] + bounds[3]) / 2.];
    let angle = TAU * angle as f32 / SAMPLES as f32;
    let x = (center[0] + angle.sin() * (bounds[2] - bounds[0]) * radius).round() as usize;
    let y = (center[1] - angle.cos() * (bounds[3] - bounds[1]) * radius).round() as usize;
    assert!(
        x < capture.extent[0] as usize && y < capture.extent[1] as usize,
        "wheel sample ({x}, {y}): expected coordinates within {:?}",
        capture.extent
    );
    let offset = (y * capture.extent[0] as usize + x) * 4;
    capture.rgba[offset..offset + 3]
        .iter()
        .map(|value| i32::from(*value))
        .sum()
}

fn separator(capture: &ViewportCapture, bounds: [f32; 4], angle: usize) -> bool {
    RADII.into_iter().all(|radius| {
        let middle = sample(capture, bounds, angle, radius);
        let before = sample(
            capture,
            bounds,
            (angle + SAMPLES - NEIGHBOR) % SAMPLES,
            radius,
        );
        let after = sample(capture, bounds, (angle + NEIGHBOR) % SAMPLES, radius);
        middle - before.max(after) > MIN_CONTRAST
    })
}

/// Count separators in the actual pixels; example: compare two captured wheel layouts.
pub(super) fn visible_separators(capture: &ViewportCapture, bounds: [f32; 4]) -> usize {
    let samples: Vec<bool> = (0..SAMPLES)
        .map(|angle| separator(capture, bounds, angle))
        .collect();
    (0..SAMPLES)
        .filter(|&angle| samples[angle] && !samples[(angle + SAMPLES - 1) % SAMPLES])
        .count()
}

#[cfg(test)]
mod tests {
    use super::*;

    fn capture_with_separators(count: usize) -> ViewportCapture {
        let mut capture = ViewportCapture {
            view_id: 1,
            scene_generation: 1,
            camera_generation: 1,
            surface_generation: 1,
            overlay_generation: 1,
            extent: [64, 64],
            rgba: [40, 40, 40, 255].repeat(64 * 64),
        };
        for index in 0..count {
            let angle = TAU * (index as f32 + 0.5) / count as f32;
            for radius in RADII {
                let x = (32. + angle.sin() * 64. * radius).round() as usize;
                let y = (32. - angle.cos() * 64. * radius).round() as usize;
                let offset = (y * 64 + x) * 4;
                capture.rgba[offset..offset + 3].copy_from_slice(&[100, 100, 100]);
            }
        }
        capture
    }

    #[test]
    fn distinguishes_more_separators_from_a_small_circle_without_them() {
        let bounds = [0., 0., 64., 64.];
        assert_eq!(visible_separators(&capture_with_separators(0), bounds), 0);
        assert_eq!(visible_separators(&capture_with_separators(4), bounds), 4);
        assert_eq!(visible_separators(&capture_with_separators(8), bounds), 8);
    }
}
