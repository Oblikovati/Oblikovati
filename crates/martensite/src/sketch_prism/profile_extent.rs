// SPDX-License-Identifier: GPL-2.0-only
//! Native analytic bounds for the same authored curves as region discovery.
use super::*;
use crate::planar_curve::{
    PlanarCurveDefinition, construct,
    sketch::{SketchLineDomain, shape},
};
use martensite_tkgeombase::bnd_lib::Add2dCurve;
use martensite_tkmath::{bnd::BoundingBox2, gp::Point2};
use oblikovati_model::sketch::SketchEntityId;

/// Axis-aligned sketch bounds and their diagonal in document working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchProfileExtent {
    pub minimum: [f64; 2],
    pub maximum: [f64; 2],
    pub diagonal: f64,
}
/// Measure the current profile without a mesh; example: `sketch_profile_extent(&document, sketch)?`.
pub fn sketch_profile_extent(
    document: &DocumentSnapshot,
    sketch_id: DocumentSketchId,
) -> Result<SketchProfileExtent, EvaluationFailure> {
    let sketch = document
        .require_converged_resolved_sketch(sketch_id)
        .map_err(|cause| {
            Phase::Input
                .refused(Kind::InvalidInput, Entity::Body, cause.to_string())
                .with_sketch_identity(sketch_id)
        })?;
    resolved_profile_extent(&sketch)
}
pub(crate) fn resolved_profile_extent(
    sketch: &oblikovati_model::history::ResolvedDocumentSketch<'_>,
) -> Result<SketchProfileExtent, EvaluationFailure> {
    let sketch_id = sketch.sketch.id();
    let geometry = sketch.sketch.geometry();
    let (profile, _) = super::regions::input::profile(geometry, sketch.plane)
        .map_err(|cause| cause.with_sketch_identity(sketch_id))?;
    let index = geometry.read_index();
    let mut bounds = BoundingBox2::default();
    for curve in &profile.curves {
        let id = SketchEntityId(curve.id().0);
        // This query measures the carrier and authored endpoints; actual incidence is checked by
        // the kernel with the operation's explicit sew limit when the region is built.
        let resolved = shape::resolve(&index, id, SketchLineDomain::Segment, f64::MAX)
            .map_err(|cause| extent_failure(sketch_id, cause))?;
        add_curve_bounds(&mut bounds, &resolved.definition)
            .map_err(|cause| extent_failure(sketch_id, cause))?;
    }
    for vertex in &profile.vertices {
        bounds.add_point(Point2::new(vertex.point[0], vertex.point[1]));
    }
    finish_bounds(&bounds).map_err(|cause| extent_failure(sketch_id, cause))
}
fn add_curve_bounds(
    bounds: &mut BoundingBox2,
    definition: &PlanarCurveDefinition,
) -> Result<(), String> {
    let curve = construct::build_curve(definition).map_err(|cause| cause.to_string())?;
    let carrier = curve
        .get()
        .ok_or_else(|| "curve bounds: expected a native analytic curve".to_string())?;
    Add2dCurve::add_optimal(
        &curve,
        carrier.first_parameter(),
        carrier.last_parameter(),
        0.,
        bounds,
    )
    .map_err(|cause| cause.to_string())
}
fn finish_bounds(bounds: &BoundingBox2) -> Result<SketchProfileExtent, String> {
    let [xmin, ymin, xmax, ymax] = bounds.get().map_err(|cause| cause.to_string())?;
    let minimum = [xmin, ymin];
    let maximum = [xmax, ymax];
    let diagonal = (xmax - xmin).hypot(ymax - ymin);
    if !diagonal.is_finite() || diagonal <= 0. {
        return Err(format!(
            "profile bounds {minimum:?}..{maximum:?}, diagonal {diagonal}: expected finite nonzero extent"
        ));
    }
    Ok(SketchProfileExtent {
        minimum,
        maximum,
        diagonal,
    })
}
fn extent_failure(sketch: DocumentSketchId, cause: impl std::fmt::Display) -> EvaluationFailure {
    Phase::Input
        .refused(Kind::InvalidGeometry, Entity::Body, cause.to_string())
        .with_sketch_identity(sketch)
}
#[cfg(test)]
#[path = "../../tests/unit/sketch_profile_extent.rs"]
mod tests;
