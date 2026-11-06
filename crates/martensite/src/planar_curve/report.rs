// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarCurveFailure, PlanarIntersectionOptions, PlanarIntersectionPoint,
    PlanarIntersectionReport, PlanarOverlap, construct::CurveHandle, transition,
};
use martensite_tkg2d::{geom2d::Geom2dCurve, geom2d_adaptor::Geom2dAdaptorCurve};
use martensite_tkgeomalgo::{
    geom2d_int::GInter,
    int_res2d::{IntersectionPoint, IntersectionSegment},
};
use martensite_tkmath::gp::Point2;
use std::cmp::Ordering;

pub(super) fn intersect(
    first: &CurveHandle,
    second: &CurveHandle,
    options: PlanarIntersectionOptions,
) -> Result<PlanarIntersectionReport, PlanarCurveFailure> {
    let first_adaptor = Geom2dAdaptorCurve::with_curve(first)?;
    let second_adaptor = Geom2dAdaptorCurve::with_curve(second)?;
    let native = GInter::between(
        &first_adaptor,
        &second_adaptor,
        options.confusion_tolerance,
        options.intersection_tolerance,
    )?;
    extract(&native, [first, second], options.maximum_point_residual)
}
fn extract(
    native: &GInter,
    curves: [&CurveHandle; 2],
    limit: f64,
) -> Result<PlanarIntersectionReport, PlanarCurveFailure> {
    if !native.is_done() {
        return Err(PlanarCurveFailure::incomplete());
    }
    let mut points = (1..=native.nb_points()?)
        .map(|index| convert_point(native.point(index)?, curves, limit))
        .collect::<Result<Vec<_>, PlanarCurveFailure>>()?;
    let mut overlaps = (1..=native.nb_segments()?)
        .map(|index| convert_overlap(native.segment(index)?, curves, limit))
        .collect::<Result<Vec<_>, PlanarCurveFailure>>()?;
    points.sort_by(compare_points);
    sort_overlaps(&mut overlaps);
    Ok(PlanarIntersectionReport { points, overlaps })
}
fn sort_overlaps(overlaps: &mut [PlanarOverlap]) {
    overlaps.sort_by(|a, b| {
        compare_bounds(&a.first, &b.first)
            .then_with(|| compare_bounds(&a.last, &b.last))
            .then(a.opposite.cmp(&b.opposite))
    });
}
fn convert_point(
    native: IntersectionPoint,
    curves: [&CurveHandle; 2],
    limit: f64,
) -> Result<PlanarIntersectionPoint, PlanarCurveFailure> {
    let position = native.value();
    let parameters = [native.param_on_first(), native.param_on_second()];
    let residuals = [
        residual(curves[0], parameters[0], position, limit)?,
        residual(curves[1], parameters[1], position, limit)?,
    ];
    Ok(PlanarIntersectionPoint {
        position: [position.x(), position.y()],
        parameters,
        residuals,
        transitions: [
            transition::convert(native.transition_of_first())?,
            transition::convert(native.transition_of_second())?,
        ],
    })
}
fn residual(
    curve: &CurveHandle,
    parameter: f64,
    position: Point2,
    limit: f64,
) -> Result<f64, PlanarCurveFailure> {
    let source = curve.get().ok_or_else(|| {
        PlanarCurveFailure::result("null curve: expected a constructed analytic input".into())
    })?;
    require_parameter(source, parameter)?;
    let evaluated = source.value(parameter)?;
    let distance = (evaluated.x() - position.x()).hypot(evaluated.y() - position.y());
    if !distance.is_finite() || distance > limit {
        return Err(PlanarCurveFailure::result(format!(
            "intersection point {position:?} at {parameter}, residual {distance}: expected finite distance <= {limit}"
        )));
    }
    Ok(distance)
}
fn require_parameter(source: &dyn Geom2dCurve, parameter: f64) -> Result<(), PlanarCurveFailure> {
    if !parameter.is_finite()
        || parameter < source.first_parameter()
        || parameter > source.last_parameter()
    {
        return Err(PlanarCurveFailure::result(format!(
            "intersection parameter {parameter}: expected finite parameter in [{}, {}]",
            source.first_parameter(),
            source.last_parameter()
        )));
    }
    Ok(())
}
fn convert_overlap(
    native: IntersectionSegment,
    curves: [&CurveHandle; 2],
    limit: f64,
) -> Result<PlanarOverlap, PlanarCurveFailure> {
    let first = if native.has_first_point() {
        Some(convert_point(native.first_point()?, curves, limit)?)
    } else {
        None
    };
    let last = if native.has_last_point() {
        Some(convert_point(native.last_point()?, curves, limit)?)
    } else {
        None
    };
    Ok(PlanarOverlap {
        first,
        last,
        opposite: native.is_opposite(),
    })
}
fn compare_points(first: &PlanarIntersectionPoint, second: &PlanarIntersectionPoint) -> Ordering {
    first.parameters[0]
        .total_cmp(&second.parameters[0])
        .then_with(|| first.parameters[1].total_cmp(&second.parameters[1]))
        .then_with(|| first.position[0].total_cmp(&second.position[0]))
        .then_with(|| first.position[1].total_cmp(&second.position[1]))
}
fn compare_bounds(
    first: &Option<PlanarIntersectionPoint>,
    second: &Option<PlanarIntersectionPoint>,
) -> Ordering {
    match (first, second) {
        (Some(first), Some(second)) => compare_points(first, second),
        (None, None) => Ordering::Equal,
        (None, Some(_)) => Ordering::Less,
        (Some(_), None) => Ordering::Greater,
    }
}

#[cfg(test)]
#[path = "../../tests/unit/planar_curve_report.rs"]
mod tests;
