// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn equation(name: &str) -> (PlanarEquation, Vec<f64>) {
    match name {
        "coincident" => (PlanarEquation::Coincident, vec![3.0, 4.0, 0.0, 0.0]),
        "horizontal" => (PlanarEquation::Coordinate(1), vec![3.0, 4.0, 0.0, 0.0]),
        "vertical" => (PlanarEquation::Coordinate(0), vec![3.0, 4.0, 0.0, 0.0]),
        "midpoint" => (PlanarEquation::Midpoint, vec![1.0, 2.0, 0.0, 0.0, 4.0, 6.0]),
        "fix" => (PlanarEquation::Fix([3.0, 4.0]), vec![5.0, 7.0]),
        "aligned" => (
            PlanarEquation::Distance {
                measure: DistanceMeasure::Aligned,
                target: 2.0,
            },
            vec![3.0, 4.0, 0.0, 0.0],
        ),
        "x-distance" => (
            PlanarEquation::Distance {
                measure: DistanceMeasure::Horizontal,
                target: 2.0,
            },
            vec![3.0, 4.0, 0.0, 0.0],
        ),
        "y-distance" => (
            PlanarEquation::Distance {
                measure: DistanceMeasure::Vertical,
                target: 2.0,
            },
            vec![3.0, 4.0, 0.0, 0.0],
        ),
        _ => panic!("unknown fixture {name}"),
    }
}
fn values(text: &str) -> Vec<f64> {
    text.split(',')
        .map(|value| value.parse().unwrap())
        .collect()
}
fn assert_near(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() < 1e-9,
        "actual {actual}, expected {expected}"
    );
}
#[test]
fn every_planar_formula_matches_go_values_and_jacobians() {
    for fixture in include_str!("../fixtures/go-planar-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let (equation, input) = equation(fields[0]);
        let source = PreparedConstraint {
            equation,
            variables: (0..input.len()).collect(),
        };
        let evaluated = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
        assert_eq!(evaluated.len(), source.equations());
        assert_eq!(evaluated.len(), values(fields[1]).len());
        assert_eq!(evaluated.len(), fields[2].split(';').count());
        for ((actual, expected), partials) in evaluated
            .iter()
            .zip(values(fields[1]))
            .zip(fields[2].split(';'))
        {
            assert_near(actual.value(), expected);
            assert_eq!(actual.gradient().len(), values(partials).len());
            for (&actual, expected) in actual.gradient().iter().zip(values(partials)) {
                assert_near(actual, expected);
            }
        }
        check_independent_partials(fields[0], &input, &evaluated);
    }
}
fn scalar_residuals(name: &str, input: &[f64]) -> Vec<f64> {
    match name {
        "coincident" => vec![input[0] - input[2], input[1] - input[3]],
        "horizontal" => vec![input[1] - input[3]],
        "vertical" => vec![input[0] - input[2]],
        "midpoint" => vec![
            input[0] - (input[2] + input[4]) / 2.0,
            input[1] - (input[3] + input[5]) / 2.0,
        ],
        "fix" => vec![input[0] - 3.0, input[1] - 4.0],
        "aligned" => vec![(input[0] - input[2]).hypot(input[1] - input[3]) - 2.0],
        "x-distance" => vec![(input[0] - input[2]).abs() - 2.0],
        "y-distance" => vec![(input[1] - input[3]).abs() - 2.0],
        _ => panic!("unknown scalar oracle {name}"),
    }
}
fn check_independent_partials(name: &str, input: &[f64], evaluated: &[DualNumber]) {
    let step = 1e-5;
    for column in 0..input.len() {
        let mut forward = input.to_vec();
        let mut backward = input.to_vec();
        forward[column] += step;
        backward[column] -= step;
        let difference = scalar_residuals(name, &forward)
            .into_iter()
            .zip(scalar_residuals(name, &backward));
        for (row, (forward, backward)) in difference.enumerate() {
            assert_near(
                evaluated[row].gradient()[column],
                (forward - backward) / (2.0 * step),
            );
        }
    }
}
#[test]
fn midpoint_avoids_avoidable_overflow_and_bad_local_shapes_are_refused() {
    let source = PreparedConstraint {
        equation: PlanarEquation::Midpoint,
        variables: vec![0; 6],
    };
    let input = [f64::MAX, 0.0, f64::MAX, 0.0, f64::MAX, 0.0];
    let evaluated = source.evaluate(&DualNumber::seed(&input).unwrap()).unwrap();
    assert_eq!(evaluated[0].value(), 0.0);
    assert_eq!(evaluated[0].gradient(), [1.0, 0.0, -0.5, 0.0, -0.5, 0.0]);
    assert!(
        source
            .evaluate(&[])
            .unwrap_err()
            .to_string()
            .contains("received 0 variables: expected 6")
    );
}
