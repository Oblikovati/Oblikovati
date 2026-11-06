// SPDX-License-Identifier: GPL-2.0-only
use crate::planar_curve::{PlanarGeometry, construct};
use martensite_tkg2d::geom2d_adaptor::Geom2dAdaptorCurve;
use martensite_tkgeombase::gcpnts::QuasiUniformDeflection;
use martensite_tkmath::geom_abs::Shape;
/// Presentation error and resource policy, separate from modelling tolerance.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchSamplingOptionsV1 {
    pub deflection: f64,
    pub maximum_curve_points: u32,
}
pub(super) fn sample_definition(
    definition: &crate::planar_curve::PlanarCurveDefinition,
    options: SketchSamplingOptionsV1,
) -> Result<Vec<[f64; 2]>, String> {
    require_sampling_capacity(&definition.geometry, options)?;
    let curve = construct::build_curve(definition).map_err(|cause| cause.to_string())?;
    let adaptor = Geom2dAdaptorCurve::with_curve(&curve).map_err(|cause| cause.to_string())?;
    let sampled = QuasiUniformDeflection::with_curve(&adaptor, options.deflection, Shape::C1)
        .map_err(|cause| cause.to_string())?;
    sampled_positions(&sampled, options.maximum_curve_points)
}
pub(super) fn require_sampling_options(options: SketchSamplingOptionsV1) -> Result<(), String> {
    if !options.deflection.is_finite()
        || options.deflection <= 0.
        || !(4..=i32::MAX as u32).contains(&options.maximum_curve_points)
    {
        return Err(format!(
            "sketch sampling {options:?}: expected finite positive deflection and 4..=i32::MAX points per curve"
        ));
    }
    Ok(())
}
fn require_sampling_capacity(
    geometry: &PlanarGeometry,
    options: SketchSamplingOptionsV1,
) -> Result<(), String> {
    let radius = match geometry {
        PlanarGeometry::Circle { radius, .. } => *radius,
        PlanarGeometry::Ellipse { radii, .. } => radii[0],
        _ => return Ok(()),
    };
    let capacity = f64::from(options.maximum_curve_points - 2);
    // A full turn is < 8 radians and 2*acos(1-d) >= sqrt(8*d). This is conservative
    // admission before the native sampler allocates, not a second facet-count policy.
    // Also keep 1-d distinct from 1 in the native acos expression. Ellipses use
    // the same major-radius admission floor; their actual native count is checked below.
    let minimum_ratio = (8. / (capacity * capacity)).max(f64::EPSILON);
    if options.deflection / radius < minimum_ratio {
        return Err(format!(
            "sketch radius {radius}, deflection {}: expected ratio >= {minimum_ratio} with a {} native sample limit",
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
        return Err(format!("sketch sample count {count}: expected 2..={limit}"));
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
            "sketch point {result:?}: expected finite coordinates"
        ));
    }
    Ok(result)
}
