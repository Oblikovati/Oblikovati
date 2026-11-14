// SPDX-License-Identifier: GPL-2.0-only
use super::{ColumnPivotedQr, LinearAlgebraError, finite, reflector};
impl ColumnPivotedQr {
    /// Project a vector orthogonal to the factored column space.
    /// Example: `let free_motion = qr.project_orthogonal(&motion)?;`.
    pub fn project_orthogonal(&self, values: &[f64]) -> Result<Vec<f64>, LinearAlgebraError> {
        if values.len() != self.triangular.len() {
            return Err(LinearAlgebraError(format!(
                "projection vector length {}: expected {} rows",
                values.len(),
                self.triangular.len()
            )));
        }
        let mut projected = values
            .iter()
            .map(|&value| finite(value, "projection vector"))
            .collect::<Result<Vec<_>, _>>()?;
        for (pivot, vector) in self.reflectors.iter().enumerate() {
            reflector::apply(&mut projected[pivot..], vector)?;
        }
        projected[..self.rank()].fill(0.0);
        for (pivot, vector) in self.reflectors.iter().enumerate().rev() {
            reflector::apply(&mut projected[pivot..], vector)?;
        }
        Ok(projected)
    }
    /// Measure each unit coordinate's squared distance from the factored column space.
    /// Example: `let freedom = qr.coordinate_projection_residuals()?;`.
    pub fn coordinate_projection_residuals(&self) -> Result<Vec<f64>, LinearAlgebraError> {
        (0..self.triangular.len())
            .map(|coordinate| self.coordinate_tail_squared(coordinate))
            .collect()
    }
    fn coordinate_tail_squared(&self, coordinate: usize) -> Result<f64, LinearAlgebraError> {
        let mut projected = vec![0.0; self.triangular.len()];
        projected[coordinate] = 1.0;
        // #3766: forward reflectors apply Q^T; the tail is orthogonal to every retained column.
        for (pivot, vector) in self.reflectors.iter().enumerate() {
            reflector::apply(&mut projected[pivot..], vector)?;
        }
        let norm = projected
            .iter()
            .skip(self.rank())
            .fold(0.0_f64, |norm, entry| norm.hypot(*entry));
        finite(norm * norm, "coordinate projection squared residual")
    }
}
