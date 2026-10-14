// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PlanarCurveDefinition, PlanarIntersectionOptions, PlanarIntersectionReport,
    intersect_planar_curves,
};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
mod failure;
mod prepare;
mod shape;
pub use failure::{SketchCurveFailure, SketchCurveFailureCause};
/// Go trim uses finite lines; extension uses infinite supports for all line operands.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchLineDomain {
    Segment,
    Support,
}
/// Explicit tolerances for native intersections and conversion of authored endpoints.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchIntersectionOptionsV1 {
    pub intersection: PlanarIntersectionOptions,
    pub maximum_endpoint_residual: f64,
}
/// Owned analytic curve in source coordinates, with measured authored endpoint errors.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedSketchCurve {
    pub entity: SketchEntityId,
    pub definition: PlanarCurveDefinition,
    /// Line/arc endpoint distances to native evaluation; absent for full conics.
    pub endpoint_residuals: Option<[f64; 2]>,
}
/// All reported incidences against one target; overlaps and empty results are retained.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchCurveIncidences {
    pub target: ResolvedSketchCurve,
    pub intersection: PlanarIntersectionReport,
}
/// A read-only query, not a trim decision or a proof of root completeness.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchLineIncidences {
    pub source: ResolvedSketchCurve,
    pub targets: Vec<SketchCurveIncidences>,
    pub line_domain: SketchLineDomain,
    pub options: SketchIntersectionOptionsV1,
}
/// Resolve every curve before querying the selected line; no unsupported curve is silently omitted.
/// Example: `intersect_sketch_line(&sketch, line, SketchLineDomain::Support, options)?;`.
pub fn intersect_sketch_line(
    sketch: &Sketch2D,
    line: SketchEntityId,
    domain: SketchLineDomain,
    options: SketchIntersectionOptionsV1,
) -> Result<SketchLineIncidences, SketchCurveFailure> {
    prepare::require_request(sketch, line, options)?;
    let index = sketch.read_index();
    let source = shape::resolve(&index, line, domain, options.maximum_endpoint_residual)?;
    let targets = prepare::targets(&index, line, domain, options.maximum_endpoint_residual)?;
    let targets = targets
        .into_iter()
        .map(|target| intersect_target(&source, target, options))
        .collect::<Result<Vec<_>, _>>()?;
    Ok(SketchLineIncidences {
        source,
        targets,
        line_domain: domain,
        options,
    })
}
fn intersect_target(
    source: &ResolvedSketchCurve,
    target: ResolvedSketchCurve,
    options: SketchIntersectionOptionsV1,
) -> Result<SketchCurveIncidences, SketchCurveFailure> {
    let intersection =
        intersect_planar_curves(&source.definition, &target.definition, options.intersection)
            .map_err(|cause| SketchCurveFailure::curve(target.entity, cause))?;
    Ok(SketchCurveIncidences {
        target,
        intersection,
    })
}

mod preview;
pub use preview::{
    PlacementPreviewMember, sample_placement_preview, sample_placement_preview_with_policy,
    sample_sketch_preview_members,
};
pub use sampling::SketchSamplingOptionsV1;
mod presentation;
mod sampling;
pub use presentation::{
    SketchDisplayDecline, SketchDisplayEntity, SketchDisplayPrimitive, SketchDisplayReport,
    sample_sketch_display,
};
