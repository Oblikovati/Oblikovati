// SPDX-License-Identifier: GPL-2.0-only
//! Length inputs for Extrude and shared scalar feature validation.
use super::*;
use oblikovati_model::expression::Expression;

pub(super) fn distance(snapshot: &DocumentSnapshot, source: &str) -> Result<DatumLength, Error> {
    extrusion_length(snapshot, source, "extrude distance")
}
/// Validate an authored extrusion expression. Creation stores compound expressions
/// on a feature-owned scalar so they keep their reads on upstream parameters.
pub(crate) fn extrusion_length(
    snapshot: &DocumentSnapshot,
    source: &str,
    label: &str,
) -> Result<DatumLength, Error> {
    let parameters = &snapshot.parameters;
    if parameters.find(source.trim()).is_some() {
        return positive_length(snapshot, source, label);
    }
    let quantity = parameters
        .evaluate_scalar_expression(source, Dimension::Length)
        .map_err(api_error)?;
    require_positive(quantity.value, quantity.dimension, source, label)?;
    Ok(DatumLength::Constant(quantity.value))
}

pub(crate) fn positive_length(
    snapshot: &DocumentSnapshot,
    source: &str,
    label: &str,
) -> Result<DatumLength, Error> {
    let length = finite_length(snapshot, source, label)?;
    let quantity = length.resolve(&snapshot.parameters).map_err(api_error)?;
    require_positive(quantity, Dimension::Length, source, label)?;
    Ok(length)
}
pub(crate) fn finite_length(
    snapshot: &DocumentSnapshot,
    source: &str,
    label: &str,
) -> Result<DatumLength, Error> {
    let parameters = &snapshot.parameters;
    let text = source.trim();
    if let Some(id) = parameters.find(text) {
        let quantity = parameters.model_value(id).map_err(api_error)?;
        require_finite_length(quantity.value, quantity.dimension, source, label)?;
        return Ok(DatumLength::Parameter(id));
    }
    let expression = Expression::parse(text, parameters.units()).map_err(api_error)?;
    if !expression.references().is_empty() {
        return Err(Error(format!(
            "{label} {source:?}: expected a parameter name or a reference-free length expression"
        )));
    }
    let mut quantity = expression.evaluate().map_err(api_error)?;
    if quantity.dimension == Dimension::Unitless {
        quantity = parameters
            .units()
            .from_preferred(quantity.value, Dimension::Length);
    }
    require_finite_length(quantity.value, quantity.dimension, source, label)?;
    Ok(DatumLength::Constant(quantity.value))
}
/// Check an extrusion expression with the same rules as feature creation.
/// Example: `validate_extrude_distance(&snapshot, "10 mm")?`.
pub fn validate_extrude_distance(snapshot: &DocumentSnapshot, source: &str) -> Result<(), Error> {
    distance(snapshot, source).map(|_| ())
}
/// Evaluate the current distance in database units using the feature's expression rules.
pub fn evaluated_extrude_distance(snapshot: &DocumentSnapshot, source: &str) -> Result<f64, Error> {
    distance(snapshot, source)?
        .resolve(&snapshot.parameters)
        .map_err(api_error)
}
fn require_positive(
    value: f64,
    dimension: Dimension,
    source: &str,
    label: &str,
) -> Result<(), Error> {
    require_finite_length(value, dimension, source, label)?;
    if value <= 0. {
        return Err(Error(format!(
            "{label} {source:?} resolved to {value:?} {dimension:?}: expected a finite positive length"
        )));
    }
    Ok(())
}
fn require_finite_length(
    value: f64,
    dimension: Dimension,
    source: &str,
    label: &str,
) -> Result<(), Error> {
    if dimension != Dimension::Length || !value.is_finite() {
        return Err(Error(format!(
            "{label} {source:?} resolved to {value:?} {dimension:?}: expected a finite length"
        )));
    }
    Ok(())
}
