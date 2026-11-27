// SPDX-License-Identifier: GPL-2.0-only
use super::{ColumnPivotedQr, DenseMatrix, LinearAlgebraError, finite, reflector};

impl ColumnPivotedQr {
    /// Solve the transposed system in least squares with minimum motion at the retained rank.
    /// Example: `let motion = qr.best_fit_transpose(&targets, 1e-9)?`.
    pub fn best_fit_transpose(
        &self,
        rhs: &[f64],
        rank_tolerance: f64,
    ) -> Result<Vec<f64>, LinearAlgebraError> {
        let targets = self.permuted_transpose_targets(rhs)?;
        let reduced = self.transposed_rank_factor()?;
        let coefficients =
            ColumnPivotedQr::factor(&reduced, rank_tolerance)?.least_squares(&targets)?;
        let mut result = vec![0.0; self.triangular.len()];
        result[..self.rank()].copy_from_slice(&coefficients);
        for (pivot, vector) in self.reflectors.iter().enumerate().rev() {
            reflector::apply(&mut result[pivot..], vector)?;
        }
        Ok(result)
    }

    fn permuted_transpose_targets(&self, rhs: &[f64]) -> Result<Vec<f64>, LinearAlgebraError> {
        if rhs.len() != self.permutation.len() {
            return Err(LinearAlgebraError(format!(
                "transpose right-hand side length {}: expected {} columns",
                rhs.len(),
                self.permutation.len()
            )));
        }
        self.permutation
            .iter()
            .map(|&column| finite(rhs[column] / self.scale, "transpose target"))
            .collect()
    }

    fn transposed_rank_factor(&self) -> Result<DenseMatrix, LinearAlgebraError> {
        let rows = (0..self.permutation.len())
            .map(|column| {
                (0..self.rank())
                    .map(|row| self.triangular[row][column])
                    .collect()
            })
            .collect();
        DenseMatrix::new(self.rank(), rows)
    }
    /// Solve the transposed system with minimum Euclidean norm when columns are independent.
    /// Example: `let motion = qr.minimum_norm_transpose(&targets)?;`.
    pub fn minimum_norm_transpose(&self, rhs: &[f64]) -> Result<Vec<f64>, LinearAlgebraError> {
        self.validate_transpose_rhs(rhs)?;
        let mut result = vec![0.0; self.triangular.len()];
        for pivot in 0..self.rank() {
            let target = finite(
                rhs[self.permutation[pivot]] / self.scale,
                "transpose target",
            )?;
            let known: f64 = (0..pivot)
                .map(|row| self.triangular[row][pivot] * result[row])
                .sum();
            result[pivot] = finite(
                (target - known) / self.triangular[pivot][pivot],
                "transpose solution",
            )?;
        }
        for (pivot, vector) in self.reflectors.iter().enumerate().rev() {
            reflector::apply(&mut result[pivot..], vector)?;
        }
        Ok(result)
    }

    fn validate_transpose_rhs(&self, rhs: &[f64]) -> Result<(), LinearAlgebraError> {
        if rhs.len() != self.permutation.len() {
            return Err(LinearAlgebraError(format!(
                "transpose right-hand side length {}: expected {} columns",
                rhs.len(),
                self.permutation.len()
            )));
        }
        if self.rank() != self.permutation.len() {
            return Err(LinearAlgebraError(format!(
                "matrix rank {}: expected {} independent columns for transpose solve",
                self.rank(),
                self.permutation.len()
            )));
        }
        Ok(())
    }
    /// Solve a full-column-rank least-squares system; singular/wide systems are refused.
    /// Example: `let solution = qr.least_squares(&rhs)?;`.
    pub fn least_squares(&self, rhs: &[f64]) -> Result<Vec<f64>, LinearAlgebraError> {
        if rhs.len() != self.triangular.len() {
            return Err(LinearAlgebraError(format!(
                "right-hand side length {}: expected {} rows",
                rhs.len(),
                self.triangular.len()
            )));
        }
        if self.rank() != self.permutation.len() {
            return Err(LinearAlgebraError(format!(
                "matrix rank {}: expected {} independent columns for least squares",
                self.rank(),
                self.permutation.len()
            )));
        }
        let mut projected: Vec<f64> = rhs
            .iter()
            .map(|entry| finite(entry / self.scale, "scaled right-hand side"))
            .collect::<Result<_, _>>()?;
        for (pivot, vector) in self.reflectors.iter().enumerate() {
            reflector::apply(&mut projected[pivot..], vector)?;
        }
        self.back_substitute(&projected)
    }

    fn back_substitute(&self, rhs: &[f64]) -> Result<Vec<f64>, LinearAlgebraError> {
        let mut ordered = vec![0.0; self.permutation.len()];
        for pivot in (0..ordered.len()).rev() {
            let tail: f64 = self.triangular[pivot]
                .iter()
                .zip(&ordered)
                .skip(pivot + 1)
                .map(|(entry, value)| entry * value)
                .sum();
            ordered[pivot] = finite(
                (rhs[pivot] - tail) / self.triangular[pivot][pivot],
                "triangular solution",
            )?;
        }
        let mut original = vec![0.0; ordered.len()];
        for (&column, value) in self.permutation.iter().zip(ordered) {
            original[column] = value;
        }
        Ok(original)
    }
}
