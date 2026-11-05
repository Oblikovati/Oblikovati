// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkgeombase::extrema::{ExtPC2d, POnCurv2d};
pub(super) fn boundaries(
    curve: &Geom2dAdaptorCurve,
    query: Point2,
    bounds: &[f64],
) -> Result<Vec<PlanarPickPoint>, PlanarCurveFailure> {
    bounds
        .iter()
        .copied()
        .filter(|value| !precision::is_infinite(*value))
        .map(|parameter| {
            witness(
                curve.value(parameter)?,
                parameter,
                query,
                PlanarPickOrigin::Boundary,
            )
        })
        .collect()
}
pub(super) fn stationary(
    curve: &Geom2dAdaptorCurve,
    query: Point2,
    span: [f64; 2],
    options: PlanarPickOptionsV1,
) -> Result<Vec<PlanarPickPoint>, PlanarCurveFailure> {
    let extrema = ExtPC2d::with_range(query, curve, span[0], span[1], options.native_tolerance)?;
    if !extrema.is_done() {
        return Err(PlanarCurveFailure::incomplete_operation(
            "native planar pick extrema",
        ));
    }
    let mut points = Vec::new();
    for index in 1..=extrema.nb_ext()? {
        // v0.8.0 (59aca389) recomputes the distance derivative when classifying extrema.
        // Keep this runtime minimum check; root order does not identify the physical branch.
        if extrema.is_min(index)? {
            points.push(checked_stationary(
                curve,
                query,
                span,
                extrema.point(index)?,
                options,
            )?);
        }
    }
    Ok(points)
}
fn checked_stationary(
    curve: &Geom2dAdaptorCurve,
    query: Point2,
    span: [f64; 2],
    native: POnCurv2d,
    options: PlanarPickOptionsV1,
) -> Result<PlanarPickPoint, PlanarCurveFailure> {
    let mut result = checked_native_witness(curve, query, span, native, options)?;
    if result.parameter == span[0] || result.parameter == span[1] {
        result.origin = PlanarPickOrigin::Boundary;
        return Ok(result);
    }
    result.stationarity_residual = Some(stationarity_residual(
        curve,
        query,
        &result,
        options.maximum_stationarity_residual,
    )?);
    Ok(result)
}
fn checked_native_witness(
    curve: &Geom2dAdaptorCurve,
    query: Point2,
    span: [f64; 2],
    native: POnCurv2d,
    options: PlanarPickOptionsV1,
) -> Result<PlanarPickPoint, PlanarCurveFailure> {
    let parameter = native.parameter();
    require_span_parameter(parameter, span)?;
    let mut result = witness(
        curve.value(parameter)?,
        parameter,
        query,
        PlanarPickOrigin::Stationary,
    )?;
    result.position_residual = native
        .value()
        .distance(Point2::new(result.position[0], result.position[1]));
    require_residual(
        result.position_residual,
        options.maximum_position_residual,
        "position",
    )?;
    Ok(result)
}
fn require_span_parameter(parameter: f64, span: [f64; 2]) -> Result<(), PlanarCurveFailure> {
    if !parameter.is_finite() || parameter < span[0] || parameter > span[1] {
        return Err(PlanarCurveFailure::result(format!(
            "pick parameter {parameter}: expected a finite parameter inside {span:?}"
        )));
    }
    Ok(())
}
fn stationarity_residual(
    curve: &Geom2dAdaptorCurve,
    query: Point2,
    result: &PlanarPickPoint,
    limit: f64,
) -> Result<f64, PlanarCurveFailure> {
    let (_, tangent) = curve.d1(result.parameter)?;
    let magnitude = tangent.x().hypot(tangent.y());
    let residual = ((result.position[0] - query.x()) * (tangent.x() / magnitude)
        + (result.position[1] - query.y()) * (tangent.y() / magnitude))
        .abs();
    require_residual(residual, limit, "normal stationarity")?;
    Ok(residual)
}

fn witness(
    point: Point2,
    parameter: f64,
    query: Point2,
    origin: PlanarPickOrigin,
) -> Result<PlanarPickPoint, PlanarCurveFailure> {
    let position = [point.x(), point.y()];
    let distance = (point.x() - query.x()).hypot(point.y() - query.y());
    if !parameter.is_finite()
        || !distance.is_finite()
        || position.iter().any(|value| !value.is_finite())
    {
        return Err(PlanarCurveFailure::result(format!(
            "pick witness {parameter}/{position:?}/{distance}: expected finite native parameter, coordinates and working distance"
        )));
    }
    Ok(PlanarPickPoint {
        parameter,
        position,
        distance,
        position_residual: 0.,
        stationarity_residual: None,
        origin,
    })
}
fn require_residual(value: f64, limit: f64, role: &str) -> Result<(), PlanarCurveFailure> {
    if !value.is_finite() || value > limit {
        return Err(PlanarCurveFailure::result(format!(
            "pick {role} residual {value}: expected a finite length <= {limit}"
        )));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../../../tests/planar_pick_witness.rs"]
mod tests;
