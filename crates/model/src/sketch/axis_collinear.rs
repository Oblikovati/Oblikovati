// SPDX-License-Identifier: GPL-2.0-only
use super::{axis_operand::AxisLayout, ellipse_axis, line_residual, residual::point};
use oblikovati_solve::{
    ad::{DualNumber, DualVector},
    residual::ResidualError,
};
pub(super) fn collinear_residual(
    variables: &[DualNumber],
    layouts: [AxisLayout; 2],
) -> Result<Vec<DualNumber>, ResidualError> {
    let directions = &variables[4..];
    let mut residuals = ellipse_axis::pair_residual(directions, layouts, false)?;
    let direction = stable_offset_direction(directions, layouts)?;
    let offset = point(variables, 2)?.minus(&point(variables, 0)?)?;
    residuals.push(direction.cross(&offset)?);
    Ok(residuals)
}
fn stable_offset_direction(
    variables: &[DualNumber],
    layouts: [AxisLayout; 2],
) -> Result<DualVector<2>, ResidualError> {
    // #3788: a direct rotation frame avoids amplifying offset derivatives through a short line.
    let prefer_second =
        matches!(layouts[0], AxisLayout::Line) && !matches!(layouts[1], AxisLayout::Line);
    let (variables, layout) = if prefer_second {
        (&variables[layouts[0].width()..], layouts[1])
    } else {
        (&variables[..layouts[0].width()], layouts[0])
    };
    operand_direction(variables, layout)
}
fn operand_direction(
    variables: &[DualNumber],
    layout: AxisLayout,
) -> Result<DualVector<2>, ResidualError> {
    if let AxisLayout::Line = layout {
        return line_residual::unit_direction(&point(variables, 2)?.minus(&point(variables, 0)?)?);
    }
    let angle = ellipse_axis::operand_angle(variables, layout)?;
    Ok(DualVector::new([angle.cos()?, angle.sin()?])?)
}

#[cfg(test)]
#[path = "../../tests/sketch/axis_collinear.rs"]
mod tests;
