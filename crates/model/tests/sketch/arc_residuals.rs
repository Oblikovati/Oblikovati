// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
use oblikovati_solve::residual::{ResidualBlock, ResidualSource, evaluate_residuals};
fn evaluated(input: &[f64]) -> DualNumber {
    let source = PreparedConstraint {
        equation: PlanarEquation::ArcCircularity,
        variables: (0..6).collect(),
    };
    let rows = source.evaluate(&DualNumber::seed(input).unwrap()).unwrap();
    assert_eq!(rows.len(), source.equations());
    rows[0].clone()
}
fn scalar(input: &[f64]) -> f64 {
    (input[4] - input[0]).hypot(input[5] - input[1])
        - (input[2] - input[0]).hypot(input[3] - input[1])
}
#[test]
fn circularity_matches_analytic_residual_and_independent_center_start_end_derivatives() {
    let input = [1.0, 2.0, 4.0, 6.0, 1.0, 12.0];
    let actual = evaluated(&input);
    assert_eq!(actual.value(), 5.0);
    for (&actual, expected) in actual
        .gradient()
        .iter()
        .zip([0.6, -0.2, -0.6, -0.8, 0.0, 1.0])
    {
        assert!((actual - expected).abs() < 1e-12);
    }
    for column in 0..6 {
        let mut left = input;
        let mut right = input;
        left[column] -= 1e-5;
        right[column] += 1e-5;
        let partial = (scalar(&right) - scalar(&left)) / 2e-5;
        assert!((actual.gradient()[column] - partial).abs() < 1e-8);
    }
}
#[test]
fn circularity_preserves_value_units_and_partials_without_squared_distance_overflow() {
    let reference = evaluated(&[0.0, 0.0, 3.0, 4.0, 0.0, 10.0]);
    for scale in [1e-200, 1e-100, 1.0, 1e100, 1e200] {
        let actual = evaluated(&[0.0, 0.0, 3.0 * scale, 4.0 * scale, 0.0, 10.0 * scale]);
        assert!((actual.value() / scale - 5.0).abs() < 1e-12);
        for (&actual, &expected) in actual.gradient().iter().zip(reference.gradient()) {
            assert!((actual - expected).abs() < 1e-12);
        }
    }
}
#[test]
fn collapsed_radius_and_wrong_variable_shapes_are_named_declines() {
    let source = PreparedConstraint {
        equation: PlanarEquation::ArcCircularity,
        variables: (0..6).collect(),
    };
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
    ] {
        assert!(
            source
                .evaluate(&DualNumber::seed(&input).unwrap())
                .unwrap_err()
                .to_string()
                .contains("positive")
        );
    }
}
#[test]
fn repeated_global_endpoint_columns_cancel_without_losing_the_structural_row() {
    let source = PreparedConstraint {
        equation: PlanarEquation::ArcCircularity,
        variables: vec![0, 1, 2, 3, 2, 3],
    };
    let block = ResidualBlock {
        source: &source,
        variables: &source.variables,
        equations: 1,
    };
    let evaluated = evaluate_residuals(&[0.0, 0.0, 3.0, 4.0], &[block]).unwrap();
    assert_eq!(evaluated.values(), [0.0]);
    assert_eq!(evaluated.jacobian(), [vec![0.0; 4]]);
}

#[test]
fn structural_arc_values_and_full_jacobians_match_the_fixed_go_baseline() {
    for fixture in include_str!("../fixtures/go-arc-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let input = match fields[0] {
            "translated" => [1.0, 2.0, 4.0, 6.0, 1.0, 12.0],
            "quarter" => [0.0, 0.0, 3.0, 0.0, 0.0, 4.0],
            "negative" => [0.0, 0.0, -3.0, -4.0, 0.0, -10.0],
            name => panic!("unexpected fixture {name}"),
        };
        let actual = evaluated(&input);
        assert!((actual.value() - fields[1].parse::<f64>().unwrap()).abs() < 1e-12);
        let expected: Vec<f64> = fields[2]
            .split(',')
            .map(|value| value.parse().unwrap())
            .collect();
        assert_eq!(actual.gradient().len(), expected.len());
        for (&actual, expected) in actual.gradient().iter().zip(expected) {
            assert!((actual - expected).abs() < 1e-12);
        }
    }
}
