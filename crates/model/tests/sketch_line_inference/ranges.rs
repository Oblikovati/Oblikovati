// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn malformed_policy_and_coordinates_decline_even_when_inference_is_disabled() {
    let sketch = Sketch2D::default();
    let mut disabled = default_sketch_settings();
    disabled.infer_constraints = false;
    for angle in [
        0.0,
        -1.0,
        f64::NAN,
        f64::INFINITY,
        std::f64::consts::FRAC_PI_4,
    ] {
        let bad = SketchInferencePolicy {
            angle_tolerance_radians: angle,
            ..policy()
        };
        assert_policy_decline(&sketch, &disabled, bad, "inference angle");
    }
}
#[test]
fn invalid_snap_reach_declines_even_when_disabled() {
    let sketch = Sketch2D::default();
    let mut disabled = default_sketch_settings();
    disabled.infer_constraints = false;
    for distance in [0.0, -1.0, f64::NAN, f64::INFINITY] {
        let bad = SketchInferencePolicy {
            snap_distance_working: distance,
            ..policy()
        };
        assert!(
            sketch
                .infer_line_placement([[0.0; 2], [1.0, 0.0]], &disabled, bad)
                .unwrap_err()
                .to_string()
                .contains("snap distance")
        );
    }
}
#[test]
fn nonfinite_endpoints_are_refused_before_any_proposal() {
    for invalid in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let error = Sketch2D::default()
            .infer_line_placement(
                [[invalid, 0.0], [1.0, 0.0]],
                &default_sketch_settings(),
                policy(),
            )
            .unwrap_err();
        assert!(error.to_string().contains("position") && error.to_string().contains("expected"));
    }
}
#[test]
fn zero_length_probe_and_reference_do_not_define_a_parallel_relation() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 5, [[4.0, 4.0]; 2]);
    assert!(
        infer(&sketch, [[0.0; 2], [1.0, 1.0]], PRIORITY_NONE)
            .direction
            .is_none()
    );
    let mut nonzero = Sketch2D::default();
    add_line(&mut nonzero, 5, [[1.0, 4.0], [2.0, 5.0]]);
    assert!(
        infer(&nonzero, [[0.0; 2]; 2], PRIORITY_NONE)
            .direction
            .is_none()
    );
}
#[test]
fn point_snaps_use_hypot_without_underflow_or_overflow_false_coincidences() {
    let mut sketch = Sketch2D::default();
    sketch
        .add_point(SketchEntityId(1), [-f64::MAX, 0.0])
        .unwrap();
    assert!(
        infer(&sketch, [[f64::MAX, 0.0], [1.0, 1.0]], PRIORITY_NONE)
            .points
            .is_empty()
    );
}
#[test]
fn subnormal_point_distance_is_not_rounded_to_false_coincidence() {
    let mut tiny = Sketch2D::default();
    tiny.add_point(SketchEntityId(1), [0.0; 2]).unwrap();
    let tiny_policy = SketchInferencePolicy {
        snap_distance_working: 1e-300,
        ..policy()
    };
    let report = tiny
        .infer_line_placement(
            [[2e-300, 0.0], [1.0, 1.0]],
            &default_sketch_settings(),
            tiny_policy,
        )
        .unwrap();
    assert!(report.points.is_empty());
}
#[test]
fn finite_large_and_subnormal_lines_preserve_axis_and_relative_families() {
    for magnitude in [1e-300, 1e300] {
        let mut sketch = Sketch2D::default();
        add_line(&mut sketch, 5, [[0.0, 0.0], [magnitude, magnitude]]);
        let exact = direction(&sketch, [[0.0; 2], [magnitude, magnitude]], PRIORITY_NONE);
        assert_eq!(exact.kind, INFER_PARALLEL);
        assert_eq!(
            direction(
                &Sketch2D::default(),
                [[0.0; 2], [magnitude, 0.0]],
                PRIORITY_NONE
            )
            .kind,
            INFER_HORIZONTAL
        );
    }
}
#[test]
fn overflowing_coordinate_differences_keep_their_actual_direction() {
    let mut sketch = Sketch2D::default();
    add_line(
        &mut sketch,
        5,
        [[-f64::MAX, -f64::MAX], [f64::MAX, f64::MAX]],
    );
    let result = direction(
        &sketch,
        [[-f64::MAX, f64::MAX], [f64::MAX, -f64::MAX]],
        PRIORITY_NONE,
    );
    assert_eq!(result.kind, INFER_PERPENDICULAR);
}
#[test]
fn unknown_stored_priority_uses_the_go_horizontal_vertical_default() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 5, [[0.0, 5.0], [4.0, 5.02]]);
    assert_eq!(
        direction(
            &sketch,
            [[0.0; 2], [4.0, 0.04]],
            ConstraintInferencePriority(9876)
        )
        .kind,
        INFER_HORIZONTAL
    );
}

fn assert_policy_decline(
    sketch: &Sketch2D,
    settings: &SketchSettings,
    policy: SketchInferencePolicy,
    label: &str,
) {
    let error = sketch
        .infer_line_placement([[0.0; 2], [1.0, 0.0]], settings, policy)
        .unwrap_err()
        .to_string();
    assert!(
        error.contains(label) && error.contains("expected"),
        "{error}"
    );
}
#[test]
fn subnormal_squared_lengths_do_not_create_a_false_parallel_suggestion() {
    let mut sketch = Sketch2D::default();
    add_line(&mut sketch, 5, [[0.0; 2], [2e-162, 2e-162]]);
    let report = infer(&sketch, [[0.0; 2], [2e-162, 2.5e-162]], PRIORITY_NONE);
    assert!(report.direction.is_none(), "{report:?}");
}
