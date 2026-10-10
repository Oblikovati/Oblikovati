// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
use oblikovati_solve::residual::ResidualSource;
fn equation(name: &str, scale: f64) -> (PlanarEquation, Vec<f64>) {
    let (equation, input) = match name {
        "left" => (
            PlanarEquation::LineTangent(LineContactSide::Left),
            vec![0.0, 0.0, 4.0, 0.0, 3.0, 2.0, 1.0],
        ),
        "right" => (
            PlanarEquation::LineTangent(LineContactSide::Right),
            vec![0.0, 0.0, 4.0, 0.0, 3.0, -2.0, 1.0],
        ),
        "external" => (
            PlanarEquation::CircleTangent(CircleContact::External),
            vec![0.0, 0.0, 5.0, 1.0, 3.0, 2.0],
        ),
        "first-outer" => (
            PlanarEquation::CircleTangent(CircleContact::FirstContainsSecond),
            vec![0.0, 0.0, 2.0, 1.0, 5.0, 2.0],
        ),
        "second-outer" => (
            PlanarEquation::CircleTangent(CircleContact::SecondContainsFirst),
            vec![0.0, 0.0, 2.0, 1.0, 2.0, 5.0],
        ),
        _ => panic!("unknown fixture {name}"),
    };
    (equation, input.iter().map(|value| value * scale).collect())
}
fn evaluated(name: &str, scale: f64) -> DualNumber {
    let (equation, input) = equation(name, scale);
    let source = PreparedConstraint {
        equation,
        variables: (0..input.len()).collect(),
    };
    let result = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
    assert_eq!(result.len(), 1);
    assert_eq!(source.equations(), 1);
    result[0].clone()
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
fn tangent_values_and_partials_match_go_on_each_recorded_contact_branch() {
    for fixture in include_str!("../fixtures/go-tangent-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let actual = evaluated(fields[0], 1.0);
        close(actual.value(), fields[1].parse().unwrap());
        assert_eq!(actual.gradient().len(), numbers(fields[2]).len());
        for (&actual, expected) in actual.gradient().iter().zip(numbers(fields[2])) {
            close(actual, expected);
        }
        independent_derivatives(fields[0], &actual);
    }
}
fn scalar(name: &str, input: &[f64]) -> f64 {
    if name == "left" || name == "right" {
        let dx = input[2] - input[0];
        let dy = input[3] - input[1];
        let distance = (dx * (input[5] - input[1]) - dy * (input[4] - input[0])) / dx.hypot(dy);
        return (if name == "left" { distance } else { -distance }) - input[6];
    }
    let distance = (input[0] - input[2]).hypot(input[1] - input[3]);
    let target = match name {
        "external" => input[4] + input[5],
        "first-outer" => input[4] - input[5],
        "second-outer" => input[5] - input[4],
        _ => panic!("unknown scalar {name}"),
    };
    distance - target
}
fn independent_derivatives(name: &str, evaluated: &DualNumber) {
    let (_, input) = equation(name, 1.0);
    let step = 1e-5;
    for column in 0..input.len() {
        let mut forward = input.clone();
        let mut backward = input.clone();
        forward[column] += step;
        backward[column] -= step;
        close(
            evaluated.gradient()[column],
            (scalar(name, &forward) - scalar(name, &backward)) / (2.0 * step),
        );
    }
}
#[test]
fn tangent_residuals_keep_length_units_and_partials_across_coordinate_scales() {
    for name in ["left", "right", "external", "first-outer", "second-outer"] {
        let baseline = evaluated(name, 1.0);
        for scale in [1e-200, 1e-100, 1e100, 1e200] {
            let actual = evaluated(name, scale);
            close(actual.value() / scale, baseline.value());
            for (&actual, &expected) in actual.gradient().iter().zip(baseline.gradient()) {
                close(actual, expected);
            }
        }
    }
}
#[test]
fn undefined_contact_directions_and_invalid_containment_are_domain_errors() {
    let coincident = DualNumber::seed(&[0.0, 0.0, 0.0, 0.0, 3.0, 2.0]).unwrap();
    assert!(
        circle_tangent(&coincident, CircleContact::External)
            .unwrap_err()
            .to_string()
            .contains("distinct centers")
    );
    let first = DualNumber::constant(2.0).unwrap();
    let second = DualNumber::constant(3.0).unwrap();
    assert!(contact_distance(&first, &second, CircleContact::FirstContainsSecond).is_err());
    assert!(contact_distance(&first, &first, CircleContact::FirstContainsSecond).is_err());
    let line = DualNumber::seed(&[0.0, 0.0, 0.0, 0.0, 1.0, 2.0, 1.0]).unwrap();
    assert!(line_tangent(&line, LineContactSide::Left).is_err());
}
