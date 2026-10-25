// SPDX-License-Identifier: GPL-2.0-only
use super::PlanarCurveFailure;
use martensite_tkgeombase::gce::MakeCirc2d;
use martensite_tkmath::{
    gp::{Circle2, Point2},
    predicates::{OrientationSign, orient2d},
};
pub use oblikovati_model::sketch::{CircumcircleOptions, PlanarCircumcircle};
/// Construct a circular seed through three points using the released kernel.
/// Example: `construct_planar_circumcircle(points, options)?` before authoring a circle or arc.
pub fn construct_planar_circumcircle(
    points: [[f64; 2]; 3],
    options: CircumcircleOptions,
) -> Result<PlanarCircumcircle, PlanarCurveFailure> {
    construct(points, options).map_err(|failure| {
        failure.with_context(format!(
            "circumcircle points {points:?}, options {options:?}"
        ))
    })
}
fn construct(
    points: [[f64; 2]; 3],
    options: CircumcircleOptions,
) -> Result<PlanarCircumcircle, PlanarCurveFailure> {
    require_options(options)?;
    let counter_clockwise = classify_points(points)?;
    let [a, b, c] = points.map(|p| Point2::new(p[0], p[1]));
    let circle = MakeCirc2d::through_points(a, b, c)?.value()?;
    require_circle(circle)?;
    let center = [circle.location().x(), circle.location().y()];
    let point_residuals = measure_residuals(points, center, circle.radius(), options)?;
    Ok(PlanarCircumcircle {
        center,
        radius: circle.radius(),
        counter_clockwise,
        point_residuals,
    })
}
pub(super) fn require_options(options: CircumcircleOptions) -> Result<(), PlanarCurveFailure> {
    if !options.maximum_point_residual.is_finite() || options.maximum_point_residual < 0.0 {
        return Err(PlanarCurveFailure::input(format!(
            "circumcircle bound {}: expected a finite nonnegative residual bound",
            options.maximum_point_residual
        )));
    }
    Ok(())
}
pub(super) fn classify_points(points: [[f64; 2]; 3]) -> Result<bool, PlanarCurveFailure> {
    match classify_turn(points)? {
        OrientationSign::Positive => Ok(true),
        OrientationSign::Negative => Ok(false),
        OrientationSign::Zero => Err(PlanarCurveFailure::input(format!(
            "circumcircle points {points:?}: expected three distinct noncollinear points"
        ))),
    }
}
fn require_circle(circle: Circle2) -> Result<(), PlanarCurveFailure> {
    let center = [circle.location().x(), circle.location().y()];
    // gce returns its default RealLast circle when bisectors are judged parallel (OCCT 3d097a03).
    if circle == Circle2::default()
        || !circle.radius().is_finite()
        || circle.radius() <= 0.0
        || center.iter().any(|v| !v.is_finite())
    {
        return Err(PlanarCurveFailure::result(format!(
            "native circle center {center:?}, radius {}: expected a finite positive constructed circle, not the native default sentinel",
            circle.radius()
        )));
    }
    Ok(())
}
pub(super) fn measure_residuals(
    points: [[f64; 2]; 3],
    center: [f64; 2],
    radius: f64,
    options: CircumcircleOptions,
) -> Result<[f64; 3], PlanarCurveFailure> {
    let residuals = points.map(|p| ((p[0] - center[0]).hypot(p[1] - center[1]) - radius).abs());
    if residuals
        .iter()
        .any(|value| !value.is_finite() || *value > options.maximum_point_residual)
    {
        return Err(PlanarCurveFailure::result(format!(
            "circumcircle point residuals {residuals:?}: expected each finite and <= {}",
            options.maximum_point_residual
        )));
    }
    Ok(residuals)
}
pub(super) fn classify_turn(points: [[f64; 2]; 3]) -> Result<OrientationSign, PlanarCurveFailure> {
    orient2d(points[0], points[1], points[2])
        .map_err(|cause| PlanarCurveFailure::input(cause.to_string()))
}
