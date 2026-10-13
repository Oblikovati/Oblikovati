// SPDX-License-Identifier: GPL-2.0-only
use super::{
    ResolvedSketchCurve, SketchCurveFailure, SketchIntersectionOptionsV1, SketchLineDomain, shape,
};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
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
        return Err(SketchCurveFailure::input(
            entity,
            format!(
                "spline entity {entity:?}: expected a released native sketch constructor; Martensite issue #9 remains required"
            ),
        ));
    }
    Ok(())
}
pub(super) fn targets(
    sketch: &Sketch2D,
    source: SketchEntityId,
    domain: SketchLineDomain,
    limit: f64,
) -> Result<Vec<ResolvedSketchCurve>, SketchCurveFailure> {
    let lines = sketch
        .lines()
        .iter()
        .filter(|line| line.id != source)
        .map(|line| shape::line(sketch, line.id, domain, limit));
    let circles = sketch
        .circles()
        .iter()
        .map(|circle| shape::circle(sketch, circle.id));
    let arcs = sketch
        .arcs()
        .iter()
        .map(|arc| shape::arc(sketch, arc.id, limit));
    let ellipses = sketch
        .ellipses()
        .iter()
        .map(|ellipse| shape::ellipse(sketch, ellipse.id()));
    let mut targets = lines
        .chain(circles)
        .chain(arcs)
        .chain(ellipses)
        .collect::<Result<Vec<_>, _>>()?;
    targets.sort_by_key(|target| target.entity);
    Ok(targets)
}
