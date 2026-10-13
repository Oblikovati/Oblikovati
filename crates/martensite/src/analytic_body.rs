// SPDX-License-Identifier: GPL-2.0-only
use crate::{ReleaseProbeProperties, require_finite_result};
use rustycad_tkbrep::topods::Shape;
use rustycad_tkgeombase::gprop::GProps;
use rustycad_tktopalgo::brep_gprop::BRepGProp;
// tol:numerical — ADR-0022 bounds eigensolver error, never model incidence.
const PRINCIPAL_RELATIVE_ERROR: f64 = 128. * f64::EPSILON;
/// Analytic B-rep integrals in working units, with unit-density centroidal inertia.
#[derive(Clone, Debug, PartialEq)]
pub struct AnalyticBodyProperties {
    pub geometry: ReleaseProbeProperties,
    pub area: f64,
    pub inertia: [[f64; 3]; 3],
    pub principal_moments: [f64; 3],
    pub principal_axes: [[f64; 3]; 3],
}
pub(crate) fn volume_integrals(shape: &Shape, context: &str) -> Result<GProps, String> {
    crate::checked_shape::volume_properties(shape, context).map_err(|cause| cause.to_string())
}
pub(crate) fn geometry_integrals(
    properties: &GProps,
    context: &str,
) -> Result<ReleaseProbeProperties, String> {
    let centroid = properties.centre_of_mass();
    let result = ReleaseProbeProperties {
        volume: properties.mass(),
        centroid: [centroid.x(), centroid.y(), centroid.z()],
    };
    require_finite_result(&result, context)?;
    Ok(result)
}
pub(crate) fn integrate_body(shape: &Shape) -> Result<AnalyticBodyProperties, String> {
    integrate_nonempty(shape, std::iter::empty())
}
fn completed_integrals(volume: &GProps, area: f64) -> Result<AnalyticBodyProperties, String> {
    let (principal_moments, principal_axes) = principal_integrals(volume)?;
    let result = AnalyticBodyProperties {
        geometry: geometry_integrals(volume, "body properties")?,
        area,
        inertia: inertia_integrals(volume)?,
        principal_moments,
        principal_axes,
    };
    certify_integrals(&result)?;
    Ok(result)
}
fn inertia_integrals(properties: &GProps) -> Result<[[f64; 3]; 3], String> {
    let matrix = properties.matrix_of_inertia();
    let mut rows = [[0.; 3]; 3];
    for (row, row_index) in rows.iter_mut().zip(1..=3) {
        for (value, column_index) in row.iter_mut().zip(1..=3) {
            *value = matrix
                .value(row_index, column_index)
                .map_err(|cause| cause.to_string())?;
        }
    }
    Ok(rows)
}
fn principal_integrals(properties: &GProps) -> Result<([f64; 3], [[f64; 3]; 3]), String> {
    let principal = properties
        .principal_properties()
        .map_err(|cause| cause.to_string())?;
    let (x, y, z) = principal.moments();
    let axes = [
        principal.first_axis_of_inertia(),
        principal.second_axis_of_inertia(),
        principal.third_axis_of_inertia(),
    ];
    let mut pairs = axes
        .map(|axis| canonical_axis([axis.x(), axis.y(), axis.z()]))
        .into_iter()
        .zip([x, y, z])
        .collect::<Vec<_>>();
    pairs.sort_by(|a, b| {
        a.1.total_cmp(&b.1).then_with(|| {
            a.0.iter()
                .zip(b.0)
                .map(|(a, b)| a.total_cmp(&b))
                .find(|order| !order.is_eq())
                .unwrap_or(std::cmp::Ordering::Equal)
        })
    });
    Ok((
        std::array::from_fn(|i| pairs[i].1),
        std::array::from_fn(|i| pairs[i].0),
    ))
}
fn canonical_axis(mut axis: [f64; 3]) -> [f64; 3] {
    let pivot = (0..3)
        .max_by(|&a, &b| {
            axis[a]
                .abs()
                .total_cmp(&axis[b].abs())
                .then_with(|| b.cmp(&a))
        })
        .expect("three coordinates");
    let sign = if axis[pivot].is_sign_negative() {
        -1.
    } else {
        1.
    };
    for value in &mut axis {
        *value = if *value == 0. { 0. } else { *value * sign };
    }
    axis
}
fn certify_integrals(properties: &AnalyticBodyProperties) -> Result<(), String> {
    let values = properties
        .inertia
        .iter()
        .flatten()
        .chain(properties.principal_moments.iter())
        .chain(properties.principal_axes.iter().flatten());
    if properties.geometry.volume <= 0.
        || !properties.area.is_finite()
        || properties.area <= 0.
        || values.into_iter().any(|value| !value.is_finite())
    {
        return Err(format!(
            "body properties {properties:?}: expected finite positive solid integrals"
        ));
    }
    for (moment, axis) in properties
        .principal_moments
        .iter()
        .zip(properties.principal_axes)
    {
        certify_principal(properties.inertia, *moment, axis)?;
    }
    certify_orthogonal(properties.principal_axes)
}
fn certify_principal(inertia: [[f64; 3]; 3], moment: f64, axis: [f64; 3]) -> Result<(), String> {
    let relative = PRINCIPAL_RELATIVE_ERROR;
    let scale = inertia
        .iter()
        .flatten()
        .map(|value| value.abs())
        .fold(moment.abs(), f64::max);
    let norm: f64 = axis.iter().map(|value| value * value).sum();
    let residual = inertia
        .iter()
        .zip(axis)
        .map(|(row, coordinate)| {
            (row.iter().zip(axis).map(|(a, b)| a * b).sum::<f64>() - moment * coordinate).abs()
        })
        .fold(0., f64::max);
    if moment <= 0. || (norm - 1.).abs() > relative || residual > relative * scale {
        return Err(format!(
            "principal moment {moment}, axis {axis:?}, residual {residual}: expected a positive normalized eigenpair within {}",
            relative * scale
        ));
    }
    Ok(())
}

fn certify_orthogonal(axes: [[f64; 3]; 3]) -> Result<(), String> {
    for (left, right) in [(0, 1), (0, 2), (1, 2)] {
        let dot: f64 = axes[left].iter().zip(axes[right]).map(|(a, b)| a * b).sum();
        if dot.abs() > PRINCIPAL_RELATIVE_ERROR {
            return Err(format!(
                "principal axes {axes:?}, dot {dot}: expected an orthogonal basis within {PRINCIPAL_RELATIVE_ERROR}"
            ));
        }
    }
    Ok(())
}
#[cfg(test)]
#[path = "../tests/unit/analytic_body.rs"]
mod tests;

pub(crate) fn integrate_system<'a>(
    shapes: impl IntoIterator<Item = &'a Shape>,
) -> Result<Option<AnalyticBodyProperties>, String> {
    let mut shapes = shapes.into_iter();
    let Some(first) = shapes.next() else {
        return Ok(None);
    };
    integrate_nonempty(first, shapes).map(Some)
}
fn integrate_nonempty<'a>(
    first: &Shape,
    shapes: impl Iterator<Item = &'a Shape>,
) -> Result<AnalyticBodyProperties, String> {
    // Seed with the first integral to preserve the single-body result bit for bit.
    let mut volume = volume_integrals(first, "part body 0")?;
    let mut area = surface_integral(first)?;
    for (index, shape) in shapes.enumerate() {
        let component = volume_integrals(shape, &format!("part body {}", index + 1))?;
        volume.add(&component, 1.).map_err(|cause| {
            format!(
                "part body {}: expected composable analytic integrals: {cause}",
                index + 1
            )
        })?;
        area += surface_integral(shape)?;
    }
    completed_integrals(&volume, area)
}
fn surface_integral(shape: &Shape) -> Result<f64, String> {
    let mut surface = GProps::new();
    BRepGProp::surface_properties(shape, &mut surface, false, false)
        .map_err(|cause| cause.to_string())?;
    Ok(surface.mass())
}
