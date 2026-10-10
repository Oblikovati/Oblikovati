// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_solve::linalg::{ColumnPivotedQr, DenseMatrix};

const RANK_TOLERANCE: f64 = 1e-9;
fn matrix(rows: &[&[f64]]) -> DenseMatrix {
    DenseMatrix::new(
        rows.first().map_or(0, |row| row.len()),
        rows.iter().map(|row| row.to_vec()).collect(),
    )
    .unwrap()
}
fn close(actual: f64, expected: f64) {
    assert!(
        (actual - expected).abs() < 1e-9,
        "actual {actual:?}, expected {expected:?}"
    );
}

#[test]
fn square_go_fixture_solves_in_original_column_order_without_mutation() {
    let original = matrix(&[&[2.0, 1.0, 1.0], &[1.0, 3.0, 2.0], &[1.0, 0.0, 0.0]]);
    let saved = original.clone();
    let qr = ColumnPivotedQr::factor(&original, RANK_TOLERANCE).unwrap();
    assert_eq!(qr.rank(), 3);
    assert_eq!(qr.permutation()[0], 1);
    let rhs = [3.0, 1.0, 1.0];
    let solution = qr.least_squares(&rhs).unwrap();
    for (&actual, expected) in solution.iter().zip([1.0, -2.0, 3.0]) {
        close(actual, expected);
    }
    assert_eq!(original, saved);
    assert_eq!(original.columns(), 3);
    assert_eq!(rhs, [3.0, 1.0, 1.0]);
    assert_eq!(qr.least_squares(&rhs).unwrap(), solution);
}

#[test]
fn overdetermined_go_fixture_has_residual_orthogonal_to_every_column() {
    let original = matrix(&[&[1.0, 1.0], &[1.0, 2.0], &[1.0, 3.0], &[1.0, 4.0]]);
    let rhs = [6.0, 5.0, 7.0, 10.0];
    let solution = ColumnPivotedQr::factor(&original, RANK_TOLERANCE)
        .unwrap()
        .least_squares(&rhs)
        .unwrap();
    close(solution[0], 3.5);
    close(solution[1], 1.4);
    let residual: Vec<f64> = original
        .rows()
        .iter()
        .zip(rhs)
        .map(|(row, target)| row.iter().zip(&solution).map(|(a, x)| a * x).sum::<f64>() - target)
        .collect();
    for column in 0..2 {
        close(
            original
                .rows()
                .iter()
                .zip(&residual)
                .map(|(row, r)| row[column] * r)
                .sum(),
            0.0,
        );
    }
}

#[test]
fn exact_rank_fixture_and_scale_sweeps_preserve_numerical_rank() {
    for scale in [1e-200, 1.0, 1e200] {
        // Exact rank 2 independently certified by Jacobian matrix.rank.compute (#3701).
        let rows = [
            [1.0, 2.0, 3.0, 4.0],
            [2.0, 4.0, 6.0, 8.0],
            [0.0, 1.0, 1.0, 0.0],
        ];
        let original = DenseMatrix::new(
            4,
            rows.iter()
                .map(|row| row.iter().map(|entry| entry * scale).collect())
                .collect(),
        )
        .unwrap();
        let qr = ColumnPivotedQr::factor(&original, RANK_TOLERANCE).unwrap();
        assert_eq!(qr.rank(), 2);
        assert!(
            qr.least_squares(&[0.0; 3])
                .unwrap_err()
                .to_string()
                .contains("rank 2")
        );
        let full = matrix(&[&[2.0 * scale, 0.0], &[0.0, -3.0 * scale]]);
        let solution = ColumnPivotedQr::factor(&full, RANK_TOLERANCE)
            .unwrap()
            .least_squares(&[4.0 * scale, 6.0 * scale])
            .unwrap();
        close(solution[0], 2.0);
        close(solution[1], -2.0);
    }
}

#[test]
fn tolerance_controls_near_dependency_and_ties_are_repeatable() {
    let near = matrix(&[&[1.0, 1.0], &[1.0, 1.0 + 1e-12]]);
    assert_eq!(ColumnPivotedQr::factor(&near, 1e-9).unwrap().rank(), 1);
    assert_eq!(ColumnPivotedQr::factor(&near, 1e-14).unwrap().rank(), 2);
    let identity = matrix(&[&[1.0, 0.0], &[0.0, 1.0]]);
    let qr = ColumnPivotedQr::factor(&identity, RANK_TOLERANCE).unwrap();
    assert_eq!(qr.permutation(), [0, 1]);
    assert_eq!(
        qr.least_squares(&[f64::MAX, 1e-300]).unwrap(),
        [f64::MAX, 1e-300]
    );
}

#[test]
fn empty_axes_zero_columns_and_rank_deficiency_are_explicit() {
    let empty =
        ColumnPivotedQr::factor(&DenseMatrix::new(3, vec![]).unwrap(), RANK_TOLERANCE).unwrap();
    assert_eq!(empty.rank(), 0);
    assert!(empty.least_squares(&[]).is_err());
    let no_variables = ColumnPivotedQr::factor(
        &DenseMatrix::new(0, vec![vec![], vec![]]).unwrap(),
        RANK_TOLERANCE,
    )
    .unwrap();
    assert!(no_variables.least_squares(&[1.0, 2.0]).unwrap().is_empty());
    let zero =
        ColumnPivotedQr::factor(&matrix(&[&[0.0, 0.0], &[0.0, 0.0]]), RANK_TOLERANCE).unwrap();
    assert_eq!(zero.rank(), 0);
    let singular =
        ColumnPivotedQr::factor(&matrix(&[&[1.0, 0.0], &[2.0, 0.0]]), RANK_TOLERANCE).unwrap();
    assert_eq!(singular.rank(), 1);
    assert!(singular.least_squares(&[1.0, 2.0]).is_err());
}

#[test]
fn malformed_matrices_tolerances_rhs_and_nonfinite_solutions_are_refused() {
    assert!(
        DenseMatrix::new(2, vec![vec![1.0]])
            .unwrap_err()
            .to_string()
            .contains("expected 2")
    );
    for value in [f64::NAN, f64::INFINITY] {
        assert!(DenseMatrix::new(1, vec![vec![value]]).is_err());
    }
    let original = matrix(&[&[1.0]]);
    for tolerance in [0.0, -1.0, 1.0, f64::NAN, f64::INFINITY] {
        assert!(ColumnPivotedQr::factor(&original, tolerance).is_err());
    }
    let qr = ColumnPivotedQr::factor(&original, RANK_TOLERANCE).unwrap();
    assert!(
        qr.least_squares(&[])
            .unwrap_err()
            .to_string()
            .contains("expected 1 rows")
    );
    assert!(qr.least_squares(&[f64::NAN]).is_err());
    let tiny = ColumnPivotedQr::factor(&matrix(&[&[1e-300]]), RANK_TOLERANCE).unwrap();
    assert!(tiny.least_squares(&[1e300]).is_err());
}
