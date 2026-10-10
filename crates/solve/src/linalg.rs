// SPDX-License-Identifier: GPL-2.0-only

//! Checked dense numerical linear algebra; rank is relative to an explicit tolerance.
mod factor;
mod reflector;
mod solve;
use std::{error::Error, fmt};

/// Finite rectangular row-major matrix; empty axes retain their declared width.
#[derive(Clone, Debug, PartialEq)]
pub struct DenseMatrix {
    columns: usize,
    rows: Vec<Vec<f64>>,
}
impl DenseMatrix {
    /// Validate dimensions and entries before numerical work; example: `DenseMatrix::new(2, vec![vec![1.0, 2.0]])?`.
    pub fn new(columns: usize, rows: Vec<Vec<f64>>) -> Result<Self, LinearAlgebraError> {
        for (index, row) in rows.iter().enumerate() {
            if row.len() != columns {
                return Err(LinearAlgebraError(format!(
                    "matrix row {index} has {} columns: expected {columns}",
                    row.len()
                )));
            }
            for (column, &entry) in row.iter().enumerate() {
                finite(entry, &format!("matrix [{index}, {column}]"))?;
            }
        }
        Ok(Self { columns, rows })
    }
    /// Inspect rows without changing the matrix; example: `matrix.rows()`.
    pub fn rows(&self) -> &[Vec<f64>] {
        &self.rows
    }
    /// Retain the column count for empty matrices; example: `matrix.columns()`.
    pub fn columns(&self) -> usize {
        self.columns
    }
}

/// Invalid matrix, tolerance, rank, right-hand side, or non-finite arithmetic.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct LinearAlgebraError(String);
impl fmt::Display for LinearAlgebraError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(output)
    }
}
impl Error for LinearAlgebraError {}

/// Reusable column-pivoted Householder factorization of a globally scaled matrix.
#[derive(Clone, Debug)]
pub struct ColumnPivotedQr {
    triangular: Vec<Vec<f64>>,
    permutation: Vec<usize>,
    reflectors: Vec<Vec<f64>>,
    scale: f64,
}
impl ColumnPivotedQr {
    /// Read numerical rank at the factorization's relative tolerance; example: `qr.rank()`.
    pub fn rank(&self) -> usize {
        self.reflectors.len()
    }
    /// Map factored columns back to input columns; example: `qr.permutation()[0]`.
    pub fn permutation(&self) -> &[usize] {
        &self.permutation
    }
}

fn finite(value: f64, context: &str) -> Result<f64, LinearAlgebraError> {
    if !value.is_finite() {
        return Err(LinearAlgebraError(format!(
            "{context} value {value:?}: expected finite arithmetic"
        )));
    }
    Ok(value)
}
