// SPDX-License-Identifier: GPL-2.0-only
use super::{
    ResolvedSketchCurve, SketchCurveFailure, SketchIntersectionOptionsV1, SketchLineDomain, shape,
};
use oblikovati_model::sketch::{OwnedSketchGeometry, Sketch2D, SketchEntityId, SketchReadIndex};
pub(super) fn require_request(
    sketch: &Sketch2D,
    line: SketchEntityId,
    options: SketchIntersectionOptionsV1,
) -> Result<(), SketchCurveFailure> {
    sketch
        .line(line)
        .map_err(|cause| SketchCurveFailure::model(line, cause))?;
    super::super::construct::require_options(options.intersection)
        .map_err(|cause| SketchCurveFailure::curve(line, cause))?;
    let limit = options.maximum_endpoint_residual;
    if !limit.is_finite() || limit <= 0.0 {
        return Err(SketchCurveFailure::input(
            line,
            format!("endpoint residual bound {limit}: expected a finite positive length"),
        ));
    }
    require_released_geometry(sketch)
}
fn require_released_geometry(sketch: &Sketch2D) -> Result<(), SketchCurveFailure> {
    let unsupported = sketch
        .splines()
        .iter()
        .map(|curve| curve.id())
        .chain(sketch.offset_splines().iter().map(|curve| curve.id()))
        .min();
    if let Some(entity) = unsupported {
        return Err(super::failure::unreleased_spline(entity));
    }
    Ok(())
}
pub(super) fn targets(
    sketch: &SketchReadIndex<'_>,
    source: SketchEntityId,
    domain: SketchLineDomain,
    limit: f64,
) -> Result<Vec<ResolvedSketchCurve>, SketchCurveFailure> {
    let mut targets = Vec::new();
    for entity in sketch
        .entity_ids()
        .iter()
        .copied()
        .filter(|id| *id != source)
    {
        if matches!(sketch.entity(entity), Ok(OwnedSketchGeometry::Point(_))) {
            continue;
        }
        targets.push(shape::resolve(sketch, entity, domain, limit)?);
    }
    targets.sort_by_key(|target| target.entity);
    Ok(targets)
}
