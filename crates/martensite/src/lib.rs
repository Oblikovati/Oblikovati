// SPDX-License-Identifier: GPL-2.0-only

//! Release acceptance at the application-owned Martensite boundary.
//! Diagnostic snapshots retain isolated geometry; full evaluation and provenance remain open.
use rustycad_tkbrep::topods::Shape;
use rustycad_tkgeombase::gprop::GProps;
use rustycad_tkprim::brep_prim_api::BRepPrimAPIMakeBox;
use rustycad_tktopalgo::{brep_check::BRepCheckAnalyzer, brep_gprop::BRepGProp};

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
    let valid = BRepCheckAnalyzer::new(shape, true, false, true)
        .and_then(|checker| checker.is_valid())
        .map_err(|error| format!("{context}: expected geometric validity: {error}"))?;
    if !valid {
        return Err(format!("{context}: expected a geometrically valid B-rep"));
    }
    Ok(())
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
    let mut properties = GProps::new();
    // Analytic integration explicitly disables the kernel's triangulation path.
    BRepGProp::volume_properties(shape, &mut properties, true, false, false)
        .map_err(|error| format!("{context}: expected analytic properties: {error}"))?;
    let centroid = properties.centre_of_mass();
    let result = ReleaseProbeProperties {
        volume: properties.mass(),
        centroid: [centroid.x(), centroid.y(), centroid.z()],
    };
    require_finite_result(&result, context)?;
    Ok(result)
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

mod prism;
pub use prism::{
    NamedPrismSnapshot, PrismFaceMesh, PrismFaceProbe, PrismFaceRole, PrismMeshOptions,
    PrismPresentation, PrismProbeDefinition, PrismProbeLoop, PrismProbeReport, PrismProbeSnapshot,
    probe_planar_prism, retain_named_sketch_prism, retain_planar_prism, retain_prism_on_plane,
};
mod sketch_prism;
mod stored_prism;
pub use sketch_prism::{
    MartensitePrismEvaluator, SketchDirectedLine, SketchPrismSelection,
    retain_resolved_sketch_prism, retain_selected_sketch_prism, retain_sketch_prism_definition,
};
pub use stored_prism::retain_stored_sketch_prism;
