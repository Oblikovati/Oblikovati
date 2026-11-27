// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn go_origin_to_top_point_case_and_directed_reversal_are_preserved() {
    let first = [0.0; 3];
    let second = [0.0, 0.0, 4.0];
    let forward = WorkAxisLine::through_points(first, second).unwrap();
    let reverse = WorkAxisLine::through_points(second, first).unwrap();
    assert_eq!(forward.direction(), [0.0, 0.0, 1.0]);
    assert_eq!(reverse.direction(), [0.0, 0.0, -1.0]);
    assert_eq!(reverse.origin(), second);
}
#[test]
fn finite_extreme_and_subnormal_points_produce_checked_lines() {
    for scale in [f64::from_bits(1), f64::MIN_POSITIVE, 1.0, f64::MAX] {
        let axis = WorkAxisLine::through_points([-scale, 0.0, 0.0], [scale, 0.0, 0.0]).unwrap();
        assert_eq!(axis.direction(), [1.0, 0.0, 0.0]);
        assert_eq!(axis.origin(), [-scale, 0.0, 0.0]);
    }
    let axis = WorkAxisLine::through_points([-f64::MAX, -f64::MAX, 0.0], [f64::MAX, f64::MAX, 0.0])
        .unwrap();
    assert!((axis.direction()[0] - std::f64::consts::FRAC_1_SQRT_2).abs() < f64::EPSILON);
    assert_eq!(axis.direction()[0], axis.direction()[1]);
}
#[test]
fn invalid_and_coincident_point_inputs_have_no_line_result() {
    for (first, second) in [
        ([f64::NAN, 0.0, 0.0], [0.0; 3]),
        ([0.0; 3], [f64::INFINITY, 0.0, 0.0]),
        ([1.0; 3], [1.0; 3]),
    ] {
        let error = WorkAxisLine::through_points(first, second).unwrap_err();
        assert!(error.to_string().contains("expected"));
    }
}
#[test]
fn generated_point_pairs_keep_their_origins_and_unit_direction() {
    for sample in 1..=256 {
        let first = [sample as f64, -2.0, 3.0];
        let second = [sample as f64 + 2.0, 4.0, 6.0];
        let axis = WorkAxisLine::through_points(first, second).unwrap();
        assert_eq!(axis.origin(), first);
        let direct = WorkAxisLine::new(first, [2.0, 6.0, 3.0]).unwrap();
        assert!(axis.same_recipe(direct));
        for (actual, expected) in
            axis.direction()
                .into_iter()
                .zip([2.0 / 7.0, 6.0 / 7.0, 3.0 / 7.0])
        {
            assert!((actual - expected).abs() < f64::EPSILON);
        }
    }
}

#[test]
fn overflowing_differences_retain_adjacent_finite_component_increments() {
    let first = [-f64::MAX, f64::MAX / 2.0, 0.0];
    let next = f64::from_bits(first[1].to_bits() + 1);
    let axis = WorkAxisLine::through_points(first, [f64::MAX, next, 0.0]).unwrap();
    let expected_slope = (next - first[1]) / f64::MAX / 2.0;
    assert_eq!(axis.direction()[0], 1.0);
    assert_eq!(axis.direction()[1].to_bits(), expected_slope.to_bits());
    assert!(expected_slope > 0.0);
}
