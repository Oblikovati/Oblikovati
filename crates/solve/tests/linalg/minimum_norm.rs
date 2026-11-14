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
