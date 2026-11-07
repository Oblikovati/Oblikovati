// SPDX-License-Identifier: GPL-2.0-only
//! Certified sketch regions reported with source curve identities rather than native handles.
use super::*;
use martensite_tkbo::profile_extrusion::{ProfileLoop, build_planar_regions};
use oblikovati_model::sketch::SketchEntityId;
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
) -> Result<Vec<SketchDirectedLine>, EvaluationFailure> {
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
            Ok(SketchDirectedLine {
                line: curve.curve,
                reversed: curve.reversed,
            })
        })
        .collect()
}
/// Resolve a current fully constrained sketch to all bounded analytic regions.
/// Example: `certified_sketch_regions(&document, sketch_id, resolution)?` before choosing a profile.
pub fn certified_sketch_regions(
    document: &DocumentSnapshot,
    sketch_id: DocumentSketchId,
    resolution: crate::PrismResolution,
) -> Result<Vec<SketchRegionBoundary>, EvaluationFailure> {
    let resolution = resolution
        .validate()
        .map_err(|cause| Phase::Input.refused(Kind::InvalidInput, Entity::Body, cause))?;
    let sketch = document
        .require_resolved_sketch(sketch_id)
        .map_err(|cause| {
            Phase::Input
                .refused(Kind::InvalidInput, Entity::Body, cause.to_string())
                .with_sketch_identity(sketch_id)
        })?;
    let (profile, _) = input::profile(sketch.sketch.geometry(), sketch.plane)
        .map_err(|cause| cause.with_sketch_identity(sketch_id))?;
    if profile.curves.is_empty() {
        return Ok(Vec::new());
    }
    let regions = build_planar_regions(&profile, resolution.kernel()).map_err(|cause| {
        Phase::Profile
            .operation(Entity::Body, cause)
            .with_sketch_identity(sketch_id)
    })?;
    Ok(regions
        .regions()
        .iter()
        .map(|region| SketchRegionBoundary {
            outer: boundary(region.outer()),
            holes: region.holes().iter().map(boundary).collect(),
        })
        .collect())
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
