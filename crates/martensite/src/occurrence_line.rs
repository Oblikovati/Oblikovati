// SPDX-License-Identifier: GPL-2.0-only
//! Map a sight line to an occurrence without changing its geometry.
use martensite_tkmath::gp::{GeneralTransform3, Matrix3, Xyz};
use oblikovati_api::types::Matrix;

/// Transform an owner-space line into component coordinates with its direction unnormalized.
/// Example: `let local = line_in_occurrence(&placement, sight_line)?;`.
pub fn line_in_occurrence(
    placement: &Matrix,
    line: [[f64; 3]; 2],
) -> Result<[[f64; 3]; 2], String> {
    let cells = &placement.cells;
    if !cells.iter().all(|value| value.is_finite()) || cells[12..] != [0., 0., 0., 1.] {
        return Err(format!(
            "occurrence placement {cells:?}: expected a finite affine matrix"
        ));
    }
    if !line.iter().flatten().all(|value| value.is_finite()) {
        return Err(format!("sight line {line:?}: expected finite coordinates"));
    }
    let matrix = Matrix3::new(std::array::from_fn(|row| {
        std::array::from_fn(|column| cells[row * 4 + column])
    }));
    let inverse = checked_inverse(matrix, cells)?;
    let offset = Xyz::new(cells[3], cells[7], cells[11]);
    let transform = GeneralTransform3::new(inverse, inverse.apply(offset).reversed());
    let origin = transform.apply(xyz(line[0]));
    let direction = inverse.apply(xyz(line[1]));
    let result = [*origin.coordinates(), *direction.coordinates()];
    let magnitude = length(result[1]);
    if !result.iter().flatten().all(|value| value.is_finite())
        || !magnitude.is_finite()
        || magnitude == 0.
    {
        return Err(format!(
            "sight line {line:?} and placement {cells:?}: expected finite coordinates and nonzero direction"
        ));
    }
    Ok(result)
}

fn checked_inverse(matrix: Matrix3, cells: &[f64; 16]) -> Result<Matrix3, String> {
    if matrix.is_singular() {
        return Err(format!(
            "occurrence placement {cells:?}: expected an invertible matrix"
        ));
    }
    let inverse = matrix.inverted().map_err(|cause| {
        format!("occurrence placement {cells:?}: expected an invertible matrix: {cause}")
    })?;
    let condition = matrix_norm(matrix) * matrix_norm(inverse);
    if !condition.is_finite() || condition > 1. / f64::EPSILON.sqrt() {
        return Err(format!(
            "occurrence placement {cells:?}: expected a well-conditioned inverse, got condition {condition}"
        ));
    }
    Ok(inverse)
}
fn matrix_norm(matrix: Matrix3) -> f64 {
    matrix
        .coefficients()
        .iter()
        .map(|row| row.iter().map(|cell| cell.abs()).sum::<f64>())
        .fold(0., f64::max)
}
fn xyz(value: [f64; 3]) -> Xyz {
    Xyz::new(value[0], value[1], value[2])
}
fn length(value: [f64; 3]) -> f64 {
    value[0].hypot(value[1]).hypot(value[2])
}

#[cfg(test)]
#[path = "../tests/occurrence_line.rs"]
mod tests;
