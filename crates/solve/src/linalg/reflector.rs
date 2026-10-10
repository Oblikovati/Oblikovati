// SPDX-License-Identifier: GPL-2.0-only
use super::{LinearAlgebraError, finite};

pub(super) fn build(rows: &[Vec<f64>], pivot: usize, norm: f64) -> Vec<f64> {
    // Normalize before the signed update so v*v cannot overflow (#3701).
    let mut vector: Vec<f64> = rows
        .iter()
        .skip(pivot)
        .map(|row| row[pivot] / norm)
        .collect();
    vector[0] += if vector[0] >= 0.0 { 1.0 } else { -1.0 };
    let length = vector
        .iter()
        .fold(0.0_f64, |norm, entry| norm.hypot(*entry));
    for entry in &mut vector {
        *entry /= length;
    }
    vector
}

pub(super) fn apply_matrix(
    rows: &mut [Vec<f64>],
    pivot: usize,
    vector: &[f64],
) -> Result<(), LinearAlgebraError> {
    for column in pivot..rows[pivot].len() {
        let mut values: Vec<f64> = rows.iter().skip(pivot).map(|row| row[column]).collect();
        apply(&mut values, vector)?;
        for (row, value) in rows.iter_mut().skip(pivot).zip(values) {
            row[column] = value;
        }
    }
    Ok(())
}

pub(super) fn apply(values: &mut [f64], vector: &[f64]) -> Result<(), LinearAlgebraError> {
    let scale = values
        .iter()
        .fold(0.0_f64, |largest, value| largest.max(value.abs()));
    if scale == 0.0 {
        return Ok(());
    }
    let dot: f64 = values
        .iter()
        .zip(vector)
        .map(|(value, coefficient)| (value / scale) * coefficient)
        .sum();
    for (value, coefficient) in values.iter_mut().zip(vector) {
        // Two subtractions avoid overflowing 2*projection when the result is finite.
        let projection = finite((dot * coefficient) * scale, "Householder projection")?;
        *value = finite((*value - projection) - projection, "Householder update")?;
    }
    Ok(())
}
