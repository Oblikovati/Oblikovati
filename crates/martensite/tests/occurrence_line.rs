// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn converts_a_translated_rotated_and_scaled_line() {
    let placement = Matrix {
        cells: [
            0., -2., 0., 10., 2., 0., 0., 20., 0., 0., 2., 30., 0., 0., 0., 1.,
        ],
    };
    let local = line_in_occurrence(&placement, [[8., 22., 36.], [0., 2., 0.]]).unwrap();
    assert_eq!(local, [[1., 1., 3.], [1., 0., 0.]]);
}

#[test]
fn refuses_singular_ill_conditioned_and_invalid_lines() {
    let mut cells = [
        1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 0., 0., 0., 0., 0., 1.,
    ];
    let line = [[0.; 3], [0., 0., 1.]];
    assert!(
        line_in_occurrence(&Matrix { cells }, line)
            .unwrap_err()
            .contains("invertible")
    );
    cells[10] = f64::EPSILON;
    assert!(
        line_in_occurrence(&Matrix { cells }, line)
            .unwrap_err()
            .contains("condition")
    );
    cells[10] = 1.;
    assert!(
        line_in_occurrence(&Matrix { cells }, [[0.; 3]; 2])
            .unwrap_err()
            .contains("nonzero")
    );
    assert!(line_in_occurrence(&Matrix { cells }, [[0.; 3], [f64::MAX; 3]]).is_err());
}
