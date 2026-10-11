// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::SketchPlane;
#[test]
fn direction_constructor_preserves_origin_and_normalizes_extreme_axes() {
    for scale in [f64::from_bits(1), 2.0, f64::MAX] {
        let frame = SketchPlane::from_directions(
            [-0.0, 0.0, 5.0],
            [scale, 0.0, 0.0],
            [0.0, scale, 0.0],
            Default::default(),
        )
        .unwrap();
        assert_eq!(frame.normal(), [0.0, 0.0, 1.0]);
        assert_eq!(
            frame.recipe().origin.map(f64::to_bits),
            [-0.0f64, 0.0, 5.0].map(f64::to_bits)
        );
        let restored = SketchPlane::from_recipe(frame.recipe(), frame.options()).unwrap();
        assert!(frame.same_recipe(restored));
    }
}
#[test]
fn malformed_direction_frames_decline_before_construction() {
    for direction in [[0.0; 3], [f64::NAN, 0.0, 0.0], [0.0, 1.0, 0.0]] {
        assert!(
            SketchPlane::from_directions([0.0; 3], direction, [0.0, 1.0, 0.0], Default::default())
                .is_err()
        );
    }
    assert!(
        SketchPlane::from_directions(
            [f64::INFINITY, 0.0, 0.0],
            [1.0, 0.0, 0.0],
            [0.0, 1.0, 0.0],
            Default::default()
        )
        .is_err()
    );
}
