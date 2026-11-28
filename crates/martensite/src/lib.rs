// SPDX-License-Identifier: GPL-2.0-only

//! Release acceptance at the application-owned Martensite boundary.
//! Diagnostic snapshots retain isolated geometry; full evaluation and provenance remain open.
use martensite_tkbrep::topods::Shape;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox;
mod checked_shape;
mod occurrence_line;
pub use occurrence_line::{line_in_occurrence, validate_occurrence_placement};

/// Analytic values from a release probe, in the supplied working units.
#[derive(Clone, Debug, PartialEq)]
pub struct ReleaseProbeProperties {
    pub volume: f64,
    pub centroid: [f64; 3],
}

/// Check the released modeling packages without sharing their mutable shape handles.
/// Example: `let properties = probe_box_release([2.0, 3.0, 4.0])?;`.
pub fn probe_box_release(extents: [f64; 3]) -> Result<ReleaseProbeProperties, String> {
    retain_box_release(extents)?.properties()
}

mod snapshot;
pub use snapshot::{ReleaseProbeSnapshot, retain_box_release};

fn require_extents(extents: [f64; 3]) -> Result<(), String> {
    if extents
        .iter()
        .any(|extent| !extent.is_finite() || *extent <= 0.0)
    {
        return Err(format!(
            "probe extents {extents:?}: expected three finite positive working lengths"
        ));
    }
    Ok(())
}

fn build_probe(extents: [f64; 3]) -> Result<Shape, String> {
    let [width, depth, height] = extents;
    let mut maker = BRepPrimAPIMakeBox::from_sizes(width, depth, height).map_err(|error| {
        format!("probe extents {extents:?}: expected a constructible solid: {error}")
    })?;
    maker.solid().map(Into::into).map_err(|error| {
        format!("probe extents {extents:?}: expected a constructed solid: {error}")
    })
}

fn require_valid_probe(shape: &Shape, extents: [f64; 3]) -> Result<(), String> {
    require_geometric_shape(shape, &format!("probe extents {extents:?}"))
}
fn require_geometric_shape(shape: &Shape, context: &str) -> Result<(), String> {
    checked_shape::require_geometry(shape, context).map_err(|cause| cause.to_string())
}

fn analytic_probe_properties(
    shape: &Shape,
    extents: [f64; 3],
) -> Result<ReleaseProbeProperties, String> {
    analytic_shape_properties(shape, &format!("probe extents {extents:?}"))
}
fn analytic_shape_properties(
    shape: &Shape,
    context: &str,
) -> Result<ReleaseProbeProperties, String> {
    let properties = analytic_body::volume_integrals(shape, context)?;
    analytic_body::geometry_integrals(&properties, context)
}

fn require_finite_result(properties: &ReleaseProbeProperties, context: &str) -> Result<(), String> {
    if !properties.volume.is_finite()
        || properties
            .centroid
            .iter()
            .any(|coordinate| !coordinate.is_finite())
    {
        return Err(format!(
            "{context}, properties {properties:?}: expected finite analytic volume and centroid"
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/unit/validity_errors.rs"]
mod validity_errors;

mod planar_spline;
pub use planar_spline::{PlanarSplineDefinition, PlanarSplineSample, evaluate_planar_spline};

mod evaluation_failure;
pub use evaluation_failure::{
    EvaluationEntity, EvaluationFailure, EvaluationFailureKind, EvaluationPhase,
};
mod prism;
/// Compatibility name for the shared blend candidate.
pub type FilletCandidate = BlendCandidate;
/// Compatibility name for a shared blend history record.
pub type FilletHistoryRecord = BlendHistoryRecord;
/// Compatibility name for a shared named blend result.
pub type FilletNamedTopology = BlendNamedTopology;
pub use prism::{
    BlendCandidate, BlendHistoryRecord, BlendNamedTopology, BodyPresentationSource,
    CapturedNamedSolid, CertifiedNamedSolidEvaluator, CertifiedPrismEvidence,
    CertifiedPrismSnapshot, DisplayTriangle, DrawingBaseProjection, DrawingProjectedCurve,
    DrawingProjection, DrawingProjectionCurveKind, DrawingProjectionFrame, FaceProbe,
    MarkEligibility, NamedEdgeHit, NamedFaceMesh, NamedPrismSnapshot, NamedSolid,
    PlanarProfileLoop, PrismAchievedTolerance, PrismFaceMesh, PrismFaceProbe, PrismFaceRole,
    PrismFaceTopology, PrismHistoryRelation, PrismHistorySource, PrismMeshOptions,
    PrismPresentation, PrismPresentationSource, PrismProbeDefinition, PrismProbeLoop,
    PrismProbeReport, PrismProbeSnapshot, PrismResolution, PrismResolutionPolicy,
    PrismSubentityHistory, PrismSubentityProvenance, PrismSubentityRole, PrismSurfaceKind,
    PrismTopologySummary, RigidMotion, SharedFilletVertex, SolidFilletEdgeGroups,
    ThreadableFaceInfo, TopologySummary, capture_full_named_candidate, check_mark_eligibility,
    complete_named_candidate, complete_named_checkpoints, drawing_base_view_bodies,
    drawing_base_view_solids, drawing_projection_bodies, drawing_projection_solids,
    finish_appearance_rgb_u8, finish_appearance_rgba, named_part_properties,
    named_solid_part_properties, prepare_dependent_named_publication,
    prepare_full_named_publication, probe_planar_prism, retain_certified_named_sketch_prism,
    retain_certified_profile_prism, retain_certified_profile_prisms, retain_named_sketch_prism,
    retain_planar_prism, retain_prism_on_plane, validate_full_named_snapshot,
};
mod loft;
pub use loft::check_loft_guide_closure;
mod boundary_patch;
pub mod coil;
pub use boundary_patch::{chain_sketch_boundary_loop, is_interior_diagonal_curve};
mod decal;
mod derive;
mod emboss;
pub mod import;
mod revolve;
mod rib;
pub mod sweep;
mod unwrap;
pub use revolve::{
    RevolveFaceRole, RevolveProbeDefinition, RevolveProbeReport, probe_planar_revolve,
    probe_revolve_on_plane,
};
mod region_identity;
mod sketch_prism;
pub use region_identity::{
    RegionSelectionError, equivalent_region_boundary, resolve_region_by_boundary,
};
mod stored_prism;
pub use sketch_prism::{
    ConditionalProfileLine, ConditionalProfileMapping, ConditionalProfileVertex,
    SketchDirectedCurve, SketchPrismSelection, SketchProfileExtent, SketchRegionBoundary,
    SketchRegionCurveUse, SketchRegionMeasure, certified_sketch_region_measures,
    certified_sketch_regions, certified_sketch_regions_with_datums,
    certified_sketch_regions_with_policy, conditional_profile_mapping, offset_sketch_profile,
    pick_sketch_region, pick_sketch_region_index, retain_certified_sketch_prism,
    retain_resolved_sketch_prism, retain_selected_sketch_prism, retain_sketch_prism_definition,
    sketch_profile_extent,
};
pub use stored_prism::retain_stored_sketch_prism;

mod analytic_body;
pub use analytic_body::AnalyticBodyProperties;

/// Immutable analytic curve inputs and checked intersection reports.
pub mod planar_curve;
pub mod sketch_region;

mod sketch_projection;
pub use sketch_projection::{SKETCH_PROJECTION_MIN_NORMAL_COSINE, sketch_projection};
