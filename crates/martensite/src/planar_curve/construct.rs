// SPDX-License-Identifier: GPL-2.0-only
use super::PlanarConicFrame;
use super::{PlanarCurveDefinition, PlanarCurveFailure, PlanarGeometry, PlanarIntersectionOptions};
use crate::{PlanarSplineDefinition, planar_spline};
use rustycad_tkernel::standard::Handle;
use rustycad_tkg2d::geom2d::{
    Geom2dCircle, Geom2dCurve, Geom2dEllipse, Geom2dLine, Geom2dTrimmedCurve,
};
use rustycad_tkmath::gp::{Axis2d, Direction2, Ellipse2, Point2};
pub(super) type CurveHandle = Handle<dyn Geom2dCurve>;

pub(super) fn require_options(
    options: PlanarIntersectionOptions,
) -> Result<(), PlanarCurveFailure> {
    let scalars = [
        options.confusion_tolerance,
        options.intersection_tolerance,
        options.maximum_point_residual,
    ];
    if scalars
        .iter()
        .any(|value| !value.is_finite() || *value <= 0.0)
    {
        return Err(PlanarCurveFailure::input(format!(
            "intersection options {options:?}: expected finite positive tolerances and residual bound"
        )));
    }
    Ok(())
}
pub(super) fn build_curve(
    input: &PlanarCurveDefinition,
) -> Result<CurveHandle, PlanarCurveFailure> {
    require_finite(&[input.left_offset])?;
    let basis = build_geometry(&input.geometry)?;
    let curve = match input.bounds {
        None => basis,
        Some(bounds) => trim_curve(&basis, bounds)?,
    };
    if input.left_offset == 0.0 {
        return Ok(curve);
    }
    Ok(Handle::new(planar_spline::build_left_offset(&curve, input.left_offset)?).cast())
}
fn build_geometry(geometry: &PlanarGeometry) -> Result<CurveHandle, PlanarCurveFailure> {
    match geometry {
        PlanarGeometry::Line { origin, direction } => build_line(*origin, *direction),
        PlanarGeometry::Circle { frame, radius } => build_circle(*frame, *radius),
        PlanarGeometry::Ellipse { frame, radii } => build_ellipse(*frame, *radii),
        PlanarGeometry::Spline(definition) => build_spline(definition),
    }
}
fn build_line(origin: [f64; 2], direction: [f64; 2]) -> Result<CurveHandle, PlanarCurveFailure> {
    Ok(Handle::new(Geom2dLine::from_point(
        point(origin)?,
        direction2(direction)?,
    ))
    .cast())
}
fn build_circle(axes: PlanarConicFrame, radius: f64) -> Result<CurveHandle, PlanarCurveFailure> {
    require_positive(&[radius])?;
    let circle = Geom2dCircle::from_axis(
        frame(axes.center, axes.axis)?,
        radius,
        axes.counterclockwise,
    )?;
    Ok(Handle::new(circle).cast())
}
fn build_ellipse(
    axes: PlanarConicFrame,
    radii: [f64; 2],
) -> Result<CurveHandle, PlanarCurveFailure> {
    require_positive(&radii)?;
    let conic = Ellipse2::from_x_axis(
        frame(axes.center, axes.axis)?,
        radii[0],
        radii[1],
        axes.counterclockwise,
    )?;
    Ok(Handle::new(Geom2dEllipse::from_elips2d(conic)).cast())
}
fn build_spline(definition: &PlanarSplineDefinition) -> Result<CurveHandle, PlanarCurveFailure> {
    planar_spline::validation::require_definition(definition).map_err(PlanarCurveFailure::input)?;
    Ok(planar_spline::build_parent(definition)?)
}
fn point(coordinates: [f64; 2]) -> Result<Point2, PlanarCurveFailure> {
    require_finite(&coordinates)?;
    Ok(Point2::new(coordinates[0], coordinates[1]))
}
fn direction2(coordinates: [f64; 2]) -> Result<Direction2, PlanarCurveFailure> {
    require_finite(&coordinates)?;
    Ok(Direction2::new(coordinates[0], coordinates[1])?)
}
fn frame(center: [f64; 2], axis: [f64; 2]) -> Result<Axis2d, PlanarCurveFailure> {
    Ok(Axis2d::new(point(center)?, direction2(axis)?))
}
fn require_finite(values: &[f64]) -> Result<(), PlanarCurveFailure> {
    if values.iter().any(|value| !value.is_finite()) {
        return Err(PlanarCurveFailure::input(format!(
            "curve values {values:?}: expected finite coordinates or scalars"
        )));
    }
    Ok(())
}
fn require_positive(values: &[f64]) -> Result<(), PlanarCurveFailure> {
    require_finite(values)?;
    if values.iter().any(|value| *value <= 0.0) {
        return Err(PlanarCurveFailure::input(format!(
            "curve radii {values:?}: expected positive values"
        )));
    }
    Ok(())
}
fn trim_curve(curve: &CurveHandle, bounds: [f64; 2]) -> Result<CurveHandle, PlanarCurveFailure> {
    require_finite(&bounds)?;
    if bounds[0] >= bounds[1] {
        return Err(PlanarCurveFailure::input(format!(
            "curve bounds {bounds:?}: expected strictly increasing native parameters"
        )));
    }
    require_trim_domain(curve, bounds)?;
    let trimmed = Geom2dTrimmedCurve::new(curve, bounds[0], bounds[1], true, false)?;
    if [trimmed.first_parameter(), trimmed.last_parameter()] != bounds {
        return Err(PlanarCurveFailure::result(format!(
            "trim domain {:?}: expected requested bounds {bounds:?}",
            [trimmed.first_parameter(), trimmed.last_parameter()]
        )));
    }
    Ok(Handle::new(trimmed).cast())
}

fn require_trim_domain(curve: &CurveHandle, bounds: [f64; 2]) -> Result<(), PlanarCurveFailure> {
    if curve.is_periodic() {
        let span = bounds[1] - bounds[0];
        let period = curve.period()?;
        if !span.is_finite() || span > period {
            return Err(PlanarCurveFailure::input(format!(
                "periodic bounds {bounds:?}: expected span <= one period {period}"
            )));
        }
        return Ok(());
    }
    let domain = [curve.first_parameter(), curve.last_parameter()];
    if bounds[0] < domain[0] || bounds[1] > domain[1] {
        return Err(PlanarCurveFailure::input(format!(
            "curve bounds {bounds:?}: expected parameters within owned domain {domain:?}"
        )));
    }
    Ok(())
}
