// SPDX-License-Identifier: GPL-2.0-only
use super::{ColumnPivotedQr, DenseMatrix, LinearAlgebraError, finite, reflector};
impl ColumnPivotedQr {
    /// Find the least motion that satisfies preference rows without moving along constraint rows.
    /// Example: `qr.minimum_norm_free_motion(&preference_transpose, &targets, 1e-9)?`.
    pub fn minimum_norm_free_motion(
        &self,
        preference: &DenseMatrix,
        targets: &[f64],
        rank_tolerance: f64,
    ) -> Result<Vec<f64>, LinearAlgebraError> {
        self.try_minimum_norm_free_motion(preference, targets, rank_tolerance)?
            .ok_or_else(|| {
                LinearAlgebraError(format!(
                    "preference rank deficient: expected {} independent reachable targets",
                    preference.columns()
                ))
            })
    }

    pub(crate) fn try_minimum_norm_free_motion(
        &self,
        preference: &DenseMatrix,
        targets: &[f64],
        rank_tolerance: f64,
    ) -> Result<Option<Vec<f64>>, LinearAlgebraError> {
        validate_targets(preference, targets)?;
        let factor = self.projected_preference_factor(preference, rank_tolerance)?;
        if factor.rank() < preference.columns() {
            return Ok(None);
        }
        factor.minimum_norm_transpose(targets).map(Some)
    }

    pub(crate) fn best_fit_free_motion(
        &self,
        preference: &DenseMatrix,
        targets: &[f64],
        rank_tolerance: f64,
    ) -> Result<Vec<f64>, LinearAlgebraError> {
        validate_targets(preference, targets)?;
        self.projected_preference_factor(preference, rank_tolerance)?
            .best_fit_transpose(targets, rank_tolerance)
    }

    fn projected_preference_factor(
        &self,
        preference: &DenseMatrix,
        rank_tolerance: f64,
    ) -> Result<ColumnPivotedQr, LinearAlgebraError> {
        if preference.rows().len() != self.triangular.len() {
            return Err(LinearAlgebraError(format!(
                "preference rows {}: expected {} variables",
                preference.rows().len(),
                self.triangular.len()
            )));
        }
        if !rank_tolerance.is_finite() || rank_tolerance <= 0.0 || rank_tolerance >= 1.0 {
            return Err(LinearAlgebraError(format!(
                "rank tolerance {rank_tolerance:?}: expected a finite value strictly between 0 and 1"
            )));
        }
        let columns = self.project_preference_columns(preference, rank_tolerance)?;
        let projected = DenseMatrix::new(self.triangular.len(), columns)?.transpose();
        ColumnPivotedQr::factor(&projected, rank_tolerance)
    }

    fn project_preference_columns(
        &self,
        preference: &DenseMatrix,
        rank_tolerance: f64,
    ) -> Result<Vec<Vec<f64>>, LinearAlgebraError> {
        (0..preference.columns())
            .map(|column| {
                let values: Vec<_> = preference.rows().iter().map(|row| row[column]).collect();
                let mut projected = self.project_orthogonal(&values)?;
                let original_norm = vector_norm(&values)?;
                let projected_norm = vector_norm(&projected)?;
                if original_norm == 0.0 || projected_norm / original_norm <= rank_tolerance {
                    projected.fill(0.0);
                }
                Ok(projected)
            })
            .collect()
    }
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

fn validate_targets(preference: &DenseMatrix, targets: &[f64]) -> Result<(), LinearAlgebraError> {
    if targets.len() != preference.columns() {
        return Err(LinearAlgebraError(format!(
            "preference target count {}: expected {} columns",
            targets.len(),
            preference.columns()
        )));
    }
    for &target in targets {
        finite(target, "preference target")?;
    }
    Ok(())
}

fn vector_norm(values: &[f64]) -> Result<f64, LinearAlgebraError> {
    finite(
        values
            .iter()
            .fold(0.0_f64, |norm, value| norm.hypot(*value)),
        "projection vector norm",
    )
}
