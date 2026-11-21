// SPDX-License-Identifier: GPL-2.0-only
use super::{
    LineContactSide, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchError,
};
use super::{line_residual::signed_offset, tangent::select_signed_side};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};
impl Sketch2D {
    /// Retain the picked point-to-line side; example: `sketch.add_offset_dimension(id, point, line, parameter, false)?`.
    pub fn add_offset_dimension(
        &mut self,
        id: SketchConstraintId,
        point: SketchEntityId,
        line: SketchEntityId,
        parameter: u64,
        linear_diameter: bool,
    ) -> Result<(), SketchError> {
        let side = self.offset_seed_side(id, point, line)?;
        self.add_constraint(
            id,
            PlanarConstraint::Offset {
                point,
                line,
                parameter,
                side,
                linear_diameter,
            },
        )
    }
    fn offset_seed_side(
        &self,
        id: SketchConstraintId,
        point: SketchEntityId,
        line: SketchEntityId,
    ) -> Result<LineContactSide, SketchError> {
        let [a, b] = self.line(line)?.endpoints;
        let failure = |cause| SketchError::ContactSelection {
            constraint: id,
            cause,
        };
        let variables = self.point_constants(&[point, a, b], failure)?;
        let distance = signed_offset(&variables, 2, 0).map_err(failure)?;
        select_signed_side(distance.value()).map_err(failure)
    }
}
pub(super) fn offset_residual(
    variables: &[DualNumber],
    target: f64,
    side: LineContactSide,
) -> Result<Vec<DualNumber>, ResidualError> {
    let sign = if side == LineContactSide::Left {
        1.0
    } else {
        -1.0
    };
    // A signed row retains rank at zero offset; Go's absolute value is stationary (#3715).
    Ok(vec![
        signed_offset(variables, 2, 0)?
            .scaled(sign)?
            .shifted(-target)?,
    ])
}
#[cfg(test)]
#[path = "../../tests/sketch/offset_residuals.rs"]
mod tests;
