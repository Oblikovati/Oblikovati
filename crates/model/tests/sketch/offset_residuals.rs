// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::residual::{PlanarEquation, PreparedConstraint};
use oblikovati_solve::residual::ResidualSource;
fn source(target: f64, side: LineContactSide) -> PreparedConstraint {
    PreparedConstraint {
        projection: None,
        variables: (0..6).collect(),
        equation: PlanarEquation::Offset { target, side },
    }
}
fn scalar(v: &[f64; 6], target: f64, sign: f64) -> f64 {
    let dx = v[4] - v[2];
    let dy = v[5] - v[3];
    sign * (dx * (v[1] - v[3]) - dy * (v[0] - v[2])) / dx.hypot(dy) - target
}
#[test]
fn offset_rows_match_independent_metric_values_all_partials_and_extreme_units() {
    for side in [LineContactSide::Left, LineContactSide::Right] {
        let sign = if side == LineContactSide::Left {
            1.0
        } else {
            -1.0
        };
        let input = [2.0, sign * 6.0, 1.0, 2.0, 5.0, 3.0];
        let reference = source(2.0, side)
            .evaluate(&DualNumber::seed(&input).unwrap())
            .unwrap();
        assert!((reference[0].value() - scalar(&input, 2.0, sign)).abs() < 1e-12);
        for column in 0..6 {
            let mut left = input;
            let mut right = input;
            left[column] -= 1e-5;
            right[column] += 1e-5;
            let expected = (scalar(&right, 2.0, sign) - scalar(&left, 2.0, sign)) / 2e-5;
            assert!((reference[0].gradient()[column] - expected).abs() < 1e-8);
        }
        for scale in [1e-200, 1e-100, 1.0, 1e100, 1e200] {
            let actual = source(2.0 * scale, side)
                .evaluate(&DualNumber::seed(&input.map(|v| v * scale)).unwrap())
                .unwrap();
            assert!((actual[0].value() / scale - reference[0].value()).abs() < 1e-12);
            for (actual, expected) in actual[0].gradient().iter().zip(reference[0].gradient()) {
                assert!((actual - expected).abs() < 1e-12);
            }
        }
    }
}
fn numbers(text: &str) -> Vec<f64> {
    text.split(',')
        .map(|value| value.parse().unwrap())
        .collect()
}
#[test]
fn linear_and_diameter_offsets_match_go_residuals_and_jacobians_on_both_sides() {
    for fixture in include_str!("../fixtures/go-offset-residuals.tsv").lines() {
        let fields: Vec<&str> = fixture.split('\t').collect();
        let branch: Vec<&str> = fields[0].split(':').collect();
        let side = if branch[0] == "1" {
            LineContactSide::Left
        } else {
            LineContactSide::Right
        };
        let factor = if branch[1] == "true" { 2.0 } else { 1.0 };
        let actual = source(2.0 / factor, side)
            .evaluate(&DualNumber::seed(&numbers(fields[1])).unwrap())
            .unwrap();
        assert!((actual[0].value() * factor - numbers(fields[2])[0]).abs() < 1e-12);
        assert_eq!(actual[0].gradient().len(), numbers(fields[3]).len());
        for (actual, expected) in actual[0].gradient().iter().zip(numbers(fields[3])) {
            assert!((actual * factor - expected).abs() < 1e-12);
        }
    }
}
#[test]
fn zero_offset_keeps_its_derivative_and_collapsed_lines_are_named_declines() {
    let row = source(0.0, LineContactSide::Right)
        .evaluate(&DualNumber::seed(&[2.0, 0.0, 0.0, 0.0, 4.0, 0.0]).unwrap())
        .unwrap();
    assert_eq!(row[0].value(), 0.0);
    assert_eq!(row[0].gradient()[1], -1.0);
    assert_eq!(row[0].gradient()[3], 0.5);
    assert_eq!(row[0].gradient()[5], 0.5);
    let error = source(2.0, LineContactSide::Left)
        .evaluate(&DualNumber::seed(&[2.0, 3.0, 1.0, 2.0, 1.0, 2.0]).unwrap())
        .unwrap_err();
    assert!(error.to_string().contains("nonzero direction"));
    assert!(source(2.0, LineContactSide::Left).evaluate(&[]).is_err());
}
