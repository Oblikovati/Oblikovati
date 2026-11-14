// SPDX-License-Identifier: GPL-2.0-only
//! Preserve Go's analytic extrusion-face frame convention during migration.
use super::WorkPlaneError;
use crate::sketch::SketchPlane;

pub(super) fn go_prism_frame(native: SketchPlane) -> Result<SketchPlane, WorkPlaneError> {
    let normal = native.normal();
    // Go geom.axisFrame chooses this seed at aa8297d1 (#3882).
    let seed = if normal[0].abs() > 0.9 {
        [0., 1., 0.]
    } else {
        [1., 0., 0.]
    };
    let x_axis = cross(normal, seed);
    let y_axis = cross(normal, x_axis);
    SketchPlane::from_directions(native.recipe().origin, x_axis, y_axis, native.options())
        .map_err(WorkPlaneError::Frame)
}

fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn go_cube_face_axes_match_the_recorded_analytic_oracle() {
        // Go witness: architecture/rust-port/receipts/2026-11-14-face-frame/go-final.log.gz.
        let cases = [
            ([1., 0., 0.], [0., -1., 0.], [0., -1., 0.], [-1., 0., 0.]),
            ([1., 0., 0.], [0., 1., 0.], [0., 1., 0.], [-1., 0., 0.]),
            ([1., 0., 0.], [0., 0., 1.], [0., 0., 1.], [-1., 0., 0.]),
            ([0., 1., 0.], [0., 0., 1.], [0., 0., 1.], [0., -1., 0.]),
            ([-1., 0., 0.], [0., 0., 1.], [0., 0., -1.], [-1., 0., 0.]),
            ([0., -1., 0.], [0., 0., 1.], [0., 0., -1.], [0., -1., 0.]),
        ];
        for (native_x, native_y, expected_x, expected_y) in cases {
            let native = SketchPlane::new([0.; 3], native_x, native_y, Default::default()).unwrap();
            let converted = go_prism_frame(native).unwrap().recipe();
            assert_eq!(converted.x_axis, expected_x);
            assert_eq!(converted.y_axis, expected_y);
        }
    }
}
