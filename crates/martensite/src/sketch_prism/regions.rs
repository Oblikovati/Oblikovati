// SPDX-License-Identifier: GPL-2.0-only
//! Certified sketch regions reported with source curve identities rather than native handles.
use super::*;
use martensite_tkbo::profile_extrusion::{
    PlanarRegion, PlanarRegionSet, ProfileLoop, build_planar_regions,
};
use martensite_tkg3d::top_abs::State;
use martensite_tkmath::gp::Point3;
use martensite_tktopalgo::brep_class::BRepClassFaceClassifier;
use oblikovati_model::{
    feature::work_geometry::WorkGeometryEvaluation, history::ResolvedDocumentSketch,
    sketch::SketchEntityId,
};
#[path = "region_input.rs"]
pub(super) mod input;

/// One directed use of an authored sketch curve in a region boundary.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchRegionCurveUse {
    pub curve: SketchEntityId,
    pub reversed: bool,
}
/// One current bounded region; the curve lineage can be stored instead of its transient index.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchRegionBoundary {
    pub outer: Vec<SketchRegionCurveUse>,
    pub holes: Vec<Vec<SketchRegionCurveUse>>,
}
impl SketchRegionBoundary {
    /// Capture its complete directed lineage. Example: `region.selection()` before storing tool inputs.
    pub fn selection(&self) -> SketchPrismSelection {
        let directed = |boundary: &[SketchRegionCurveUse]| {
            boundary
                .iter()
                .map(|item| SketchDirectedCurve {
                    curve: item.curve,
                    reversed: item.reversed,
                })
                .collect()
        };
        SketchPrismSelection {
            outer: directed(&self.outer),
            holes: self.holes.iter().map(|hole| directed(hole)).collect(),
        }
    }

    /// Convert a line-bounded region to a saved prism selection; example: `region.line_selection(sketch)?`.
    pub fn line_selection(
        &self,
        sketch: &oblikovati_model::sketch::Sketch2D,
    ) -> Result<SketchPrismSelection, EvaluationFailure> {
        let index = sketch.read_index();
        Ok(SketchPrismSelection {
            outer: line_uses(sketch, &index, &self.outer)?,
            holes: self
                .holes
                .iter()
                .map(|hole| line_uses(sketch, &index, hole))
                .collect::<Result<_, _>>()?,
        })
    }
}
fn line_uses(
    sketch: &oblikovati_model::sketch::Sketch2D,
    index: &oblikovati_model::sketch::SketchReadIndex<'_>,
    curves: &[SketchRegionCurveUse],
) -> Result<Vec<SketchDirectedCurve>, EvaluationFailure> {
    curves
        .iter()
        .map(|curve| {
            let geometry = index
                .entity(curve.curve)
                .map_err(|cause| Phase::Input.sketch(Entity::SketchEntity(curve.curve.0), cause))?;
            if !matches!(
                geometry,
                oblikovati_model::sketch::OwnedSketchGeometry::Line(_)
            ) {
                return Err(Phase::Input.refused(
                    Kind::Unsupported,
                    Entity::SketchEntity(curve.curve.0),
                    format!(
                        "region curve {:?}: expected a line for the stored prism recipe",
                        curve.curve
                    ),
                ));
            }
            boundary::require_boundary_line(sketch, curve.curve)?;
            Ok(SketchDirectedCurve {
                curve: curve.curve,
                reversed: curve.reversed,
            })
        })
        .collect()
}
/// Resolve a current converged sketch to all bounded analytic regions.
/// Example: `certified_sketch_regions(&document, sketch_id, resolution)?` before choosing a profile.
pub fn certified_sketch_regions(
    document: &DocumentSnapshot,
    sketch_id: DocumentSketchId,
    resolution: crate::PrismResolution,
) -> Result<Vec<SketchRegionBoundary>, EvaluationFailure> {
    certified_sketch_regions_with_policy(
        document,
        sketch_id,
        crate::PrismResolutionPolicy::Fixed(resolution),
    )
}
/// Discover profiles with the same per-sketch policy as certified replay.
/// Example: `certified_sketch_regions_with_policy(&document, sketch_id, policy)?`.
pub fn certified_sketch_regions_with_policy(
    document: &DocumentSnapshot,
    sketch_id: DocumentSketchId,
    policy: crate::PrismResolutionPolicy,
) -> Result<Vec<SketchRegionBoundary>, EvaluationFailure> {
    map_regions(
        resolve_sketch(document, sketch_id, None)?,
        policy,
        region_boundary,
    )
}

/// Discover profiles on a sketch supported by current named Part geometry.
/// Example: `certified_sketch_regions_with_datums(document, id, policy, datums)?`.
pub fn certified_sketch_regions_with_datums(
    document: &DocumentSnapshot,
    sketch_id: DocumentSketchId,
    policy: crate::PrismResolutionPolicy,
    datums: &WorkGeometryEvaluation,
) -> Result<Vec<SketchRegionBoundary>, EvaluationFailure> {
    map_regions(
        resolve_sketch(document, sketch_id, Some(datums))?,
        policy,
        region_boundary,
    )
}

pub(crate) fn region_boundary(region: &PlanarRegion) -> SketchRegionBoundary {
    SketchRegionBoundary {
        outer: boundary(region.outer()),
        holes: region.holes().iter().map(boundary).collect(),
    }
}

/// Pick an unambiguous analytic profile, including its boundary.
/// Example: `pick_sketch_region(sketch, resolution, view_line)?` while selecting a profile.
pub fn pick_sketch_region(
    sketch: ResolvedDocumentSketch<'_>,
    policy: crate::PrismResolutionPolicy,
    [origin, direction]: [[f64; 3]; 2],
) -> Result<Option<SketchPrismSelection>, EvaluationFailure> {
    let failed = |cause| Phase::Profile.refused(Kind::InvalidInput, Entity::Body, cause);
    let Some(point) = crate::sketch_projection(
        sketch.plane,
        origin,
        direction,
        crate::SKETCH_PROJECTION_MIN_NORMAL_COSINE,
    )
    .map_err(failed)?
    else {
        return Ok(None);
    };
    let [x, y, z] = sketch
        .plane
        .to_model(point)
        .map_err(|cause| failed(cause.to_string()))?;
    let resolution = policy.for_sketch(&sketch)?;
    let Some(regions) = resolved_regions(sketch, policy)? else {
        return Ok(None);
    };
    classify_region(&regions, Point3::new(x, y, z), resolution.weld)
}

fn classify_region(
    regions: &PlanarRegionSet,
    point: Point3,
    tolerance: f64,
) -> Result<Option<SketchPrismSelection>, EvaluationFailure> {
    let mut hit = None;
    for region in regions.regions() {
        let classifier = BRepClassFaceClassifier::with_face_point3d(
            region.face(),
            &point,
            tolerance,
            false,
            tolerance,
        )
        .map_err(|cause| {
            Phase::Profile.native(Entity::Body, "classify sketch profile".into(), cause)
        })?;
        if matches!(classifier.state(), State::In | State::On) {
            if hit.is_some() {
                return Ok(None);
            }
            hit = Some(region_boundary(region).selection());
        }
    }
    Ok(hit)
}

/// Area in database square centimetres and hole count of a certified bounded region.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchRegionMeasure {
    pub area: f64,
    pub holes: usize,
}

/// Measure the regions in the order used by certified extrusion selection.
/// Example: `certified_sketch_region_measures(&document, sketch_id, policy)?`.
pub fn certified_sketch_region_measures(
    document: &DocumentSnapshot,
    sketch_id: DocumentSketchId,
    policy: crate::PrismResolutionPolicy,
) -> Result<Vec<SketchRegionMeasure>, EvaluationFailure> {
    map_regions(
        resolve_sketch(document, sketch_id, None)?,
        policy,
        |region| SketchRegionMeasure {
            area: region.area().value,
            holes: region.holes().len(),
        },
    )
}

fn resolve_sketch<'a>(
    document: &'a DocumentSnapshot,
    sketch_id: DocumentSketchId,
    datums: Option<&WorkGeometryEvaluation>,
) -> Result<ResolvedDocumentSketch<'a>, EvaluationFailure> {
    match datums {
        Some(datums) => document.require_converged_resolved_sketch_with_datums(sketch_id, datums),
        None => document.require_converged_resolved_sketch(sketch_id),
    }
    .map_err(|cause| {
        Phase::Input
            .refused(Kind::InvalidInput, Entity::Body, cause.to_string())
            .with_sketch_identity(sketch_id)
    })
}

fn map_regions<T>(
    sketch: ResolvedDocumentSketch<'_>,
    policy: crate::PrismResolutionPolicy,
    project: impl FnMut(&PlanarRegion) -> T,
) -> Result<Vec<T>, EvaluationFailure> {
    Ok(
        resolved_regions(sketch, policy)?.map_or_else(Vec::new, |regions| {
            regions.regions().iter().map(project).collect()
        }),
    )
}

pub(crate) fn resolved_regions(
    sketch: ResolvedDocumentSketch<'_>,
    policy: crate::PrismResolutionPolicy,
) -> Result<Option<PlanarRegionSet>, EvaluationFailure> {
    let sketch_id = sketch.sketch.id();
    let policy = policy.validate().map_err(|cause| {
        Phase::Input
            .refused(Kind::InvalidInput, Entity::Body, cause)
            .with_sketch_identity(sketch_id)
    })?;
    let (profile, _) = input::profile(sketch.sketch.geometry(), sketch.plane)
        .map_err(|cause| cause.with_sketch_identity(sketch_id))?;
    if profile.curves.is_empty() {
        return Ok(None);
    }
    let resolution = policy.for_sketch(&sketch)?;
    let regions = build_planar_regions(&profile, resolution.kernel()).map_err(|cause| {
        Phase::Profile
            .operation(Entity::Body, cause)
            .with_sketch_identity(sketch_id)
    })?;
    Ok(Some(regions))
}
fn boundary(loop_: &ProfileLoop) -> Vec<SketchRegionCurveUse> {
    loop_
        .uses
        .iter()
        .map(|use_| SketchRegionCurveUse {
            curve: SketchEntityId(use_.curve.0),
            reversed: use_.reversed,
        })
        .collect()
}
