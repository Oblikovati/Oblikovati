// SPDX-License-Identifier: GPL-2.0-only
//! Place a display scene without changing the source model.
use crate::{ViewportScene, ViewportTriangle};
use oblikovati_api::types::Matrix;

impl ViewportScene {
    /// Place a copy of a scene using a row-major affine matrix.
    /// Example: `let placed = scene.placed(&occurrence.placement()?)?;`.
    pub fn placed(&self, placement: &Matrix) -> Result<Self, String> {
        self.validate()?;
        validate_placement(&placement.cells)?;
        let triangles = self
            .triangles
            .iter()
            .map(|triangle| place_triangle(triangle, &placement.cells))
            .collect::<Result<Vec<_>, _>>()?;
        Ok(Self {
            background: self.background,
            triangles,
        })
    }
}

fn validate_placement(cells: &[f64; 16]) -> Result<(), String> {
    if cells.iter().all(|cell| cell.is_finite()) && cells[12..] == [0., 0., 0., 1.] {
        return Ok(());
    }
    Err(format!(
        "placement {cells:?}: expected a finite row-major affine matrix"
    ))
}

fn place_triangle(
    triangle: &ViewportTriangle,
    cells: &[f64; 16],
) -> Result<ViewportTriangle, String> {
    let mut positions = [[0.; 3]; 3];
    for (target, point) in positions.iter_mut().zip(triangle.positions) {
        *target = place_point(point, cells)?;
    }
    Ok(ViewportTriangle {
        positions,
        ..*triangle
    })
}

fn place_point(point: [f32; 3], cells: &[f64; 16]) -> Result<[f32; 3], String> {
    let mut placed = [0.; 3];
    for (axis, value) in placed.iter_mut().enumerate() {
        let row = &cells[axis * 4..axis * 4 + 4];
        let coordinate = row[0] * f64::from(point[0])
            + row[1] * f64::from(point[1])
            + row[2] * f64::from(point[2])
            + row[3];
        *value = coordinate as f32;
        if !value.is_finite() {
            return Err(format!(
                "placed point {point:?} with matrix {cells:?}: expected finite f32 coordinates"
            ));
        }
    }
    Ok(placed)
}

#[cfg(test)]
#[path = "../tests/scene_placement.rs"]
mod tests;
