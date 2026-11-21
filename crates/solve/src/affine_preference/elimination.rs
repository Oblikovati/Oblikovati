// SPDX-License-Identifier: GPL-2.0-only
//! Deterministic exact elimination with bounded rational growth.
use super::{AffinePreferenceError, policy::Policy};
use num_rational::BigRational;
use num_traits::Zero;

pub(super) fn solve<F: FnMut() -> bool>(
    mut matrix: Vec<Vec<BigRational>>,
    mut seed: Vec<BigRational>,
    policy: &mut Policy<'_, F>,
) -> Result<Vec<BigRational>, AffinePreferenceError> {
    let pivots = reduce(&mut matrix, seed.len(), policy)?;
    check_consistency(&matrix, seed.len())?;
    for (row, column) in pivots {
        policy.poll()?;
        seed[column] = pivot_value(&matrix[row], column, &seed, policy)?;
    }
    Ok(seed)
}

fn reduce<F: FnMut() -> bool>(
    matrix: &mut [Vec<BigRational>],
    columns: usize,
    policy: &mut Policy<'_, F>,
) -> Result<Vec<(usize, usize)>, AffinePreferenceError> {
    let mut pivots = Vec::new();
    let mut next = 0;
    for column in 0..columns {
        if !reduce_column(matrix, next, column, policy)? {
            continue;
        }
        pivots.push((next, column));
        next += 1;
        if next == matrix.len() {
            break;
        }
    }
    Ok(pivots)
}

fn reduce_column<F: FnMut() -> bool>(
    matrix: &mut [Vec<BigRational>],
    next: usize,
    column: usize,
    policy: &mut Policy<'_, F>,
) -> Result<bool, AffinePreferenceError> {
    policy.poll()?;
    let Some(pivot) = (next..matrix.len()).find(|&row| !matrix[row][column].is_zero()) else {
        return Ok(false);
    };
    matrix.swap(next, pivot);
    normalize(&mut matrix[next], column, policy)?;
    eliminate(matrix, next, column, policy)?;
    Ok(true)
}

fn normalize<F: FnMut() -> bool>(
    row: &mut [BigRational],
    column: usize,
    policy: &Policy<'_, F>,
) -> Result<(), AffinePreferenceError> {
    let divisor = row[column].clone();
    for value in &mut row[column..] {
        if value.is_zero() {
            continue;
        }
        *value /= &divisor;
        policy.check(value)?;
    }
    Ok(())
}

fn eliminate<F: FnMut() -> bool>(
    matrix: &mut [Vec<BigRational>],
    pivot: usize,
    column: usize,
    policy: &mut Policy<'_, F>,
) -> Result<(), AffinePreferenceError> {
    let pivot_row = matrix[pivot].clone();
    for (index, row) in matrix.iter_mut().enumerate() {
        policy.poll()?;
        if index == pivot || row[column].is_zero() {
            continue;
        }
        let factor = row[column].clone();
        subtract_row(row, &pivot_row, &factor, column, policy)?;
    }
    Ok(())
}

fn subtract_row<F: FnMut() -> bool>(
    row: &mut [BigRational],
    pivot: &[BigRational],
    factor: &BigRational,
    column: usize,
    policy: &Policy<'_, F>,
) -> Result<(), AffinePreferenceError> {
    for (value, coefficient) in row[column..].iter_mut().zip(&pivot[column..]) {
        if coefficient.is_zero() {
            continue;
        }
        let change = factor * coefficient;
        policy.check(&change)?;
        *value -= change;
        policy.check(value)?;
    }
    Ok(())
}

fn check_consistency(
    matrix: &[Vec<BigRational>],
    columns: usize,
) -> Result<(), AffinePreferenceError> {
    if matrix
        .iter()
        .any(|row| row[..columns].iter().all(Zero::is_zero) && !row[columns].is_zero())
    {
        return Err(AffinePreferenceError::Inconsistent);
    }
    Ok(())
}

fn pivot_value<F: FnMut() -> bool>(
    row: &[BigRational],
    pivot: usize,
    seed: &[BigRational],
    policy: &Policy<'_, F>,
) -> Result<BigRational, AffinePreferenceError> {
    let mut value = row[seed.len()].clone();
    for (column, (coefficient, seed)) in row.iter().zip(seed).enumerate() {
        if column == pivot || coefficient.is_zero() {
            continue;
        }
        let term = coefficient * seed;
        policy.check(&term)?;
        value -= term;
        policy.check(&value)?;
    }
    Ok(value)
}
