// SPDX-License-Identifier: GPL-2.0-only
//! Preview-only diffuse lighting for the renderer's unlit triangle presentation.
use oblikovati_viewport::{ViewportScene, ViewportTriangle};

const LIGHT: [f64; 3] = [0.4242640687, 0.5656854249, std::f64::consts::FRAC_1_SQRT_2];
const AMBIENT: f64 = 0.45;

pub(super) fn shaded_scene(scene: &ViewportScene) -> ViewportScene {
    let mut preview = scene.clone();
    for triangle in &mut preview.triangles {
        shade_triangle(triangle);
    }
    preview
}
fn shade_triangle(triangle: &mut ViewportTriangle) {
    let [a, b, c] = triangle.positions;
    let u = std::array::from_fn::<_, 3, _>(|axis| f64::from(b[axis]) - f64::from(a[axis]));
    let v = std::array::from_fn::<_, 3, _>(|axis| f64::from(c[axis]) - f64::from(a[axis]));
    let normal = [
        u[1] * v[2] - u[2] * v[1],
        u[2] * v[0] - u[0] * v[2],
        u[0] * v[1] - u[1] * v[0],
    ];
    let length = normal.iter().map(|part| part * part).sum::<f64>().sqrt();
    let diffuse = if length > 0. {
        normal
            .iter()
            .zip(LIGHT)
            .map(|(n, light)| n * light)
            .sum::<f64>()
            / length
    } else {
        0.
    };
    let factor = AMBIENT + (1. - AMBIENT) * diffuse.clamp(0., 1.);
    for channel in &mut triangle.color[..3] {
        *channel *= factor as f32;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn preview_lighting_distinguishes_faces_without_mutating_the_scene() {
        let triangle = |positions| ViewportTriangle {
            positions,
            color: [0.8, 0.6, 0.4, 0.5],
        };
        let scene = ViewportScene {
            background: [0.1, 0.2, 0.3],
            triangles: vec![
                triangle([[0., 0., 0.], [1., 0., 0.], [0., 1., 0.]]),
                triangle([[0., 0., 0.], [0., 1., 0.], [0., 0., 1.]]),
            ],
        };
        let preview = shaded_scene(&scene);
        assert_ne!(preview.triangles[0].color, preview.triangles[1].color);
        assert_eq!(preview.triangles[0].color[3], 0.5);
        assert_eq!(scene.triangles[0].color, [0.8, 0.6, 0.4, 0.5]);
        assert_eq!(preview.background, scene.background);
    }
    #[test]
    fn preview_lighting_stays_finite_for_extreme_and_degenerate_triangles() {
        for extent in [f32::MIN_POSITIVE, 1., f32::MAX] {
            for positions in [
                [[0., 0., 0.], [extent, 0., 0.], [0., extent, 0.]],
                [[0., 0., 0.]; 3],
            ] {
                let mut triangle = ViewportTriangle {
                    positions,
                    color: [1.; 4],
                };
                shade_triangle(&mut triangle);
                assert!(
                    triangle
                        .color
                        .iter()
                        .all(|value| value.is_finite() && (0.0..=1.0).contains(value))
                );
            }
        }
    }
}
