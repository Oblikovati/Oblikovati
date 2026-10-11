// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{SketchFrameAxis, SketchPlane};
const FRAME_CHOICES: [(SketchFrameAxis, bool); 4] = [
    (SketchFrameAxis::X, false),
    (SketchFrameAxis::X, true),
    (SketchFrameAxis::Y, false),
    (SketchFrameAxis::Y, true),
];
type OrientationCase = (SketchPlane, [f64; 3], [f64; 3]);

fn assert_vector(actual: [f64; 3], expected: [f64; 3]) {
    for (actual, expected) in actual.into_iter().zip(expected) {
        assert!(
            (actual - expected).abs() <= 32.0 * f64::EPSILON,
            "component {actual}: expected {expected}"
        );
    }
}

fn chosen_axis(plane: SketchPlane, axis: SketchFrameAxis) -> [f64; 3] {
    match axis {
        SketchFrameAxis::X => plane.recipe().x_axis,
        SketchFrameAxis::Y => plane.recipe().y_axis,
    }
}

#[test]
fn origin_frames_select_either_axis_and_reverse_without_changing_facing() {
    for (base, direction, expected) in origin_cases() {
        for (axis, reverse) in FRAME_CHOICES {
            assert_orientation(base, direction, axis, reverse, expected);
        }
    }
}
fn origin_cases() -> [OrientationCase; 3] {
    [
        (SketchPlane::xy(), [1.0, 0.0, 2.0], [1.0, 0.0, 0.0]),
        (SketchPlane::xz(), [0.0, 2.0, 3.0], [0.0, 0.0, 1.0]),
        (SketchPlane::yz(), [3.0, 2.0, 0.0], [0.0, 1.0, 0.0]),
    ]
}
fn assert_orientation(
    base: SketchPlane,
    direction: [f64; 3],
    axis: SketchFrameAxis,
    reverse: bool,
    expected: [f64; 3],
) {
    let oriented = base
        .with_orientation(direction, axis, reverse, None)
        .unwrap();
    assert_vector(
        chosen_axis(oriented, axis),
        expected.map(|value| if reverse { -value } else { value }),
    );
    assert_vector(oriented.normal(), base.normal());
    assert_eq!(oriented.recipe().origin, base.recipe().origin);
}

fn tilted_plane() -> SketchPlane {
    let angle = std::f64::consts::PI / 6.0;
    SketchPlane::new(
        [2.0, 3.0, 4.0],
        [0.0, 0.0, -1.0],
        [angle.cos(), angle.sin(), 0.0],
        Default::default(),
    )
    .unwrap()
}

#[test]
fn angled_meridian_matches_go_orientation_axis_and_preserves_plane_facing() {
    for (axis, reverse) in FRAME_CHOICES {
        assert_orientation(
            tilted_plane(),
            [0.0, 0.0, 1.0],
            axis,
            reverse,
            [0.0, 0.0, 1.0],
        );
    }
}

#[test]
fn direction_scale_does_not_change_the_result_even_at_finite_extremes() {
    let base = SketchPlane::xy();
    let expected = [
        std::f64::consts::FRAC_1_SQRT_2,
        std::f64::consts::FRAC_1_SQRT_2,
        0.0,
    ];
    for scale in [f64::from_bits(1), f64::MIN_POSITIVE, 1.0, f64::MAX] {
        let result = base
            .with_orientation([scale; 3], SketchFrameAxis::X, false, None)
            .unwrap();
        assert_vector(result.recipe().x_axis, expected);
        assert_vector(result.normal(), base.normal());
    }
}

#[test]
fn invalid_directions_and_origins_return_named_errors_without_changing_input() {
    let base = SketchPlane::xy();
    let before = base.recipe();
    for direction in [
        [0.0; 3],
        [0.0, 0.0, 1.0],
        [f64::NAN, 1.0, 0.0],
        [f64::INFINITY, 0.0, 0.0],
    ] {
        let error = base
            .with_orientation(direction, SketchFrameAxis::X, false, None)
            .unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert!(error.to_string().contains("direction"));
    }
    assert_eq!(base.recipe(), before);
}
#[test]
fn invalid_origin_is_refused_after_orientation_is_resolved() {
    let result = SketchPlane::xy().with_orientation(
        [1.0, 0.0, 0.0],
        SketchFrameAxis::Y,
        false,
        Some([f64::NAN; 3]),
    );
    assert!(result.unwrap_err().to_string().contains("origin"));
}

#[test]
fn optional_origin_and_restored_frame_keep_coordinate_round_trips() {
    let origin = [13.0, -17.0, 29.0];
    let result = tilted_plane()
        .with_orientation([3.0, 5.0, 7.0], SketchFrameAxis::Y, true, Some(origin))
        .unwrap();
    assert_eq!(result.recipe().origin, origin);
    let restored = SketchPlane::from_recipe(result.recipe(), result.options()).unwrap();
    assert!(restored.same_recipe(result));
    for local in [[0.0, 0.0], [5.0, 7.0], [-2.0, 3.0]] {
        let round_trip = result.to_sketch(result.to_model(local).unwrap()).unwrap();
        assert_vector(
            [round_trip[0], round_trip[1], 0.0],
            [local[0], local[1], 0.0],
        );
    }
}

#[test]
fn angular_policy_gates_near_perpendicular_directions_by_conditioning() {
    let base = SketchPlane::xy();
    let limit = base.options().angular_deviation;
    assert!(
        base.with_orientation([limit / 2.0, 0.0, 1.0], SketchFrameAxis::X, false, None)
            .is_err()
    );
    let accepted = base
        .with_orientation([limit * 2.0, 0.0, 1.0], SketchFrameAxis::X, false, None)
        .unwrap();
    assert_vector(accepted.recipe().x_axis, [1.0, 0.0, 0.0]);
}

fn oblique_plane() -> SketchPlane {
    let half = std::f64::consts::FRAC_1_SQRT_2;
    let sixth = 1.0 / 6.0_f64.sqrt();
    SketchPlane::new(
        [5.0, 7.0, 11.0],
        [half, half, 0.0],
        [-sixth, sixth, 2.0 * sixth],
        Default::default(),
    )
    .unwrap()
}
fn oblique_case(sample: u32) -> OrientationCase {
    let base = oblique_plane();
    let a = f64::from(sample) / 8.0;
    let b = (a * 2.0).cos();
    let c = (a * 3.0).sin();
    let recipe = base.recipe();
    let direction =
        std::array::from_fn(|i| a * recipe.x_axis[i] + b * recipe.y_axis[i] + c * base.normal()[i]);
    let length = (a * a + b * b).sqrt();
    let expected = std::array::from_fn(|i| (a * recipe.x_axis[i] + b * recipe.y_axis[i]) / length);
    (base, direction, expected)
}
#[test]
fn generated_oblique_directions_match_independent_local_basis_projection() {
    for sample in 1..=64 {
        let (base, direction, expected) = oblique_case(sample);
        for (axis, reverse) in FRAME_CHOICES {
            assert_orientation(base, direction, axis, reverse, expected);
        }
    }
}
