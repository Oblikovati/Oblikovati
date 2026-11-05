// SPDX-License-Identifier: GPL-2.0-only
use super::super::{PlanarConicFrame, PlanarCurveDefinition, PlanarGeometry, construct};
use super::{ResolvedSketchCurve, SketchCurveFailure, SketchLineDomain};
use oblikovati_model::sketch::{
    OwnedSketchGeometry, SketchArc2D, SketchCircle2D, SketchEllipse2D, SketchEntityId,
    SketchLine2D, SketchReadIndex,
};
use rustycad_tkmath::{elclib, gp::Point2};
pub(super) mod endpoints;
pub(super) fn point(
    sketch: &SketchReadIndex<'_>,
    entity: SketchEntityId,
) -> Result<[f64; 2], SketchCurveFailure> {
    sketch
        .point(entity)
        .map(|point| point.position)
        .map_err(|cause| SketchCurveFailure::model(entity, cause))
}
pub(super) fn resolve(
    sketch: &SketchReadIndex<'_>,
    entity: SketchEntityId,
    domain: SketchLineDomain,
    limit: f64,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    match sketch
        .entity(entity)
        .map_err(|cause| SketchCurveFailure::model(entity, cause))?
    {
        OwnedSketchGeometry::Line(value) => line(sketch, value, domain, limit),
        OwnedSketchGeometry::Circle(value) => circle(sketch, value),
        OwnedSketchGeometry::Arc(value) => arc(sketch, value, limit),
        OwnedSketchGeometry::Ellipse(value) => ellipse(sketch, value),
        OwnedSketchGeometry::Point(_) => Err(SketchCurveFailure::input(
            entity,
            "point geometry: expected an analytic curve".into(),
        )),
        OwnedSketchGeometry::Spline(value) => super::spline::resolve(sketch, value, limit),
        OwnedSketchGeometry::OffsetSpline(value) => super::spline::offset(sketch, value, limit),
    }
}
fn line(
    sketch: &SketchReadIndex<'_>,
    line: SketchLine2D,
    domain: SketchLineDomain,
    limit: f64,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    let entity = line.id;
    let endpoints = [
        point(sketch, line.endpoints[0])?,
        point(sketch, line.endpoints[1])?,
    ];
    let direction: [f64; 2] = std::array::from_fn(|axis| endpoints[1][axis] - endpoints[0][axis]);
    let length = direction[0].hypot(direction[1]);
    let geometry = PlanarGeometry::Line {
        origin: endpoints[0],
        direction,
    };
    let bounds = (domain == SketchLineDomain::Segment).then_some([0.0, length]);
    let mut result = resolved(entity, geometry, bounds)?;
    result.endpoint_residuals = Some(endpoints::measure(
        &result,
        [0.0, length],
        endpoints,
        limit,
    )?);
    Ok(result)
}
fn circle(
    sketch: &SketchReadIndex<'_>,
    circle: SketchCircle2D,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    let entity = circle.id;
    let frame = PlanarConicFrame {
        center: point(sketch, circle.center)?,
        axis: [1.0, 0.0],
        counterclockwise: true,
    };
    resolved(
        entity,
        PlanarGeometry::Circle {
            frame,
            radius: circle.radius,
        },
        None,
    )
}
fn ellipse(
    sketch: &SketchReadIndex<'_>,
    ellipse: SketchEllipse2D,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    let entity = ellipse.id();
    let frame = PlanarConicFrame {
        center: point(sketch, ellipse.center())?,
        axis: ellipse.major_axis(),
        counterclockwise: true,
    };
    resolved(
        entity,
        PlanarGeometry::Ellipse {
            frame,
            radii: ellipse.radii(),
        },
        None,
    )
}
fn arc(
    sketch: &SketchReadIndex<'_>,
    arc: SketchArc2D,
    limit: f64,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    let entity = arc.id;
    let center = point(sketch, arc.center)?;
    let endpoints = [point(sketch, arc.start)?, point(sketch, arc.end)?];
    let axis = std::array::from_fn(|coordinate| endpoints[0][coordinate] - center[coordinate]);
    let frame = PlanarConicFrame {
        center,
        axis,
        counterclockwise: arc.counter_clockwise,
    };
    let radius = sketch
        .arc_radius(entity)
        .map_err(|cause| SketchCurveFailure::model(entity, cause))?;
    let span = arc_span(entity, frame, radius, endpoints[1])?;
    let mut result = resolved(
        entity,
        PlanarGeometry::Circle { frame, radius },
        Some([0.0, span]),
    )?;
    result.endpoint_residuals = Some(endpoints::measure(&result, [0.0, span], endpoints, limit)?);
    Ok(result)
}
fn arc_span(
    entity: SketchEntityId,
    frame: PlanarConicFrame,
    radius: f64,
    end: [f64; 2],
) -> Result<f64, SketchCurveFailure> {
    let circle = construct::native_circle(frame, radius)
        .map_err(|cause| SketchCurveFailure::curve(entity, cause))?;
    let circle = circle
        .circ2d()
        .map_err(|cause| SketchCurveFailure::curve(entity, cause.into()))?;
    let span = elclib::parameter(&circle, Point2::new(end[0], end[1]))
        .map_err(|cause| SketchCurveFailure::curve(entity, cause.into()))?;
    if !span.is_finite() || span <= 0.0 || span >= std::f64::consts::TAU {
        return Err(SketchCurveFailure::input(
            entity,
            format!(
                "arc span {span}: expected distinct endpoint directions with a positive sweep smaller than one turn"
            ),
        ));
    }
    Ok(span)
}
fn resolved(
    entity: SketchEntityId,
    geometry: PlanarGeometry,
    bounds: Option<[f64; 2]>,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    let definition = PlanarCurveDefinition {
        geometry,
        bounds,
        left_offset: 0.0,
    };
    construct::build_curve(&definition)
        .map_err(|cause| SketchCurveFailure::curve(entity, cause))?;
    Ok(ResolvedSketchCurve {
        entity,
        definition,
        endpoint_residuals: None,
        spline_construction: None,
    })
}
