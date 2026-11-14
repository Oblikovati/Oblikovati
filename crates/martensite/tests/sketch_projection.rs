// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::sketch_projection;
use oblikovati_model::sketch::SketchPlane;
#[test]
fn camera_lines_map_to_authored_frames_in_both_directions_and_working_scales() {
    for scale in [1e-6, 1., 1e6] {
        for plane in [
            SketchPlane::xy(),
            SketchPlane::xz(),
            SketchPlane::yz(),
            tilted(),
        ] {
            let expected = [3. * scale, -2. * scale];
            let hit = plane.to_model(expected).unwrap();
            for distance in [-7. * scale, 7. * scale] {
                let origin = std::array::from_fn(|i| hit[i] + distance * plane.normal()[i]);
                let result = sketch_projection(plane, origin, plane.normal(), 1e-9)
                    .unwrap()
                    .unwrap();
                for (a, b) in result.into_iter().zip(expected) {
                    assert!((a - b).abs() < 1e-9 * scale.max(1.));
                }
            }
        }
    }
}
fn tilted() -> SketchPlane {
    SketchPlane::new(
        [7., -4., 9.],
        [0.6, 0., 0.8],
        [0., 1., 0.],
        Default::default(),
    )
    .unwrap()
}
#[test]
fn edge_on_and_nearly_edge_on_lines_have_no_unique_usable_cursor() {
    for origin in [[0.; 3], [1., 2., 3.]] {
        for z in [0., 0.5e-9, -0.5e-9, 1e-9, -1e-9] {
            assert_eq!(
                sketch_projection(SketchPlane::xy(), origin, [1., 0., z], 1e-9).unwrap(),
                None
            );
        }
    }
    let hit = sketch_projection(SketchPlane::xy(), [0., 0., 1.], [1., 0., 2e-9], 1e-9)
        .unwrap()
        .unwrap();
    assert!((hit[0] + 5e8).abs() < 1e-6);
}
#[test]
fn invalid_and_unrepresentable_projection_inputs_return_errors() {
    for cosine in [0., -1., 2., f64::NAN, f64::INFINITY] {
        assert!(sketch_projection(SketchPlane::xy(), [0.; 3], [0., 0., 1.], cosine).is_err());
    }
    for direction in [
        [0.; 3],
        [f64::NAN, 0., 1.],
        [f64::INFINITY, 0., 1.],
        [f64::MAX; 3],
    ] {
        assert!(sketch_projection(SketchPlane::xy(), [0.; 3], direction, 1e-9).is_err());
    }
    assert!(sketch_projection(tilted(), [f64::NAN; 3], [0., 0., 1.], 1e-9).is_err());
    assert!(sketch_projection(SketchPlane::xy(), [f64::MAX; 3], [1., 0., -1.], 1e-9).is_err());
}
