// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
fn relation(name: &str) -> PlanarEquation {
    match name {
        "parallel" => PlanarEquation::Lines {
            relation: LineRelation::Parallel,
            shared_endpoints: 0,
        },
        "perpendicular" => PlanarEquation::Lines {
            relation: LineRelation::Perpendicular,
            shared_endpoints: 0,
        },
        "collinear" => PlanarEquation::Lines {
            relation: LineRelation::Collinear,
            shared_endpoints: 0,
        },
        "equal" => PlanarEquation::Lines {
            relation: LineRelation::EqualLength,
            shared_endpoints: 0,
        },
        "on-line" => PlanarEquation::PointOnLine,
        _ => panic!("unknown fixture {name}"),
    }
}
fn input(name: &str, scale: f64) -> Vec<f64> {
    let coordinates: &[f64] = if name == "on-line" {
        &[2.0, 3.0, 0.0, 0.0, 3.0, 4.0]
    } else {
        &[0.0, 0.0, 3.0, 4.0, 1.0, 2.0, 5.0, 1.0]
    };
    coordinates.iter().map(|value| value * scale).collect()
}
fn evaluate(name: &str, input: &[f64]) -> Vec<DualNumber> {
    let source = PreparedConstraint {
        projection: None,
        equation: relation(name),
        variables: (0..input.len()).collect(),
    };
    use oblikovati_solve::residual::ResidualSource;
    let result = source.evaluate(&DualNumber::seed(input).unwrap()).unwrap();
    assert_eq!(result.len(), source.equations());
    result
}
fn numbers(text: &str) -> Vec<f64> {
    text.split(',')
        .map(|value| value.parse().unwrap())
        .collect()
}
fn close(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() <= 2e-8 * expected.abs().max(1.0),
        "actual {actual}, expected {expected}"
    );
}
#[test]
fn line_values_and_derivatives_match_go_and_independent_scalar_formulas() {
    for fixture in include_str!("../fixtures/go-line-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let name = fields[0];
        let input = input(name, 1.0);
        let evaluated = evaluate(name, &input);
        assert_eq!(evaluated.len(), numbers(fields[1]).len());
        assert_eq!(evaluated.len(), fields[2].split(';').count());
        for ((row, expected), partials) in evaluated
            .iter()
            .zip(numbers(fields[1]))
            .zip(fields[2].split(';'))
        {
            close(row.value(), expected);
            assert_eq!(row.gradient().len(), numbers(partials).len());
            for (&actual, expected) in row.gradient().iter().zip(numbers(partials)) {
                close(actual, expected);
            }
        }
        independent_derivatives(name, &input, &evaluated);
    }
}
fn scalar(name: &str, input: &[f64]) -> Vec<f64> {
    if name == "on-line" {
        let (dx, dy) = (input[4] - input[2], input[5] - input[3]);
        return vec![(dx * (input[1] - input[3]) - dy * (input[0] - input[2])) / dx.hypot(dy)];
    }
    let (ax, ay) = (input[2] - input[0], input[3] - input[1]);
    let (bx, by) = (input[6] - input[4], input[7] - input[5]);
    let denominator = ax.hypot(ay) * bx.hypot(by);
    match name {
        "parallel" => vec![(ax * by - ay * bx) / denominator],
        "perpendicular" => vec![(ax * bx + ay * by) / denominator],
        "collinear" => vec![
            (ax * by - ay * bx) / denominator,
            (ax * (input[5] - input[1]) - ay * (input[4] - input[0])) / ax.hypot(ay),
        ],
        "equal" => vec![ax.hypot(ay) - bx.hypot(by)],
        _ => panic!("unknown scalar {name}"),
    }
}
fn independent_derivatives(name: &str, input: &[f64], evaluated: &[DualNumber]) {
    let step = 1e-5;
    for column in 0..input.len() {
        let mut forward = input.to_vec();
        let mut backward = input.to_vec();
        forward[column] += step;
        backward[column] -= step;
        for (row, (forward, backward)) in scalar(name, &forward)
            .into_iter()
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
fn normalized_values_and_derivatives_survive_extreme_coordinate_units() {
    for name in ["parallel", "perpendicular", "collinear", "equal", "on-line"] {
        let baseline = evaluate(name, &input(name, 1.0));
        for scale in [1e-200, 1e-100, 1e100, 1e200] {
            let evaluated = evaluate(name, &input(name, scale));
            for (index, (row, original)) in evaluated.iter().zip(&baseline).enumerate() {
                let is_length =
                    name == "equal" || name == "on-line" || (name == "collinear" && index == 1);
                let output_scale = if is_length { scale } else { 1.0 };
                close(row.value() / output_scale, original.value());
                for (&actual, &expected) in row.gradient().iter().zip(original.gradient()) {
                    close(actual * (scale / output_scale), expected);
                }
            }
        }
    }
}
#[test]
fn zero_direction_is_a_domain_error_instead_of_a_satisfied_angle() {
    let zero = DualVector::new([
        DualNumber::constant(0.0).unwrap(),
        DualNumber::constant(0.0).unwrap(),
    ])
    .unwrap();
    assert!(
        unit_direction(&zero)
            .unwrap_err()
            .to_string()
            .contains("[0.0, 0.0]")
    );
    let coordinates = DualNumber::seed(&[0.0; 8]).unwrap();
    for relation in [
        LineRelation::Parallel,
        LineRelation::Perpendicular,
        LineRelation::Collinear,
    ] {
        assert!(line_relation(&coordinates, relation, 0).is_err());
    }
    assert_eq!(
        line_relation(&coordinates, LineRelation::EqualLength, 0).unwrap()[0].value(),
        0.0
    );
    assert!(point_on_line(&coordinates[..6]).is_err());
}

fn original_disjoint_collinear(variables: &[DualNumber]) -> Vec<DualNumber> {
    let first = point(variables, 2)
        .unwrap()
        .minus(&point(variables, 0).unwrap())
        .unwrap();
    let second = point(variables, 6)
        .unwrap()
        .minus(&point(variables, 4).unwrap())
        .unwrap();
    let (first, second) = oriented_directions(&first, &second).unwrap();
    let offset = point(variables, 4)
        .unwrap()
        .minus(&point(variables, 0).unwrap())
        .unwrap();
    vec![first.cross(&second).unwrap(), first.cross(&offset).unwrap()]
}
#[test]
fn disjoint_collinear_values_and_derivatives_keep_the_original_bits() {
    for scale in [0.01, 1., 100.] {
        let seeded = DualNumber::seed(&input("collinear", scale)).unwrap();
        let current = line_relation(&seeded, LineRelation::Collinear, 0).unwrap();
        let original = original_disjoint_collinear(&seeded);
        for (current, original) in current.iter().zip(original) {
            assert_eq!(current.value().to_bits(), original.value().to_bits());
            let actual: Vec<u64> = current
                .gradient()
                .iter()
                .map(|value| value.to_bits())
                .collect();
            let expected: Vec<u64> = original
                .gradient()
                .iter()
                .map(|value| value.to_bits())
                .collect();
            assert_eq!(actual, expected);
        }
    }
}
