// SPDX-License-Identifier: GPL-2.0-only
use super::{Geom2dCurve, PlanarSplineDefinition, PlanarSplineSample};

pub(crate) fn require_definition(definition: &PlanarSplineDefinition) -> Result<(), String> {
    let counts = [
        definition.poles.len(),
        definition.knots.len(),
        definition.multiplicities.len(),
    ];
    let finite = definition
        .poles
        .iter()
        .flatten()
        .chain(&definition.knots)
        .all(|value| value.is_finite());
    if counts.iter().any(|count| *count > i32::MAX as usize) || !finite {
        return Err(format!(
            "spline {definition:?}: expected finite poles/knots and arrays within signed 32-bit bounds"
        ));
    }
    require_weights(definition.weights.as_deref())?;
    Ok(())
}
pub(super) fn require_scalars(parameter: f64, left_offset: f64) -> Result<(), String> {
    if !parameter.is_finite() || !left_offset.is_finite() {
        return Err(format!(
            "spline parameter {parameter}, left offset {left_offset}: expected finite values"
        ));
    }
    Ok(())
}
pub(super) fn require_parameter(curve: &dyn Geom2dCurve, parameter: f64) -> Result<(), String> {
    let domain = [curve.first_parameter(), curve.last_parameter()];
    if parameter < domain[0] || parameter > domain[1] {
        return Err(format!(
            "spline parameter {parameter}: expected the owned domain {domain:?}"
        ));
    }
    Ok(())
}
pub(super) fn require_finite_sample(
    sample: PlanarSplineSample,
    parameter: f64,
) -> Result<(), String> {
    if sample
        .position
        .into_iter()
        .chain(sample.tangent)
        .any(|value| !value.is_finite())
    {
        return Err(format!(
            "spline sample {sample:?} at {parameter}: expected finite analytic position and tangent"
        ));
    }
    Ok(())
}
pub(super) fn require_regular_sample(
    sample: PlanarSplineSample,
    parameter: f64,
    left_offset: f64,
) -> Result<(), String> {
    if sample.tangent == [0.0, 0.0] {
        return Err(format!(
            "spline tangent {:?} at {parameter}, left offset {left_offset}: expected a regular nonstationary curve",
            sample.tangent
        ));
    }
    Ok(())
}

fn require_weights(weights: Option<&[f64]>) -> Result<(), String> {
    if weights.is_some_and(|weights| weights.iter().any(|weight| !weight.is_finite())) {
        return Err(format!(
            "spline weights {weights:?}: expected finite rational weights"
        ));
    }
    Ok(())
}
