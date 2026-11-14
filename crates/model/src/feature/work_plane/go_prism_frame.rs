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

    #[derive(serde::Deserialize)]
    struct OracleFrame {
        input: [f64; 3],
        u: [f64; 3],
        v: [f64; 3],
    }

    fn independent_native_frame(normal: [f64; 3]) -> SketchPlane {
        let seed = if normal[2].abs() > 0.9 * normal.iter().map(|v| v * v).sum::<f64>().sqrt() {
            [0., 1., 0.]
        } else {
            [0., 0., 1.]
        };
        let x = cross(seed, normal);
        let y = cross(normal, x);
        SketchPlane::from_directions([1., 2., 3.], x, y, Default::default()).unwrap()
    }

    #[test]
    fn go_analytic_oracle_matches_oblique_and_axis_boundary_frames() {
        let source = include_str!(
            "../../../../../architecture/rust-port/receipts/2026-11-14-go-frame-oracle/frames.json"
        );
        let cases: Vec<OracleFrame> = serde_json::from_str(source).unwrap();
        assert_eq!(cases.len(), 11);
        for case in cases {
            let frame = go_prism_frame(independent_native_frame(case.input))
                .unwrap()
                .recipe();
            for (actual, expected) in frame
                .x_axis
                .into_iter()
                .zip(case.u)
                .chain(frame.y_axis.into_iter().zip(case.v))
            {
                assert!(
                    (actual - expected).abs() < 1e-12,
                    "input {:?}: actual {actual}, expected {expected}",
                    case.input
                );
            }
            assert_eq!(frame.origin, [1., 2., 3.]);
        }
    }

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
