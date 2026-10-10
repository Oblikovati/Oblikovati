// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
use oblikovati_solve::residual::ResidualSource;
fn equation(name: &str, scale: f64) -> (PlanarEquation, Vec<f64>) {
    let (equation, values) = match name {
        "incidence" => (PlanarEquation::PointOnCircle, vec![4.0, 6.0, 1.0, 2.0, 3.0]),
        "concentric" => (PlanarEquation::Coincident, vec![1.0, 2.0, 4.0, 6.0]),
        "equal-radius" => (PlanarEquation::EqualRadius, vec![3.0, 2.0]),
        "radius" => (PlanarEquation::Radius(2.0 * scale), vec![3.0]),
        _ => panic!("unknown fixture {name}"),
    };
    (equation, values.iter().map(|value| value * scale).collect())
}
fn evaluate(name: &str, scale: f64) -> Vec<DualNumber> {
    let (equation, input) = equation(name, scale);
    let source = PreparedConstraint {
        projection: None,
        equation,
        variables: (0..input.len()).collect(),
    };
    let evaluated = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
    assert_eq!(evaluated.len(), source.equations());
    evaluated
}
fn numbers(text: &str) -> Vec<f64> {
    text.split(',')
        .map(|value| value.parse().unwrap())
        .collect()
}
fn close(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() < 1e-9,
        "actual {actual}, expected {expected}"
    );
}
#[test]
fn circular_equations_match_projected_go_partials_and_independent_derivatives() {
    for fixture in include_str!("../fixtures/go-circle-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let evaluated = evaluate(fields[0], 1.0);
        assert_eq!(evaluated.len(), numbers(fields[1]).len());
        assert_eq!(evaluated.len(), fields[2].split(';').count());
        for ((actual, expected), gradient) in evaluated
            .iter()
            .zip(numbers(fields[1]))
            .zip(fields[2].split(';'))
        {
            close(actual.value(), expected);
            assert_eq!(actual.gradient().len(), numbers(gradient).len());
            for (&actual, expected) in actual.gradient().iter().zip(numbers(gradient)) {
                close(actual, expected);
            }
        }
        independent_derivatives(fields[0], &evaluated);
    }
}
fn scalar(name: &str, values: &[f64]) -> Vec<f64> {
    match name {
        "incidence" => vec![(values[0] - values[2]).hypot(values[1] - values[3]) - values[4]],
        "concentric" => vec![values[0] - values[2], values[1] - values[3]],
        "equal-radius" => vec![values[0] - values[1]],
        "radius" => vec![values[0] - 2.0],
        _ => panic!("unknown scalar oracle {name}"),
    }
}
fn independent_derivatives(name: &str, evaluated: &[DualNumber]) {
    let (_, values) = equation(name, 1.0);
    let step = 1e-5;
    for column in 0..values.len() {
        let mut forward = values.clone();
        let mut backward = values.clone();
        forward[column] += step;
        backward[column] -= step;
        for (row, (forward, backward)) in scalar(name, &forward)
            .iter()
            .zip(scalar(name, &backward))
            .enumerate()
        {
            close(
                evaluated[row].gradient()[column],
                (forward - backward) / (2.0 * step),
            );
        }
    }
}
#[test]
fn circle_residuals_retain_dimension_and_gradient_across_extreme_units() {
    for name in ["incidence", "concentric", "equal-radius", "radius"] {
        let baseline = evaluate(name, 1.0);
        for scale in [1e-200, 1e-100, 1e100, 1e200] {
            for (actual, expected) in evaluate(name, scale).iter().zip(&baseline) {
                close(actual.value() / scale, expected.value());
                for (&actual, &expected) in actual.gradient().iter().zip(expected.gradient()) {
                    close(actual, expected);
                }
            }
        }
    }
}
#[test]
fn nonpositive_radius_trials_report_domains_and_do_not_publish_invalid_geometry() {
    for radius in [0.0, -1.0] {
        let number = DualNumber::variable(radius, 0, 1).unwrap();
        assert!(
            radius_target(&number, 2.0)
                .unwrap_err()
                .to_string()
                .contains("strictly positive")
        );
        let positive = DualNumber::constant(2.0).unwrap();
        assert!(equal_radius(&[number.clone(), positive.clone()]).is_err());
        assert!(equal_radius(&[positive, number.clone()]).is_err());
        let origin = DualNumber::constant(0.0).unwrap();
        assert!(
            point_on_circle(&[
                origin.clone(),
                origin.clone(),
                origin.clone(),
                origin,
                number
            ])
            .is_err()
        );
    }
}
