// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_solve::{diagnostics::coordinate_freedoms, residual::ResidualEvaluation};
fn evaluation(rows: &[&[f64]], variables: usize) -> ResidualEvaluation {
    let source = source(rows);
    let columns: Vec<usize> = (0..variables).collect();
    let evaluated = evaluate_residuals(
        &vec![0.0; variables],
        &[ResidualBlock {
            source: &source,
            variables: &columns,
            equations: rows.len(),
        }],
    )
    .unwrap();
    assert_eq!(source.calls.get(), 1);
    evaluated
}
fn freedoms(rows: &[&[f64]], variables: usize) -> Vec<bool> {
    coordinate_freedoms(&evaluation(rows, variables), 1e-9, 1e-6).unwrap()
}
#[test]
fn fixed_free_and_coupled_coordinates_are_classified_from_the_row_space() {
    assert_eq!(freedoms(&[], 2), [true, true]);
    assert_eq!(freedoms(&[&[1.0, 0.0]], 2), [false, true]);
    assert_eq!(freedoms(&[&[0.0, 1.0]], 2), [true, false]);
    assert_eq!(freedoms(&[&[1.0, -1.0]], 2), [true, true]);
    assert_eq!(freedoms(&[&[1.0, 0.0], &[0.0, 1.0]], 2), [false, false]);
    assert_eq!(
        freedoms(&[&[1.0, 2.0, 3.0], &[1.0, 2.0, 1.0]], 3),
        [true, true, false]
    );
}
#[test]
fn dependent_zero_and_oppositely_scaled_rows_preserve_the_same_freedom() {
    for scale in [1e-300, 1.0, 1e300] {
        let rows = [
            [scale, -scale, 0.0],
            [-2.0 * scale, 2.0 * scale, 0.0],
            [0.0; 3],
            [0.0, 0.0, 1.0 / scale],
        ];
        let slices: Vec<&[f64]> = rows.iter().map(|row| row.as_slice()).collect();
        assert_eq!(freedoms(&slices, 3), [true, true, false]);
    }
}
#[test]
fn empty_axes_and_constant_constraints_do_not_invent_or_drop_variables() {
    assert!(freedoms(&[], 0).is_empty());
    assert!(freedoms(&[&[], &[]], 0).is_empty());
    assert_eq!(freedoms(&[&[0.0; 4]], 4), [true; 4]);
    assert_eq!(freedoms(&[], 4), [true; 4]);
}
#[test]
fn freedom_uses_an_explicit_squared_metric_threshold() {
    let evaluated = evaluation(&[&[1.0, 1.0]], 2);
    assert_eq!(
        coordinate_freedoms(&evaluated, 1e-9, 0.49).unwrap(),
        [true; 2]
    );
    assert_eq!(
        coordinate_freedoms(&evaluated, 1e-9, 0.51).unwrap(),
        [false; 2]
    );
    let saved = evaluated.clone();
    assert_eq!(
        coordinate_freedoms(&evaluated, 1e-9, 0.49).unwrap(),
        [true; 2]
    );
    assert_eq!(evaluated, saved);
}
#[test]
fn invalid_rank_and_freedom_thresholds_decline_even_for_empty_inputs() {
    let evaluated = evaluation(&[], 0);
    for value in [0.0, -1.0, 1.0, f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let cause = coordinate_freedoms(&evaluated, 1e-9, value)
            .unwrap_err()
            .to_string();
        assert!(cause.contains(&format!("{value:?}")));
        assert!(cause.contains("squared coordinate-freedom tolerance"));
        assert!(coordinate_freedoms(&evaluated, value, 1e-6).is_err());
    }
}
