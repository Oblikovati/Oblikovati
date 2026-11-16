// SPDX-License-Identifier: GPL-2.0-only
//! Conditional bounds from a restricted set of authored length sources.
use crate::expression::{ConditionalValue, conditional_expression};
use crate::{parameter::ParameterCollection, quantity::Dimension};
use oblikovati_api::types::{MODEL_PARAM, MODEL_VALUE_NOMINAL, USER_PARAM};
use std::collections::{BTreeMap, BTreeSet};

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
    /// Read distinct sources in traversal order; example: `bound.sources()`.
    pub fn sources(&self) -> &[(u64, String)] {
        &self.sources
    }
}
/// Enclose supported authored arithmetic with nominal model values.
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
    let mut evaluator = SourceBounds::new(parameters);
    let value = evaluator.evaluate(id)?;
    if value.dimension != Dimension::Length {
        return Err(format!("parameter {id}: expected a length result"));
    }
    Ok(ConditionalParameterBound {
        id,
        bounds: value.bounds.bounds(),
        sources: evaluator.sources,
    })
}
struct SourceBounds<'a> {
    parameters: &'a ParameterCollection<u64>,
    active: BTreeSet<u64>,
    sources: Vec<(u64, String)>,
    cache: BTreeMap<u64, ConditionalValue>,
}
impl<'a> SourceBounds<'a> {
    fn new(parameters: &'a ParameterCollection<u64>) -> Self {
        Self {
            parameters,
            active: BTreeSet::new(),
            sources: Vec::new(),
            cache: BTreeMap::new(),
        }
    }
    fn evaluate(&mut self, id: u64) -> Result<ConditionalValue, String> {
        if let Some(value) = self.cache.get(&id) {
            return Ok(*value);
        }
        if self.active.len() >= 128 {
            return Err(format!(
                "parameter {id}: expected at most 128 nested sources"
            ));
        }
        if !self.active.insert(id) {
            return Err(format!("parameter {id}: expected an acyclic source chain"));
        }
        let bound = self.evaluate_uncached(id)?;
        self.active.remove(&id);
        self.cache.insert(id, bound);
        Ok(bound)
    }
    fn evaluate_uncached(&mut self, id: u64) -> Result<ConditionalValue, String> {
        let state = self
            .parameters
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
            .filter(|value| value.value.is_finite())
            .ok_or_else(|| format!("parameter {id}: expected a healthy finite value"))?;
        let source = state.source.clone();
        self.sources.push((id, source.clone()));
        let mut referenced = BTreeSet::new();
        let result = conditional_expression(&source, self.parameters.units(), |name| {
            self.reference(id, name, &mut referenced)
        })?;
        self.require_drivers(id, &referenced, result.dimension, value.dimension)?;
        Ok(ConditionalValue {
            bounds: result.bounds.include(value.value)?,
            dimension: result.dimension,
        })
    }
    fn reference(
        &mut self,
        owner: u64,
        name: &str,
        referenced: &mut BTreeSet<u64>,
    ) -> Result<ConditionalValue, String> {
        let driver = self
            .parameters
            .find(name)
            .ok_or_else(|| format!("parameter reference {name:?}: expected an existing source"))?;
        if self.parameters.bound_reference_id(owner, name) != Some(driver) {
            return Err(format!(
                "parameter {owner} reference {name:?}: expected the same bound identity {driver}"
            ));
        }
        referenced.insert(driver);
        self.evaluate(driver)
    }
    fn require_drivers(
        &self,
        id: u64,
        referenced: &BTreeSet<u64>,
        actual: Dimension,
        expected: Dimension,
    ) -> Result<(), String> {
        let drivers: BTreeSet<u64> = self
            .parameters
            .drivers(id)
            .map_err(|error| error.to_string())?
            .into_iter()
            .collect();
        if *referenced != drivers || actual != expected {
            return Err(format!(
                "parameter {id} references {referenced:?} and dimension {actual:?}: expected drivers {drivers:?} and dimension {expected:?}"
            ));
        }
        Ok(())
    }
}
