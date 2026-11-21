// SPDX-License-Identifier: GPL-2.0-only
//! Prepare and verify KKT equations for explicit affine inputs.
use super::{AffinePreferenceError, AffinePreferenceInput, policy::Policy};
use num_rational::BigRational;
use num_traits::{One, Zero};
use std::collections::BTreeSet;

pub(super) struct Problem {
    rows: Vec<Vec<BigRational>>,
    rhs: Vec<BigRational>,
    pub initial: Vec<BigRational>,
    targets: Vec<Option<BigRational>>,
}

pub(super) fn prepare<F: FnMut() -> bool>(
    input: AffinePreferenceInput<'_>,
    policy: &mut Policy<'_, F>,
) -> Result<Problem, AffinePreferenceError> {
    validate_shape(input, policy)?;
    let mut rows = Vec::with_capacity(input.equations.len());
    for row in input.equations {
        rows.push(convert(row, policy)?);
    }
    let rhs = convert(input.right_hand_side, policy)?;
    let initial = convert(input.initial, policy)?;
    let targets = convert_targets(input.targets, input.initial.len(), policy)?;
    Ok(Problem {
        rows,
        rhs,
        initial,
        targets,
    })
}

fn convert_targets<F: FnMut() -> bool>(
    input: &[(usize, f64)],
    count: usize,
    policy: &mut Policy<'_, F>,
) -> Result<Vec<Option<BigRational>>, AffinePreferenceError> {
    let mut targets = vec![None; count];
    for &(index, value) in input {
        policy.poll()?;
        targets[index] = Some(number(value, policy)?);
    }
    Ok(targets)
}

fn validate_shape<F: FnMut() -> bool>(
    input: AffinePreferenceInput<'_>,
    policy: &mut Policy<'_, F>,
) -> Result<(), AffinePreferenceError> {
    policy.poll()?;
    let columns = input.initial.len();
    validate_dimensions(input, policy.limits.max_order)?;
    validate_targets(input.targets, columns)
}

fn validate_dimensions(
    input: AffinePreferenceInput<'_>,
    maximum: usize,
) -> Result<(), AffinePreferenceError> {
    let columns = input.initial.len();
    let rows = input.equations.len();
    if columns == 0
        || rows != input.right_hand_side.len()
        || input.equations.iter().any(|row| row.len() != columns)
    {
        return Err(AffinePreferenceError::Input(format!(
            "affine dimensions {rows} rows, {} right sides, {columns} coordinates: expected nonempty coordinates and matching rows",
            input.right_hand_side.len()
        )));
    }
    validate_order(columns, rows, maximum)
}

fn validate_order(
    columns: usize,
    rows: usize,
    maximum: usize,
) -> Result<(), AffinePreferenceError> {
    let order = columns.checked_add(rows).ok_or_else(|| {
        AffinePreferenceError::Limit(
            "affine order overflow: expected representable dimensions".into(),
        )
    })?;
    if order > maximum {
        return Err(AffinePreferenceError::Limit(format!(
            "affine order {order}: expected at most {maximum}"
        )));
    }
    Ok(())
}

fn validate_targets(targets: &[(usize, f64)], columns: usize) -> Result<(), AffinePreferenceError> {
    let mut seen = BTreeSet::new();
    for &(index, value) in targets {
        if index >= columns || !value.is_finite() || !seen.insert(index) {
            return Err(AffinePreferenceError::Input(format!(
                "target ({index}, {value:?}): expected a unique in-range index and finite value"
            )));
        }
    }
    Ok(())
}

fn convert<F: FnMut() -> bool>(
    values: &[f64],
    policy: &mut Policy<'_, F>,
) -> Result<Vec<BigRational>, AffinePreferenceError> {
    let mut result = Vec::with_capacity(values.len());
    for &value in values {
        policy.poll()?;
        result.push(number(value, policy)?);
    }
    Ok(result)
}
fn number<F: FnMut() -> bool>(
    value: f64,
    policy: &Policy<'_, F>,
) -> Result<BigRational, AffinePreferenceError> {
    let rational = BigRational::from_float(value).ok_or_else(|| {
        AffinePreferenceError::Input(format!("affine value {value:?}: expected finite binary64"))
    })?;
    policy.check(&rational)?;
    Ok(rational)
}

pub(super) fn kkt<F: FnMut() -> bool>(
    problem: &Problem,
    policy: &mut Policy<'_, F>,
) -> Result<(Vec<Vec<BigRational>>, Vec<BigRational>), AffinePreferenceError> {
    let n = problem.initial.len();
    let order = n + problem.rows.len();
    let mut matrix = vec![vec![BigRational::zero(); order + 1]; order];
    fill_kkt(&mut matrix, problem, policy)?;
    let mut seed = problem.initial.clone();
    seed.resize(order, BigRational::zero());
    Ok((matrix, seed))
}

fn fill_kkt<F: FnMut() -> bool>(
    matrix: &mut [Vec<BigRational>],
    problem: &Problem,
    policy: &mut Policy<'_, F>,
) -> Result<(), AffinePreferenceError> {
    let n = problem.initial.len();
    let order = n + problem.rows.len();
    for (column, target) in problem.targets.iter().enumerate() {
        policy.poll()?;
        if let Some(target) = target {
            matrix[column][column] = BigRational::one();
            matrix[column][order] = target.clone();
        }
        fill_constraint_column(matrix, &problem.rows, column, n);
    }
    for (row, value) in problem.rhs.iter().enumerate() {
        matrix[n + row][order] = value.clone();
    }
    Ok(())
}

fn fill_constraint_column(
    matrix: &mut [Vec<BigRational>],
    rows: &[Vec<BigRational>],
    column: usize,
    offset: usize,
) {
    for (row, coefficients) in rows.iter().enumerate() {
        matrix[column][offset + row] = coefficients[column].clone();
        matrix[offset + row][column] = coefficients[column].clone();
    }
}

pub(super) fn verify<F: FnMut() -> bool>(
    problem: &Problem,
    solution: &[BigRational],
    policy: &mut Policy<'_, F>,
) -> Result<(), AffinePreferenceError> {
    let n = problem.initial.len();
    for (index, (row, expected)) in problem.rows.iter().zip(&problem.rhs).enumerate() {
        policy.poll()?;
        let actual = dot(row, &solution[..n], policy)?;
        if &actual != expected {
            return Err(AffinePreferenceError::Verification(format!(
                "affine witness row {index} value {actual}: expected {expected}"
            )));
        }
    }
    verify_stationarity(problem, solution, policy)
}

fn verify_stationarity<F: FnMut() -> bool>(
    problem: &Problem,
    solution: &[BigRational],
    policy: &mut Policy<'_, F>,
) -> Result<(), AffinePreferenceError> {
    let n = problem.initial.len();
    for column in 0..n {
        policy.poll()?;
        let gradient = stationarity_column(problem, solution, column, policy)?;
        if !gradient.is_zero() {
            return Err(AffinePreferenceError::Verification(format!(
                "affine witness column {column} gradient {gradient}: expected zero"
            )));
        }
    }
    Ok(())
}
fn stationarity_column<F: FnMut() -> bool>(
    problem: &Problem,
    solution: &[BigRational],
    column: usize,
    policy: &mut Policy<'_, F>,
) -> Result<BigRational, AffinePreferenceError> {
    let mut gradient = problem.targets[column]
        .as_ref()
        .map_or_else(BigRational::zero, |target| &solution[column] - target);
    policy.check(&gradient)?;
    for (row, multiplier) in problem.rows.iter().zip(&solution[problem.initial.len()..]) {
        policy.poll()?;
        let term = &row[column] * multiplier;
        policy.check(&term)?;
        gradient += term;
        policy.check(&gradient)?;
    }
    Ok(gradient)
}
fn dot<F: FnMut() -> bool>(
    row: &[BigRational],
    values: &[BigRational],
    policy: &mut Policy<'_, F>,
) -> Result<BigRational, AffinePreferenceError> {
    let mut total = BigRational::zero();
    for (coefficient, value) in row.iter().zip(values) {
        policy.poll()?;
        let term = coefficient * value;
        policy.check(&term)?;
        total += term;
        policy.check(&total)?;
    }
    Ok(total)
}
