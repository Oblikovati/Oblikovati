// SPDX-License-Identifier: GPL-2.0-only

//! Release acceptance at the application-owned Martensite boundary.
//! Shapes remain local until immutable evaluation and provenance are established.
use rustycad_tkbrep::topods::Shape;
use rustycad_tkgeombase::gprop::GProps;
use rustycad_tkprim::brep_prim_api::BRepPrimAPIMakeBox;
use rustycad_tktopalgo::{brep_check::BRepCheckAnalyzer, brep_gprop::BRepGProp};

/// Analytic values from a release probe, in the supplied working units.
#[derive(Debug, PartialEq)]
pub struct ReleaseProbeProperties {
    pub volume: f64,
    pub centroid: [f64; 3],
}

/// Check the released modeling packages without sharing their mutable shape handles.
/// Example: `let properties = probe_box_release([2.0, 3.0, 4.0])?;`.
pub fn probe_box_release(extents: [f64; 3]) -> Result<ReleaseProbeProperties, String> {
    require_extents(extents)?;
    let shape = build_probe(extents)?;
    require_valid_probe(&shape, extents)?;
    analytic_probe_properties(&shape, extents)
}

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
    let valid = BRepCheckAnalyzer::new(shape, true, false, true)
        .and_then(|checker| checker.is_valid())
        .map_err(|error| {
            format!("probe extents {extents:?}: expected geometric validity: {error}")
        })?;
    if !valid {
        return Err(format!(
            "probe extents {extents:?}: expected a geometrically valid solid"
        ));
    }
    Ok(())
}

fn analytic_probe_properties(
    shape: &Shape,
    extents: [f64; 3],
) -> Result<ReleaseProbeProperties, String> {
    let mut properties = GProps::new();
    // This probe has analytic surfaces and no mesh; it cannot take the kernel's mesh path.
    BRepGProp::volume_properties(shape, &mut properties, true, false, false).map_err(|error| {
        format!("probe extents {extents:?}: expected analytic properties: {error}")
    })?;
    let centroid = properties.centre_of_mass();
    Ok(ReleaseProbeProperties {
        volume: properties.mass(),
        centroid: [centroid.x(), centroid.y(), centroid.z()],
    })
}

#[cfg(test)]
#[path = "../tests/unit/validity_errors.rs"]
mod validity_errors;

mod planar_spline;
pub use planar_spline::{PlanarSplineDefinition, PlanarSplineSample, evaluate_planar_spline};
