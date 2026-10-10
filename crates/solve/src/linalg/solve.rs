// SPDX-License-Identifier: GPL-2.0-only
use super::{ColumnPivotedQr, LinearAlgebraError, finite, reflector};

impl ColumnPivotedQr {
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
