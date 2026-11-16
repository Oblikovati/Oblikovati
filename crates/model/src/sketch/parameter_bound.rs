// SPDX-License-Identifier: GPL-2.0-only
//! Conditional bounds from a restricted set of authored length sources.
use super::interval_equations::enclosure::Enclosure;
use crate::{parameter::ParameterCollection, quantity::Dimension};
use oblikovati_api::types::{MODEL_PARAM, MODEL_VALUE_NOMINAL, USER_PARAM};
use std::collections::BTreeSet;

/// An outward-rounded authored length and its source chain, conditional on arithmetic conformance.
#[derive(Clone, Debug, PartialEq)]
pub struct ConditionalParameterBound {
    id: u64,
    bounds: [f64; 2],
    sources: Vec<(u64, String)>,
}
impl ConditionalParameterBound {
    /// Read the requested parameter identity; example: `bound.id()`.
    pub fn id(&self) -> u64 {
        self.id
    }
    /// Read bounds in centimetre working units; example: `bound.bounds()`.
    pub fn bounds(&self) -> [f64; 2] {
        self.bounds
    }
    /// Read the source chain from requested parameter to literal; example: `bound.sources()`.
    pub fn sources(&self) -> &[(u64, String)] {
        &self.sources
    }
}
/// Enclose an authored decimal length or direct alias with nominal model values.
/// Example: `conditional_parameter_bound(&parameters, size_id)?`.
/// The bound relies on interval arithmetic conformance; it does not certify a solve.
pub fn conditional_parameter_bound(
    parameters: &ParameterCollection<u64>,
    id: u64,
) -> Result<ConditionalParameterBound, String> {
    if parameters.units().working_scale().to_bits() != 1.0_f64.to_bits() {
        return Err(format!(
            "parameter {id}: expected centimetre working scale 1.0"
        ));
    }
    let mut seen = BTreeSet::new();
    let mut sources = Vec::new();
    let enclosure = source_bound(parameters, id, &mut seen, &mut sources)?;
    Ok(ConditionalParameterBound {
        id,
        bounds: enclosure.bounds(),
        sources,
    })
}
fn source_bound(
    parameters: &ParameterCollection<u64>,
    id: u64,
    seen: &mut BTreeSet<u64>,
    sources: &mut Vec<(u64, String)>,
) -> Result<Enclosure, String> {
    if !seen.insert(id) {
        return Err(format!("parameter {id}: expected an acyclic source chain"));
    }
    let state = parameters
        .get(id)
        .ok_or_else(|| format!("parameter {id}: expected an existing source"))?;
    if !matches!(state.kind, MODEL_PARAM | USER_PARAM)
        || state.model_value_type != MODEL_VALUE_NOMINAL
    {
        return Err(format!(
            "parameter {id}: expected an authored nominal model or user parameter"
        ));
    }
    let value = state
        .current_value()
        .filter(|value| value.dimension == Dimension::Length && value.value.is_finite())
        .ok_or_else(|| format!("parameter {id}: expected a healthy finite length"))?;
    let source = state.source.trim();
    sources.push((id, state.source.clone()));
    let value_bound = match parameters.find(source) {
        Some(driver) => alias_bound(parameters, id, driver, seen, sources)?,
        None => literal_bound(parameters, id, source)?,
    };
    value_bound.include(value.value)
}
fn alias_bound(
    parameters: &ParameterCollection<u64>,
    id: u64,
    driver: u64,
    seen: &mut BTreeSet<u64>,
    sources: &mut Vec<(u64, String)>,
) -> Result<Enclosure, String> {
    let drivers = parameters.drivers(id).map_err(|e| e.to_string())?;
    if drivers != [driver] {
        return Err(format!(
            "parameter {id} drivers {drivers:?}: expected the direct alias {driver}"
        ));
    }
    source_bound(parameters, driver, seen, sources)
}
fn literal_bound(
    parameters: &ParameterCollection<u64>,
    id: u64,
    source: &str,
) -> Result<Enclosure, String> {
    let drivers = parameters.drivers(id).map_err(|e| e.to_string())?;
    if !drivers.is_empty() {
        return Err(format!(
            "parameter {id} drivers {drivers:?}: expected a literal without drivers"
        ));
    }
    let end = crate::units::numeric_prefix_len(source);
    let (decimal, suffix) = source.split_at(end);
    let number = decimal
        .parse::<f64>()
        .ok()
        .filter(|value| value.is_finite());
    if number.is_none() || decimal.len() > 128 {
        return Err(format!(
            "parameter {id} literal {source:?}: expected a finite decimal length"
        ));
    }
    let factor = match suffix.trim() {
        "mm" => "0.1",
        "cm" => "1",
        "m" => "100",
        _ => {
            return Err(format!(
                "parameter {id} literal {source:?}: expected explicit mm, cm, or m"
            ));
        }
    };
    Enclosure::decimal(decimal)?.product(Enclosure::decimal(factor)?)
}
