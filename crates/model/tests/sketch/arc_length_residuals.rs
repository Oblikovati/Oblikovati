// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
use oblikovati_solve::residual::ResidualSource;
use std::f64::consts::{PI, TAU};
fn source(target: f64, counter_clockwise: bool) -> PreparedConstraint {
    PreparedConstraint {
        equation: PlanarEquation::ArcLength {
            target,
            counter_clockwise,
        },
        projection: None,
        variables: (0..6).collect(),
    }
}
fn scalar(input: &[f64; 6], counter_clockwise: bool, target: f64) -> f64 {
    let start = (input[3] - input[1]).atan2(input[2] - input[0]);
    let end = (input[5] - input[1]).atan2(input[4] - input[0]);
    let sweep = if counter_clockwise {
        (end - start).rem_euclid(TAU)
    } else {
        (start - end).rem_euclid(TAU)
    };
    (input[2] - input[0]).hypot(input[3] - input[1]) * sweep - target
}
#[test]
fn minor_and_major_lengths_match_analytic_values_and_all_coordinate_derivatives() {
    for counter_clockwise in [false, true] {
        let input = [1.0, 2.0, 4.0, 6.0, -3.0, 5.0];
        let actual = source(2.0, counter_clockwise)
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap();
        let sweep = if counter_clockwise {
            PI / 2.0
        } else {
            3.0 * PI / 2.0
        };
        assert!((actual[0].value() - (5.0 * sweep - 2.0)).abs() < 1e-12);
        for column in 0..6 {
            let mut left = input;
            let mut right = input;
            left[column] -= 1e-5;
            right[column] += 1e-5;
            let partial = (scalar(&right, counter_clockwise, 2.0)
                - scalar(&left, counter_clockwise, 2.0))
                / 2e-5;
            assert!((actual[0].gradient()[column] - partial).abs() < 1e-8);
        }
    }
}
#[test]
fn directed_lengths_keep_units_and_gradients_at_extreme_scales() {
    let input = [1.0, 2.0, 4.0, 6.0, -3.0, 5.0];
    for counter_clockwise in [false, true] {
        let reference = source(2.0, counter_clockwise)
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap();
        for scale in [1e-200, 1e-100, 1.0, 1e100, 1e200] {
            let actual = source(2.0 * scale, counter_clockwise)
                .evaluate(&DualNumber::seed(&input.map(|value| value * scale)).unwrap())
                .unwrap();
            assert!((actual[0].value() / scale - reference[0].value()).abs() < 1e-12);
            for (&actual, &expected) in actual[0].gradient().iter().zip(reference[0].gradient()) {
                assert!((actual - expected).abs() < 1e-12);
            }
        }
    }
}
#[test]
fn ambiguous_sweeps_collapsed_radii_and_wrong_shapes_are_refused() {
    let source = source(1.0, true);
    assert!(
        source
            .evaluate(&[])
            .unwrap_err()
            .to_string()
            .contains("expected 6")
    );
    for input in [
        [0.0, 0.0, 0.0, 0.0, 0.0, 1.0],
        [0.0, 0.0, 1.0, 0.0, 0.0, 0.0],
        [0.0, 0.0, 1.0, 0.0, 2.0, 0.0],
    ] {
        assert!(source.evaluate(&DualNumber::seed(&input).unwrap()).is_err());
    }
}

#[test]
fn minor_sweep_length_rows_match_go_in_both_directions() {
    for fixture in include_str!("../fixtures/go-circular-dimensions.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        if !fields[0].starts_with("length-") {
            continue;
        }
        let input: Vec<f64> = fields[1]
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        let actual = source(2.0, fields[0] == "length-ccw")
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap();
        assert!((actual[0].value() - fields[2].parse::<f64>().unwrap()).abs() < 1e-12);
        let expected: Vec<f64> = fields[3]
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        assert_eq!(actual[0].gradient().len(), expected.len());
        for (&actual, expected) in actual[0].gradient().iter().zip(expected) {
            assert!((actual - expected).abs() < 1e-12);
        }
    }
}
#[test]
fn complementary_sweep_values_and_gradients_sum_to_the_full_circumference() {
    let input = [1.0, 2.0, 4.0, 6.0, -3.0, 5.0];
    let variables = DualNumber::seed(&input).unwrap();
    let first = source(2.0, true).evaluate(&variables).unwrap();
    let second = source(2.0, false).evaluate(&variables).unwrap();
    assert!((first[0].value() + second[0].value() + 4.0 - TAU * 5.0).abs() < 1e-12);
    for (column, expected) in [-0.6, -0.8, 0.6, 0.8, 0.0, 0.0].into_iter().enumerate() {
        assert!(
            (first[0].gradient()[column] + second[0].gradient()[column] - TAU * expected).abs()
                < 1e-12
        );
    }
}
