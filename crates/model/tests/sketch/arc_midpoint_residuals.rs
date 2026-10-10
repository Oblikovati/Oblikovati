// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
use oblikovati_solve::residual::ResidualSource;
fn evaluated(input: &[f64], counter_clockwise: bool) -> Vec<DualNumber> {
    let source = PreparedConstraint {
        equation: PlanarEquation::ArcMidpoint(counter_clockwise),
        projection: None,
        variables: (0..8).collect(),
    };
    let result = source.evaluate(&DualNumber::seed(input).unwrap()).unwrap();
    assert_eq!(source.equations(), 2);
    result
}
fn scalar(input: &[f64], counter_clockwise: bool) -> [f64; 2] {
    let start = (input[5] - input[3]).atan2(input[4] - input[2]);
    let end = (input[7] - input[3]).atan2(input[6] - input[2]);
    let sweep = if counter_clockwise {
        (end - start).rem_euclid(TAU)
    } else {
        -(start - end).rem_euclid(TAU)
    };
    let radius = (input[4] - input[2]).hypot(input[5] - input[3]);
    let angle = start + sweep / 2.0;
    [
        input[0] - input[2] - radius * angle.cos(),
        input[1] - input[3] - radius * angle.sin(),
    ]
}
#[test]
fn both_directed_midpoint_rows_match_independent_polar_values_and_derivatives() {
    for counter_clockwise in [false, true] {
        let input = [5.0, 6.0, 1.0, 2.0, 4.0, 6.0, -3.0, 5.0];
        let actual = evaluated(&input, counter_clockwise);
        for (row, actual) in actual.iter().enumerate() {
            assert!((actual.value() - scalar(&input, counter_clockwise)[row]).abs() < 1e-12);
            for column in 0..8 {
                let mut left = input;
                let mut right = input;
                left[column] -= 1e-5;
                right[column] += 1e-5;
                let partial = (scalar(&right, counter_clockwise)[row]
                    - scalar(&left, counter_clockwise)[row])
                    / 2e-5;
                assert!((actual.gradient()[column] - partial).abs() < 1e-8);
            }
        }
    }
}
#[test]
fn midpoint_values_and_partials_keep_their_units_at_extreme_scales() {
    for counter_clockwise in [false, true] {
        let input = [5.0, 6.0, 1.0, 2.0, 4.0, 6.0, -3.0, 5.0];
        let reference = evaluated(&input, counter_clockwise);
        for scale in [1e-200, 1e-100, 1.0, 1e100, 1e200] {
            let actual = evaluated(&input.map(|value| value * scale), counter_clockwise);
            for row in 0..2 {
                assert!((actual[row].value() / scale - reference[row].value()).abs() < 1e-12);
                for (&actual, &expected) in
                    actual[row].gradient().iter().zip(reference[row].gradient())
                {
                    assert!((actual - expected).abs() < 1e-12);
                }
            }
        }
    }
}
#[test]
fn ambiguous_sweeps_collapsed_radials_and_wrong_shapes_are_named_declines() {
    let source = PreparedConstraint {
        equation: PlanarEquation::ArcMidpoint(true),
        projection: None,
        variables: (0..8).collect(),
    };
    assert!(
        source
            .evaluate(&[])
            .unwrap_err()
            .to_string()
            .contains("expected 8")
    );
    for input in [
        [0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 1.0],
        [0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, 0.0],
    ] {
        assert!(
            source
                .evaluate(&DualNumber::seed(&input).unwrap())
                .unwrap_err()
                .to_string()
                .contains("nonzero")
        );
    }
    let input = [0.0, 0.0, 0.0, 0.0, 1.0, 0.0, 2.0, 0.0];
    assert!(
        source
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap_err()
            .to_string()
            .contains("ambiguous")
    );
}
#[test]
fn rotation_across_polar_wrap_and_half_circle_boundaries_is_continuous() {
    for counter_clockwise in [false, true] {
        for start in [std::f64::consts::PI - 1e-8, -std::f64::consts::PI + 1e-8] {
            let end = start + if counter_clockwise { 0.5 } else { -0.5 };
            let input = [
                0.0,
                0.0,
                0.0,
                0.0,
                start.cos(),
                start.sin(),
                end.cos(),
                end.sin(),
            ];
            let actual = evaluated(&input, counter_clockwise);
            let expected = scalar(&input, counter_clockwise);
            for row in 0..2 {
                assert!((actual[row].value() - expected[row]).abs() < 1e-12);
            }
        }
        for end in [std::f64::consts::PI - 1e-8, std::f64::consts::PI + 1e-8] {
            let input = [0.0, 0.0, 0.0, 0.0, 1.0, 0.0, end.cos(), end.sin()];
            let actual = evaluated(&input, counter_clockwise);
            let expected = scalar(&input, counter_clockwise);
            for row in 0..2 {
                assert!((actual[row].value() - expected[row]).abs() < 1e-12);
            }
        }
    }
}

#[test]
fn near_zero_and_near_full_sweeps_keep_distinct_midpoint_branches() {
    for counter_clockwise in [false, true] {
        for end_angle in [-1e-10_f64, 1e-10] {
            let input = [
                0.0,
                0.0,
                0.0,
                0.0,
                1.0,
                0.0,
                end_angle.cos(),
                end_angle.sin(),
            ];
            let actual = evaluated(&input, counter_clockwise);
            let expected = scalar(&input, counter_clockwise);
            for (row, actual) in actual.iter().enumerate() {
                assert!((actual.value() - expected[row]).abs() < 1e-12);
            }
            let minor =
                (counter_clockwise && end_angle > 0.0) || (!counter_clockwise && end_angle < 0.0);
            assert_eq!(actual[0].value().is_sign_negative(), minor);
        }
    }
}
