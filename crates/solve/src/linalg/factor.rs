// SPDX-License-Identifier: GPL-2.0-only
use super::{ColumnPivotedQr, DenseMatrix, LinearAlgebraError, reflector};

impl ColumnPivotedQr {
    /// Factor without normal equations or input mutation; equal pivots retain current column order.
    /// Example: `ColumnPivotedQr::factor(&matrix, 1e-9)?`.
    pub fn factor(
        matrix: &DenseMatrix,
        relative_tolerance: f64,
    ) -> Result<Self, LinearAlgebraError> {
        if !relative_tolerance.is_finite() || relative_tolerance <= 0.0 || relative_tolerance >= 1.0
        {
            return Err(LinearAlgebraError(format!(
                "rank tolerance {relative_tolerance:?}: expected a finite value strictly between 0 and 1"
            )));
        }
        let scale = matrix
            .rows
            .iter()
            .flatten()
            .fold(0.0_f64, |largest, entry| largest.max(entry.abs()));
        let scale = if scale == 0.0 { 1.0 } else { scale };
        let triangular = matrix
            .rows
            .iter()
            .map(|row| row.iter().map(|entry| entry / scale).collect())
            .collect();
        let mut factor = Self {
            triangular,
            permutation: (0..matrix.columns).collect(),
            reflectors: vec![],
            scale,
        };
        factor.reduce(relative_tolerance)?;
        Ok(factor)
    }

    fn reduce(&mut self, tolerance: f64) -> Result<(), LinearAlgebraError> {
        let mut first = 0.0;
        for pivot in 0..self.triangular.len().min(self.permutation.len()) {
            let norm = self.pivot_column(pivot);
            if pivot == 0 {
                first = norm;
            }
            if norm == 0.0 || norm / first < tolerance {
                break;
            }
            let vector = reflector::build(&self.triangular, pivot, norm);
            reflector::apply_matrix(&mut self.triangular, pivot, &vector)?;
            for row in self.triangular.iter_mut().skip(pivot + 1) {
                row[pivot] = 0.0;
            }
            self.reflectors.push(vector);
        }
        Ok(())
    }

    fn pivot_column(&mut self, pivot: usize) -> f64 {
        let mut best = pivot;
        let mut norm = column_norm(&self.triangular, pivot, pivot);
        for column in pivot + 1..self.permutation.len() {
            let candidate = column_norm(&self.triangular, pivot, column);
            if candidate > norm {
                best = column;
                norm = candidate;
            }
        }
        for row in &mut self.triangular {
            row.swap(pivot, best);
        }
        self.permutation.swap(pivot, best);
        norm
    }
}

fn column_norm(rows: &[Vec<f64>], start: usize, column: usize) -> f64 {
    rows.iter()
        .skip(start)
        .fold(0.0_f64, |norm, row| norm.hypot(row[column]))
}
