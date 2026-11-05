// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, residual::point};
use oblikovati_solve::{
    ad::{DualNumber, DualVector},
    residual::ResidualError,
};
impl Sketch2D {
    /// Locate half the authored sweep without solving or changing points; example: `sketch.arc_midpoint(id)?`.
    pub fn arc_midpoint(&self, id: SketchEntityId) -> Result<[f64; 2], SketchError> {
        let arc = self.arc(id)?;
        let variables = self.point_constants(&[arc.center, arc.start, arc.end], |cause| {
            SketchError::ArcMidpoint { arc: id, cause }
        })?;
        let midpoint = midpoint_target(&variables, arc.counter_clockwise)
            .map_err(|cause| SketchError::ArcMidpoint { arc: id, cause })?;
        Ok(midpoint.components().each_ref().map(DualNumber::value))
    }
}
pub(super) fn midpoint_residual(
    variables: &[DualNumber],
    counter_clockwise: bool,
) -> Result<Vec<DualNumber>, ResidualError> {
    let target = midpoint_target(&variables[2..], counter_clockwise)?;
    // Go #1872 seeds an unsigned bisector. Direct coordinates exclude its complementary root.
    Ok(point(variables, 0)?.minus(&target)?.components().to_vec())
}
fn midpoint_target(
    variables: &[DualNumber],
    counter_clockwise: bool,
) -> Result<DualVector<2>, ResidualError> {
    let center = point(variables, 0)?;
    let start = point(variables, 2)?.minus(&center)?;
    let end = point(variables, 4)?.minus(&center)?;
    let half = super::arc_sweep::radial_sweep(&start, &end, counter_clockwise)?.scaled(0.5)?;
    Ok(center.plus(&rotate_radial(&start, &half)?)?)
}
fn rotate_radial(
    radial: &DualVector<2>,
    angle: &DualNumber,
) -> Result<DualVector<2>, ResidualError> {
    let [x, y] = radial.components();
    let cosine = angle.cos()?;
    let sine = angle.sin()?;
    Ok(DualVector::new([
        x.product(&cosine)?.minus(&y.product(&sine)?)?,
        x.product(&sine)?.plus(&y.product(&cosine)?)?,
    ])?)
}
#[cfg(test)]
#[path = "../../tests/sketch/arc_midpoint_residuals.rs"]
mod tests;
