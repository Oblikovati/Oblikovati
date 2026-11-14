// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn transpose_solve_uses_the_minimum_norm_and_respects_column_pivots() {
    let columns = matrix(&[&[1.0, 0.0], &[1.0, 3.0], &[0.0, 3.0]]);
    let qr = ColumnPivotedQr::factor(&columns, RANK_TOLERANCE).unwrap();
    assert_eq!(qr.permutation()[0], 1);
    let result = qr.minimum_norm_transpose(&[2.0, 3.0]).unwrap();
    for (actual, expected) in result.iter().zip([1.0, 1.0, 0.0]) {
        close(*actual, expected);
    }
}

#[test]
fn transpose_solve_rejects_singular_and_invalid_inputs() {
    let qr = ColumnPivotedQr::factor(&matrix(&[&[1.0, 2.0], &[2.0, 4.0]]), RANK_TOLERANCE).unwrap();
    assert!(qr.minimum_norm_transpose(&[1.0, 2.0]).is_err());
    let full = ColumnPivotedQr::factor(&matrix(&[&[1.0], &[1.0]]), RANK_TOLERANCE).unwrap();
    assert!(full.minimum_norm_transpose(&[]).is_err());
    assert!(full.minimum_norm_transpose(&[f64::NAN]).is_err());
}

#[test]
fn transpose_best_fit_handles_dependent_and_conflicting_targets() {
    let columns = matrix(&[&[1.0, 1.0], &[1.0, 1.0]]);
    let qr = ColumnPivotedQr::factor(&columns, RANK_TOLERANCE).unwrap();
    assert_eq!(qr.rank(), 1);
    for (rhs, expected) in [([2.0, 2.0], 1.0), ([2.0, 4.0], 1.5)] {
        let result = qr.best_fit_transpose(&rhs, RANK_TOLERANCE).unwrap();
        for value in result {
            close(value, expected);
        }
    }
}

#[test]
fn transpose_best_fit_keeps_target_order_when_columns_are_pivoted() {
    let columns = matrix(&[&[1.0, 2.0], &[1.0, 2.0]]);
    let qr = ColumnPivotedQr::factor(&columns, RANK_TOLERANCE).unwrap();
    assert_eq!(qr.permutation()[0], 1);
    let result = qr.best_fit_transpose(&[2.0, 6.0], RANK_TOLERANCE).unwrap();
    for value in result {
        close(value, 1.4);
    }
}

#[test]
fn transpose_best_fit_of_zero_matrix_is_zero_and_validates_inputs() {
    let qr = ColumnPivotedQr::factor(&matrix(&[&[0.0, 0.0], &[0.0, 0.0]]), RANK_TOLERANCE).unwrap();
    assert_eq!(
        qr.best_fit_transpose(&[2.0, 4.0], RANK_TOLERANCE).unwrap(),
        [0.0, 0.0]
    );
    assert!(qr.best_fit_transpose(&[1.0], RANK_TOLERANCE).is_err());
    assert!(
        qr.best_fit_transpose(&[f64::NAN, 0.0], RANK_TOLERANCE)
            .is_err()
    );
}
