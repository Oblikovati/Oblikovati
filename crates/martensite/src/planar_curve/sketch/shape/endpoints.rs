// SPDX-License-Identifier: GPL-2.0-only
use super::{ResolvedSketchCurve, SketchCurveFailure, construct};
pub(super) fn measure(
    curve: &ResolvedSketchCurve,
    parameters: [f64; 2],
    endpoints: [[f64; 2]; 2],
    limit: f64,
) -> Result<[f64; 2], SketchCurveFailure> {
    let native = construct::build_curve(&curve.definition)
        .map_err(|cause| SketchCurveFailure::curve(curve.entity, cause))?;
    let mut residuals = [0.0; 2];
    for (index, parameter) in parameters.into_iter().enumerate() {
        let actual = native
            .value(parameter)
            .map_err(|cause| SketchCurveFailure::curve(curve.entity, cause.into()))?;
        let residual = (actual.x() - endpoints[index][0]).hypot(actual.y() - endpoints[index][1]);
        if !residual.is_finite() || residual > limit {
            return Err(SketchCurveFailure::input(
                curve.entity,
                format!(
                    "endpoint {index} = {:?}, residual {residual}: expected distance <= {limit} to the authored analytic curve",
                    endpoints[index]
                ),
            ));
        }
        residuals[index] = residual;
    }
    Ok(residuals)
}
