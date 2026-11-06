// SPDX-License-Identifier: GPL-2.0-only
//! Measurements of authored sketch curves on the exact definitions; no display sample is read.
use super::{SketchCurveFailure, SketchLineDomain, shape};
use crate::planar_curve::{PlanarGeometry, construct};
use martensite_tkg2d::{adaptor2d::Adaptor2dCurve2d, geom2d_adaptor::Geom2dAdaptorCurve};
use martensite_tkgeomalgo::geom2d_api::ExtremaCurveCurve;
use martensite_tkgeombase::{extrema::ExtPC2d, gcpnts::AbscissaPoint};
use martensite_tkmath::gp::Point2;
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};

/// What one curve measures by itself; example: `measure.length` for a selected line.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchCurveMeasure {
    pub length: f64,
    pub endpoints: Option<[[f64; 2]; 2]>,
    pub circular: Option<SketchCircularMeasure>,
}
/// Center and size of a circle or arc; `sweep` is in radians.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchCircularMeasure {
    pub center: [f64; 2],
    pub radius: f64,
    pub sweep: f64,
}
/// One extreme separation and where it occurs on the two operands.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchDistanceWitness {
    pub distance: f64,
    pub points: [[f64; 2]; 2],
}
/// The nearest and farthest separation of two measured operands.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchDistanceRange {
    pub minimum: SketchDistanceWitness,
    pub maximum: SketchDistanceWitness,
}
/// Tolerances only reject a wrongly shaped input; a measurement never moves geometry.
const NATIVE_TOLERANCE: f64 = 1e-9; // tol:native extrema convergence, in working lengths
const ENDPOINT_LIMIT: f64 = 1e-6; // tol:weld authored endpoints against their analytic curve

/// Measure one curve; example: `measure_sketch_curve(&sketch, line)?.length`.
pub fn measure_sketch_curve(
    sketch: &Sketch2D,
    entity: SketchEntityId,
) -> Result<SketchCurveMeasure, SketchCurveFailure> {
    let resolved = resolve_sketch_curve(sketch, entity)?;
    let curve = construct::build_curve(&resolved).map_err(|cause| curve_failure(entity, cause))?;
    let adaptor = Geom2dAdaptorCurve::with_curve(&curve)
        .map_err(|cause| curve_failure(entity, cause.into()))?;
    let length =
        AbscissaPoint::length(&adaptor).map_err(|cause| curve_failure(entity, cause.into()))?;
    let ends = curve_ends(&adaptor);
    Ok(SketchCurveMeasure {
        length,
        // An unbounded basis has no authored ends, and a full circle's two ends are one point.
        endpoints: ends.filter(|_| resolved.bounds.is_some()),
        circular: circular_measure(&resolved.geometry, resolved.bounds),
    })
}
fn curve_ends(curve: &Geom2dAdaptorCurve) -> Option<[[f64; 2]; 2]> {
    let [first, last] = parameter_span(curve)?;
    Some([position(curve, first)?, position(curve, last)?])
}
fn parameter_span(curve: &Geom2dAdaptorCurve) -> Option<[f64; 2]> {
    Some([curve.first_parameter().ok()?, curve.last_parameter().ok()?])
}
fn circular_measure(
    geometry: &PlanarGeometry,
    bounds: Option<[f64; 2]>,
) -> Option<SketchCircularMeasure> {
    let PlanarGeometry::Circle { frame, radius } = geometry else {
        return None;
    };
    Some(SketchCircularMeasure {
        center: frame.center,
        radius: *radius,
        sweep: bounds.map_or(std::f64::consts::TAU, |[first, last]| last - first),
    })
}
/// The separation range of two curves; example: min and max between two segments.
pub fn sketch_curve_distance_range(
    sketch: &Sketch2D,
    entities: [SketchEntityId; 2],
) -> Result<SketchDistanceRange, SketchCurveFailure> {
    let definitions = entities.map(|entity| resolve_sketch_curve(sketch, entity));
    let [first, second] = definitions;
    let built = [
        construct::build_curve(&first?).map_err(|cause| curve_failure(entities[0], cause))?,
        construct::build_curve(&second?).map_err(|cause| curve_failure(entities[1], cause))?,
    ];
    let adaptors = [
        Geom2dAdaptorCurve::with_curve(&built[0])
            .map_err(|cause| curve_failure(entities[0], cause.into()))?,
        Geom2dAdaptorCurve::with_curve(&built[1])
            .map_err(|cause| curve_failure(entities[1], cause.into()))?,
    ];
    let [left, right] = &adaptors;
    let mut candidates = endpoint_candidates(left, right, false);
    candidates.extend(endpoint_candidates(right, left, true));
    candidates.extend(interior_candidates(&built, &adaptors));
    distance_range(candidates).ok_or_else(|| {
        SketchCurveFailure::input(
            entities[0],
            format!("curves {entities:?}: expected a finite separation"),
        )
    })
}
/// The separation range of a point and a curve; example: a line's midpoint against another line.
pub fn sketch_point_curve_distance_range(
    sketch: &Sketch2D,
    point: [f64; 2],
    entity: SketchEntityId,
) -> Result<SketchDistanceRange, SketchCurveFailure> {
    let definition = resolve_sketch_curve(sketch, entity)?;
    let curve =
        construct::build_curve(&definition).map_err(|cause| curve_failure(entity, cause))?;
    let adaptor = Geom2dAdaptorCurve::with_curve(&curve)
        .map_err(|cause| curve_failure(entity, cause.into()))?;
    let candidates = point_candidates(&adaptor, point)
        .into_iter()
        .map(|on_curve| witness(point, on_curve))
        .collect();
    distance_range(candidates).ok_or_else(|| {
        SketchCurveFailure::input(
            entity,
            format!("point {point:?}: expected a finite separation from curve {entity:?}"),
        )
    })
}
/// The exact definition of one authored curve; example: for a kernel edge built from it.
pub fn resolve_sketch_curve(
    sketch: &Sketch2D,
    entity: SketchEntityId,
) -> Result<crate::planar_curve::PlanarCurveDefinition, SketchCurveFailure> {
    let index = sketch.read_index();
    shape::resolve(&index, entity, SketchLineDomain::Segment, ENDPOINT_LIMIT)
        .map(|resolved| resolved.definition)
}
fn curve_failure(
    entity: SketchEntityId,
    cause: crate::planar_curve::PlanarCurveFailure,
) -> SketchCurveFailure {
    SketchCurveFailure::curve(entity, cause)
}
fn position(curve: &Geom2dAdaptorCurve, parameter: f64) -> Option<[f64; 2]> {
    let point = curve.value(parameter).ok()?;
    Some([point.x(), point.y()])
}
/// Each end of the first curve against the second: its own ends and stationary points.
fn endpoint_candidates(
    from: &Geom2dAdaptorCurve,
    onto: &Geom2dAdaptorCurve,
    swapped: bool,
) -> Vec<SketchDistanceWitness> {
    let mut found = Vec::new();
    for end in curve_ends(from).into_iter().flatten() {
        for on_other in point_candidates(onto, end) {
            found.push(if swapped {
                witness(on_other, end)
            } else {
                witness(end, on_other)
            });
        }
    }
    found
}
fn point_candidates(curve: &Geom2dAdaptorCurve, point: [f64; 2]) -> Vec<[f64; 2]> {
    let mut found: Vec<[f64; 2]> = curve_ends(curve).into_iter().flatten().collect();
    let Some([first, last]) = parameter_span(curve) else {
        return found;
    };
    let query = Point2::new(point[0], point[1]);
    if let Ok(extrema) = ExtPC2d::with_range(query, curve, first, last, NATIVE_TOLERANCE)
        && extrema.is_done()
    {
        for index in 1..=extrema.nb_ext().unwrap_or(0) {
            if let Ok(hit) = extrema.point(index) {
                let at = hit.value();
                found.push([at.x(), at.y()]);
            }
        }
    }
    found
}
/// Stationary pairs in both interiors; parallel segments report none and fall to their ends.
fn interior_candidates(
    built: &[construct::CurveHandle; 2],
    curves: &[Geom2dAdaptorCurve; 2],
) -> Vec<SketchDistanceWitness> {
    let (Some(first), Some(second)) = (parameter_span(&curves[0]), parameter_span(&curves[1]))
    else {
        return Vec::new();
    };
    let Ok(extrema) = ExtremaCurveCurve::new(
        &built[0], &built[1], first[0], first[1], second[0], second[1],
    ) else {
        return Vec::new();
    };
    (1..=extrema.nb_extrema())
        .filter_map(|index| extrema.points(index).ok())
        .map(|(on_first, on_second)| {
            witness([on_first.x(), on_first.y()], [on_second.x(), on_second.y()])
        })
        .collect()
}
fn witness(first: [f64; 2], second: [f64; 2]) -> SketchDistanceWitness {
    SketchDistanceWitness {
        distance: (first[0] - second[0]).hypot(first[1] - second[1]),
        points: [first, second],
    }
}
/// Ties keep the earliest candidate, so the result is the same on every run.
fn distance_range(candidates: Vec<SketchDistanceWitness>) -> Option<SketchDistanceRange> {
    let finite = || candidates.iter().filter(|found| found.distance.is_finite());
    let minimum = finite().copied().reduce(|best, next| {
        if next.distance < best.distance {
            next
        } else {
            best
        }
    })?;
    let maximum = finite().copied().reduce(|best, next| {
        if next.distance > best.distance {
            next
        } else {
            best
        }
    })?;
    Some(SketchDistanceRange { minimum, maximum })
}

#[cfg(test)]
#[path = "../../../tests/planar_curve/measure.rs"]
mod tests;
