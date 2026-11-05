// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkernel::ncollection::Array1;
use martensite_tkmath::geom_abs::Shape;
pub(super) fn continuity_bounds(
    curve: &Geom2dAdaptorCurve,
    limit: usize,
) -> Result<Vec<f64>, PlanarCurveFailure> {
    let count = curve.nb_intervals(Shape::C2)?;
    if count < 1 {
        return Err(PlanarCurveFailure::result(format!(
            "pick interval count {count}: expected a positive native count"
        )));
    }
    if count as usize > limit {
        return Err(
            PlanarCurveFailure::incomplete_operation("planar pick interval budget").with_context(
                format!("pick interval count {count}: expected at most {limit}"),
            ),
        );
    }
    let mut bounds = Array1::<f64>::with_bounds(1, count + 1).map_err(|cause| {
        PlanarCurveFailure::result(format!(
            "pick interval count {count}: expected an allocated boundary array: {cause}"
        ))
    })?;
    curve.intervals(&mut bounds, Shape::C2)?;
    let bounds = bounds.as_slice().to_vec();
    validate_bounds(&bounds)?;
    Ok(bounds)
}
fn validate_bounds(bounds: &[f64]) -> Result<(), PlanarCurveFailure> {
    if bounds.iter().any(|value| !value.is_finite())
        || bounds.windows(2).any(|span| span[0] >= span[1])
    {
        return Err(PlanarCurveFailure::result(format!(
            "pick boundaries {bounds:?}: expected strictly increasing native parameters without NaN or IEEE infinity"
        )));
    }
    Ok(())
}
