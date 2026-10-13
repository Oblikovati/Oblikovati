// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchLineDomain, shape};
use crate::planar_curve::{PlanarGeometry, construct};
use oblikovati_model::sketch::{
    Sketch2D, SketchEntityId, SketchPathMember, SketchPlacementMember, SketchPlacementRecipe,
};
use rustycad_tkg2d::geom2d_adaptor::Geom2dAdaptorCurve;
use rustycad_tkgeombase::gcpnts::QuasiUniformDeflection;
use rustycad_tkmath::geom_abs::Shape;
mod local_model;
/// Presentation error and resource policy, separate from modelling tolerance.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PlacementSamplingOptionsV1 {
    pub deflection: f64,
    pub maximum_curve_points: u32,
}
/// One sampled member in recipe order; these coordinates never drive model decisions.
#[derive(Clone, Debug, PartialEq)]
pub struct PlacementPreviewCurve {
    pub member_index: usize,
    pub construction: bool,
    pub points: Vec<[f64; 2]>,
}
/// Sample the same authoring graph used by commit, in a private temporary namespace.
/// Example: `sample_placement_preview(&recipe, options)?` for a transient outline.
pub fn sample_placement_preview(
    recipe: &SketchPlacementRecipe,
    options: PlacementSamplingOptionsV1,
) -> Result<Vec<PlacementPreviewCurve>, String> {
    require_sampling_options(options)?;
    let (geometry, members) = local_model::build(recipe)?;
    recipe
        .members
        .iter()
        .zip(members)
        .enumerate()
        .map(|(member_index, (member, id))| {
            sample_member(&geometry, id, *member, member_index, options)
                .map_err(|cause| format!("preview member {member_index}: {cause}"))
        })
        .collect()
}
fn sample_member(
    geometry: &Sketch2D,
    id: SketchEntityId,
    member: SketchPlacementMember,
    member_index: usize,
    options: PlacementSamplingOptionsV1,
) -> Result<PlacementPreviewCurve, String> {
    let resolved = match member.path {
        SketchPathMember::Line(_) => {
            shape::line(geometry, id, SketchLineDomain::Segment, options.deflection)
        }
        SketchPathMember::Arc { .. } => shape::arc(geometry, id, options.deflection),
    }
    .map_err(|cause| cause.to_string())?;
    require_sampling_capacity(&resolved.definition.geometry, options)?;
    Ok(PlacementPreviewCurve {
        member_index,
        construction: member.construction,
        points: sample_definition(&resolved.definition, options)?,
    })
}
fn sample_definition(
    definition: &crate::planar_curve::PlanarCurveDefinition,
    options: PlacementSamplingOptionsV1,
) -> Result<Vec<[f64; 2]>, String> {
    let curve = construct::build_curve(definition).map_err(|cause| cause.to_string())?;
    let adaptor = Geom2dAdaptorCurve::with_curve(&curve).map_err(|cause| cause.to_string())?;
    let sampled = QuasiUniformDeflection::with_curve(&adaptor, options.deflection, Shape::C1)
        .map_err(|cause| cause.to_string())?;
    sampled_positions(&sampled, options.maximum_curve_points)
}
fn require_sampling_options(options: PlacementSamplingOptionsV1) -> Result<(), String> {
    if !options.deflection.is_finite()
        || options.deflection <= 0.
        || !(4..=i32::MAX as u32).contains(&options.maximum_curve_points)
    {
        return Err(format!(
            "placement sampling {options:?}: expected finite positive deflection and 4..=i32::MAX points per curve"
        ));
    }
    Ok(())
}
fn require_sampling_capacity(
    geometry: &PlanarGeometry,
    options: PlacementSamplingOptionsV1,
) -> Result<(), String> {
    let PlanarGeometry::Circle { radius, .. } = geometry else {
        return Ok(());
    };
    let capacity = f64::from(options.maximum_curve_points - 2);
    // A full turn is < 8 radians and 2*acos(1-d) >= sqrt(8*d). This is conservative
    // admission before the native sampler allocates, not a second facet-count policy.
    // Also keep 1-d distinct from 1 in the native acos expression.
    let minimum_ratio = (8. / (capacity * capacity)).max(f64::EPSILON);
    if options.deflection / radius < minimum_ratio {
        return Err(format!(
            "preview radius {radius}, deflection {}: expected ratio >= {minimum_ratio} for at most {} native samples",
            options.deflection, options.maximum_curve_points
        ));
    }
    Ok(())
}
fn sampled_positions(
    sampled: &QuasiUniformDeflection,
    limit: u32,
) -> Result<Vec<[f64; 2]>, String> {
    let count = sampled.nb_points().map_err(|cause| cause.to_string())?;
    if count < 2 || i64::from(count) > i64::from(limit) {
        return Err(format!(
            "preview sample count {count}: expected 2..={limit}"
        ));
    }
    (1..=count)
        .map(|index| sampled_position(sampled, index))
        .collect()
}
fn sampled_position(sampled: &QuasiUniformDeflection, index: i32) -> Result<[f64; 2], String> {
    let point = sampled.value(index).map_err(|cause| cause.to_string())?;
    let result = [point.x(), point.y()];
    if result.iter().any(|value| !value.is_finite()) {
        return Err(format!(
            "preview point {result:?}: expected finite coordinates"
        ));
    }
    Ok(result)
}
