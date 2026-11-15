// SPDX-License-Identifier: GPL-2.0-only
//! Approximate inverse candidates; only the subsequent interval check can accept them.
use super::krawczyk::RootRow;

pub(super) fn from_rows(rows: &[RootRow]) -> Result<Vec<Vec<f64>>, String> {
    let n = rows.len();
    if n == 0 || rows.iter().any(|row| row.jacobian.len() != n) {
        return Err(format!(
            "Jacobian with {n} rows: expected a nonempty square matrix"
        ));
    }
    let mut matrix = midpoint_matrix(rows)?;
    let mut inverse: Vec<Vec<f64>> = (0..n)
        .map(|i| (0..n).map(|j| f64::from(i == j)).collect())
        .collect();
    for column in 0..n {
        eliminate_column(&mut matrix, &mut inverse, column)?;
    }
    Ok(inverse)
}
fn midpoint_matrix(rows: &[RootRow]) -> Result<Vec<Vec<f64>>, String> {
    rows.iter()
        .map(|row| {
            row.jacobian
                .iter()
                .map(|&[lo, hi]| {
                    if !lo.is_finite() || !hi.is_finite() || lo > hi {
                        return Err(format!(
                            "Jacobian interval [{lo:?}, {hi:?}]: expected finite ordered bounds"
                        ));
                    }
                    Ok(lo / 2.0 + hi / 2.0)
                })
                .collect()
        })
        .collect()
}
fn eliminate_column(
    matrix: &mut [Vec<f64>],
    inverse: &mut [Vec<f64>],
    column: usize,
) -> Result<(), String> {
    let n = matrix.len();
    let pivot = (column..n)
        .max_by(|&a, &b| matrix[a][column].abs().total_cmp(&matrix[b][column].abs()))
        .unwrap();
    matrix.swap(column, pivot);
    inverse.swap(column, pivot);
    let divisor = matrix[column][column];
    if divisor == 0.0 || !divisor.is_finite() {
        return Err(format!(
            "Jacobian pivot {divisor:?} at column {column}: expected a finite nonzero pivot"
        ));
    }
    for j in 0..n {
        matrix[column][j] /= divisor;
        inverse[column][j] /= divisor;
    }
    for i in 0..n {
        if i == column {
            continue;
        }
        let factor = matrix[i][column];
        for j in 0..n {
            matrix[i][j] -= factor * matrix[column][j];
            inverse[i][j] -= factor * inverse[column][j];
        }
    }
    if matrix
        .iter()
        .chain(inverse.iter())
        .flatten()
        .any(|value| !value.is_finite())
    {
        return Err(format!(
            "Jacobian elimination at column {column}: expected finite arithmetic"
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sketch::conditional_root::krawczyk;
    use num_rational::BigRational;

    fn rows(matrix: &[Vec<f64>]) -> Vec<RootRow> {
        matrix
            .iter()
            .map(|row| RootRow {
                value: [0.0, 0.0],
                jacobian: row.iter().map(|v| [*v, *v]).collect(),
            })
            .collect()
    }
    fn exact(value: f64) -> BigRational {
        BigRational::from_float(value).unwrap()
    }

    #[test]
    fn pivoted_candidate_matches_independent_exact_inverse() {
        let jacobian = rows(&[vec![0., 2.], vec![4., 2.]]);
        let inverse = from_rows(&jacobian).unwrap();
        for (i, inverse_row) in inverse.iter().enumerate() {
            for j in 0..2 {
                let sum: BigRational = (0..2)
                    .map(|k| exact(inverse_row[k]) * exact(jacobian[k].jacobian[j][0]))
                    .sum();
                assert_eq!(sum, exact(f64::from(i == j)));
            }
        }
        assert!(
            krawczyk::evaluate(&[0., 0.], &[[-1., 1.]; 2], &jacobian, &jacobian, &inverse).is_ok()
        );
    }
    #[test]
    fn badly_scaled_invertible_system_uses_interval_acceptance() {
        let jacobian = rows(&[vec![1e-100, 0.], vec![0., 1e100]]);
        let inverse = from_rows(&jacobian).unwrap();
        let result =
            krawczyk::evaluate(&[0., 0.], &[[-1., 1.]; 2], &jacobian, &jacobian, &inverse).unwrap();
        assert!(result.contraction_upper < 1.0);
        for i in 0..2 {
            let error = exact(1.) - exact(inverse[i][i]) * exact(jacobian[i].jacobian[i][0]);
            let bound = exact(result.contraction_upper);
            assert!(-bound.clone() <= error && error <= bound);
        }
    }
    #[test]
    fn singular_and_uncertain_near_singular_systems_cannot_pass() {
        let singular = rows(&[vec![1., 1.], vec![1., 1.]]);
        assert!(from_rows(&singular).is_err());
        let almost = rows(&[vec![1., 1.], vec![1., f64::from_bits(1.0f64.to_bits() + 1)]]);
        let inverse = from_rows(&almost).unwrap();
        let mut uncertain = rows(&[vec![1., 1.], vec![1., 1.]]);
        uncertain[1].jacobian[1] = [1., f64::from_bits(1.0f64.to_bits() + 2)];
        assert!(
            krawczyk::evaluate(&[0., 0.], &[[-1., 1.]; 2], &uncertain, &almost, &inverse).is_err()
        );
    }
    #[test]
    fn adversarial_candidate_cannot_bypass_directed_check() {
        let identity = rows(&[vec![1., 0.], vec![0., 1.]]);
        let wrong = vec![vec![0., 0.], vec![0., 0.]];
        assert!(
            krawczyk::evaluate(&[0., 0.], &[[-1., 1.]; 2], &identity, &identity, &wrong).is_err()
        );
        let broad = vec![
            RootRow {
                value: [0., 0.],
                jacobian: vec![[0., 2.], [0., 0.]],
            },
            RootRow {
                value: [0., 0.],
                jacobian: vec![[0., 0.], [1., 1.]],
            },
        ];
        let inverse = from_rows(&broad).unwrap();
        assert!(
            krawczyk::evaluate(&[0., 0.], &[[-1., 1.]; 2], &broad, &identity, &inverse).is_err()
        );
    }
}
