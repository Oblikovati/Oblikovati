// SPDX-License-Identifier: GPL-2.0-only
//! Presentation samples from analytic native derivatives.
use super::{PlanarCurveDefinition, PlanarCurveFailure, construct};
use martensite_tkernel::precision;

/// One analytic point and its direction-independent curvature vector.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PlanarCurvatureStation {
    pub position: [f64; 2],
    pub curvature: [f64; 2],
}

/// Sample a finite curve domain for the curvature comb at a bounded display density.
/// Example: `let stations = sample_planar_curvature(&definition, 32)?;`.
pub fn sample_planar_curvature(
    definition: &PlanarCurveDefinition,
    intervals: u16,
) -> Result<Vec<PlanarCurvatureStation>, PlanarCurveFailure> {
    if !(4..=256).contains(&intervals) {
        return Err(PlanarCurveFailure::input(format!(
            "curvature interval count {intervals}: expected 4..=256"
        )));
    }
    let curve = construct::build_curve(definition)?;
    let first = curve.first_parameter();
    let last = curve.last_parameter();
    parameters(first, last, intervals)?
        .map(|at| native_station(&curve, at))
        .collect()
}

fn parameters(
    first: f64,
    last: f64,
    intervals: u16,
) -> Result<impl Iterator<Item = f64>, PlanarCurveFailure> {
    let span = last - first;
    if !span.is_finite() || span <= 0. || [first, last].into_iter().any(precision::is_infinite) {
        return Err(PlanarCurveFailure::input(format!(
            "curvature domain [{first}, {last}]: expected finite increasing bounds"
        )));
    }
    let interior =
        (0..intervals).map(move |index| first + span * (f64::from(index) / f64::from(intervals)));
    Ok(interior.chain(std::iter::once(last)))
}

fn native_station(
    curve: &construct::CurveHandle,
    at: f64,
) -> Result<PlanarCurvatureStation, PlanarCurveFailure> {
    let (point, tangent, acceleration) = curve.d2(at)?;
    station(
        at,
        [point.x(), point.y()],
        [tangent.x(), tangent.y()],
        [acceleration.x(), acceleration.y()],
    )
}

fn station(
    parameter: f64,
    position: [f64; 2],
    tangent: [f64; 2],
    acceleration: [f64; 2],
) -> Result<PlanarCurvatureStation, PlanarCurveFailure> {
    let speed = tangent[0].hypot(tangent[1]);
    if !speed.is_finite() || speed == 0. {
        return Err(PlanarCurveFailure::result(format!(
            "curvature speed {speed} at {parameter}: expected finite nonzero speed"
        )));
    }
    // The sign and direction both reverse when the curve is traversed backward.
    let normal = [-tangent[1] / speed, tangent[0] / speed];
    let signed =
        (normal[0] * (acceleration[0] / speed) + normal[1] * (acceleration[1] / speed)) / speed;
    let curvature = normal.map(|component| component * signed);
    if !position.into_iter().chain(curvature).all(f64::is_finite) {
        return Err(PlanarCurveFailure::result(format!(
            "curvature station {parameter} at {position:?}, vector {curvature:?}: expected finite coordinates"
        )));
    }
    Ok(PlanarCurvatureStation {
        position,
        curvature,
    })
}

#[cfg(test)]
#[path = "../../tests/planar_curvature.rs"]
mod tests;
