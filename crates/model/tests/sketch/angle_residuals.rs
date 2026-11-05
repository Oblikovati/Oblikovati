// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
use oblikovati_solve::residual::ResidualSource;
fn source(target: f64, turn: AngleTurn) -> PreparedConstraint {
    PreparedConstraint {
        equation: PlanarEquation::Angle { target, turn },
        projection: None,
        variables: (0..8).collect(),
    }
}
#[test]
fn signed_angles_match_independent_polar_differences_and_every_partial() {
    for turn in [AngleTurn::CounterClockwise, AngleTurn::Clockwise] {
        let input = [1.0, 2.0, 5.0, 3.0, 4.0, 6.0, 2.0, 9.0];
        let source = source(PI / 3.0, turn);
        let actual = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
        let scalar = |v: &[f64; 8]| {
            let angle = (v[7] - v[5]).atan2(v[6] - v[4]) - (v[3] - v[1]).atan2(v[2] - v[0]);
            let target = if turn == AngleTurn::CounterClockwise {
                PI / 3.0
            } else {
                -PI / 3.0
            };
            (angle - target + PI).rem_euclid(TAU) - PI
        };
        assert!((actual[0].value() - scalar(&input)).abs() < 1e-12);
        for column in 0..8 {
            let mut left = input;
            let mut right = input;
            left[column] -= 1e-5;
            right[column] += 1e-5;
            let partial = (scalar(&right) - scalar(&left)) / 2e-5;
            assert!((actual[0].gradient()[column] - partial).abs() < 1e-8);
        }
    }
}
#[test]
fn angular_values_stay_dimensionless_and_partials_scale_as_inverse_length() {
    let input = [0.0, 0.0, 4.0, 1.0, 2.0, 3.0, 3.0, 5.0];
    let source = source(PI / 4.0, AngleTurn::CounterClockwise);
    let reference = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
    for scale in [1e-200, 1e-100, 1.0, 1e100, 1e200] {
        let actual = source
            .evaluate(&DualNumber::seed(&input.map(|v| v * scale)).unwrap())
            .unwrap();
        assert!((actual[0].value() - reference[0].value()).abs() < 1e-12);
        for (&actual, &expected) in actual[0].gradient().iter().zip(reference[0].gradient()) {
            assert!((actual * scale - expected).abs() < 1e-12);
        }
    }
}
#[test]
fn zero_and_straight_angles_have_simple_roots_and_wrap_both_clockwise_boundaries() {
    for turn in [AngleTurn::CounterClockwise, AngleTurn::Clockwise] {
        for (target, x) in [(0.0, 1.0), (PI, -1.0)] {
            let input = [0.0, 0.0, 1.0, 0.0, 0.0, 0.0, x, 0.0];
            let actual = source(target, turn)
                .evaluate(&DualNumber::seed(&input).unwrap())
                .unwrap();
            assert_eq!(actual[0].value(), 0.0);
            assert!(
                actual[0]
                    .gradient()
                    .iter()
                    .any(|partial| partial.abs() >= 1.0)
            );
        }
    }
    for turn in [AngleTurn::CounterClockwise, AngleTurn::Clockwise] {
        let input = [
            0.0,
            0.0,
            1.0,
            0.0,
            0.0,
            0.0,
            -1.0,
            if turn == AngleTurn::Clockwise {
                0.01
            } else {
                -0.01
            },
        ];
        let actual = source(PI - 0.01, turn)
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap();
        assert!(actual[0].value().abs() < 0.03);
    }
}
#[test]
fn malformed_or_collapsed_angle_sources_return_named_errors() {
    let source = source(1.0, AngleTurn::CounterClockwise);
    assert!(
        source
            .evaluate(&[])
            .unwrap_err()
            .to_string()
            .contains("expected 8")
    );
    for input in [
        [0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 1.0, 0.0],
        [0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0],
    ] {
        assert!(
            source
                .evaluate(&DualNumber::seed(&input).unwrap())
                .unwrap_err()
                .to_string()
                .contains("nonzero")
        );
    }
}

#[test]
fn ordinary_signed_rows_match_go_unsigned_rows_with_the_recorded_turn_sign() {
    for fixture in include_str!("../fixtures/go-angle-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let input: Vec<f64> = fields[1]
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        let turn = if fields[0] == "ccw" {
            AngleTurn::CounterClockwise
        } else {
            AngleTurn::Clockwise
        };
        let sign = if turn == AngleTurn::CounterClockwise {
            1.0
        } else {
            -1.0
        };
        let actual = source(PI / 3.0, turn)
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap();
        assert!((actual[0].value() - fields[2].parse::<f64>().unwrap() * sign).abs() < 1e-12);
        let expected: Vec<f64> = fields[3]
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        assert_eq!(actual[0].gradient().len(), expected.len());
        for (&actual, expected) in actual[0].gradient().iter().zip(expected) {
            assert!((actual - expected * sign).abs() < 1e-12);
        }
    }
}
