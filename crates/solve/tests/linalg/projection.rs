// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn projection(rows: &[&[f64]]) -> Vec<f64> {
    ColumnPivotedQr::factor(&matrix(rows), RANK_TOLERANCE)
        .unwrap()
        .coordinate_projection_residuals()
        .unwrap()
}
#[test]
fn transpose_preserves_values_and_both_empty_axis_shapes() {
    let original = matrix(&[&[1.0, 2.0, 3.0], &[4.0, 5.0, 6.0]]);
    assert_eq!(
        original.transpose(),
        matrix(&[&[1.0, 4.0], &[2.0, 5.0], &[3.0, 6.0]])
    );
    assert_eq!(original.transpose().transpose(), original);
    let empty_rows = DenseMatrix::new(3, vec![]).unwrap();
    assert_eq!(
        empty_rows.transpose(),
        DenseMatrix::new(0, vec![vec![]; 3]).unwrap()
    );
    assert_eq!(empty_rows.transpose().transpose(), empty_rows);
    let empty_columns = DenseMatrix::new(0, vec![vec![]; 2]).unwrap();
    assert_eq!(
        empty_columns.transpose(),
        DenseMatrix::new(2, vec![]).unwrap()
    );
    assert_eq!(empty_columns.transpose().transpose(), empty_columns);
}
#[test]
fn two_reflectors_match_the_independent_exact_projection_fixture() {
    // #3766: Gram [[14,8],[8,6]] inverse independently certified by Jacobian.
    // Its inverse is [[3/10,-2/5],[-2/5,7/10]]; I-A(A^T A)^-1 A^T has this diagonal.
    for rows in [
        [[1.0, 1.0], [2.0, 2.0], [3.0, 1.0]],
        [[1.0, 1.0], [2.0, 2.0], [1.0, 3.0]],
    ] {
        let original = DenseMatrix::new(2, rows.iter().map(|row| row.to_vec()).collect()).unwrap();
        let qr = ColumnPivotedQr::factor(&original, RANK_TOLERANCE).unwrap();
        assert_eq!(qr.rank(), 2);
        for (actual, expected) in qr
            .coordinate_projection_residuals()
            .unwrap()
            .into_iter()
            .zip([0.8, 0.2, 0.0])
        {
            close(actual, expected);
        }
    }
}
#[test]
fn coupled_coordinates_retain_freedom_even_when_a_pivot_uses_their_column() {
    assert_eq!(projection(&[&[1.0], &[0.0]]), [0.0, 1.0]);
    let coupled = projection(&[&[1.0], &[-1.0]]);
    close(coupled[0], 0.5);
    close(coupled[1], 0.5);
    assert_eq!(projection(&[&[0.0, 0.0], &[0.0, 0.0]]), [1.0, 1.0]);
    assert_eq!(projection(&[&[1.0, 0.0], &[0.0, 1.0]]), [0.0, 0.0]);
}
#[test]
fn empty_and_wide_factorizations_preserve_the_coordinate_universe() {
    for (columns, rows, expected) in [
        (2, vec![], vec![]),
        (0, vec![vec![]; 3], vec![1.0; 3]),
        (0, vec![], vec![]),
        (3, vec![vec![1.0, 0.0, 2.0]], vec![0.0]),
    ] {
        let original = DenseMatrix::new(columns, rows).unwrap();
        let qr = ColumnPivotedQr::factor(&original, RANK_TOLERANCE).unwrap();
        assert_eq!(qr.coordinate_projection_residuals().unwrap(), expected);
    }
}
#[test]
fn column_scales_signs_redundancy_and_pivoting_do_not_change_the_subspace() {
    for scale in [1e-200, 1.0, 1e200] {
        let rows = [
            [scale, -scale, 10.0 * scale],
            [2.0 * scale, -2.0 * scale, 20.0 * scale],
            [3.0 * scale, -3.0 * scale, 30.0 * scale],
        ];
        let matrix = DenseMatrix::new(3, rows.iter().map(|row| row.to_vec()).collect()).unwrap();
        let before = matrix.clone();
        let qr = ColumnPivotedQr::factor(&matrix, RANK_TOLERANCE).unwrap();
        assert_eq!(qr.rank(), 1);
        assert_eq!(qr.permutation()[0], 2);
        assert_rank_one_coordinate_residuals(&qr);
        assert_eq!(matrix, before);
    }
}
fn assert_rank_one_coordinate_residuals(qr: &ColumnPivotedQr) {
    let residuals = qr.coordinate_projection_residuals().unwrap();
    for (actual, expected) in residuals
        .iter()
        .copied()
        .zip([13.0 / 14.0, 10.0 / 14.0, 5.0 / 14.0])
    {
        close(actual, expected);
    }
    assert_eq!(qr.coordinate_projection_residuals().unwrap(), residuals);
}
#[test]
fn rank_threshold_controls_the_projection_without_a_second_rank_rule() {
    let nearly_dependent = matrix(&[&[1.0, 1.0], &[0.0, 1e-12]]);
    let truncated = ColumnPivotedQr::factor(&nearly_dependent, 1e-9).unwrap();
    assert_eq!(truncated.rank(), 1);
    close(truncated.coordinate_projection_residuals().unwrap()[1], 1.0);
    let retained = ColumnPivotedQr::factor(&nearly_dependent, 1e-14).unwrap();
    assert_eq!(retained.rank(), 2);
    assert_eq!(
        retained.coordinate_projection_residuals().unwrap(),
        [0.0, 0.0]
    );
}
