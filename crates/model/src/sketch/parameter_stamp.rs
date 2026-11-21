// SPDX-License-Identifier: GPL-2.0-only
//! Snapshot the relevant parameter graph at solve time.
use crate::{
    parameter::{DimensionExpressionContext, ParameterCollection},
    quantity::Dimension,
    units::UnitsOfMeasure,
};
use oblikovati_api::types::{ModelValueType, ParameterKind};
use std::collections::BTreeSet;

#[derive(Clone, Debug, PartialEq)]
struct ParameterInputStamp {
    id: u64,
    name: String,
    source: String,
    kind: ParameterKind,
    model_value_type: ModelValueType,
    context: Option<DimensionExpressionContext>,
    nominal: Option<(Dimension, u64)>,
    drivers: Vec<u64>,
}
#[derive(Clone, Debug, PartialEq)]
pub(super) struct ParameterStamp {
    units: UnitsOfMeasure,
    inputs: Vec<ParameterInputStamp>,
}
impl ParameterStamp {
    pub fn capture(parameters: &ParameterCollection<u64>, roots: &[u64]) -> Option<Self> {
        let mut seen = BTreeSet::new();
        let mut pending = roots.to_vec();
        let mut inputs = Vec::new();
        while let Some(id) = pending.pop() {
            if !seen.insert(id) {
                continue;
            }
            let state = parameters.get(id)?;
            let drivers = parameters.drivers(id).ok()?;
            pending.extend(drivers.iter().copied());
            inputs.push(ParameterInputStamp {
                id,
                name: state.name.clone(),
                source: state.source.clone(),
                kind: state.kind,
                model_value_type: state.model_value_type,
                context: state.expression_context.clone(),
                nominal: state
                    .current_value()
                    .map(|value| (value.dimension, value.value.to_bits())),
                drivers,
            });
        }
        inputs.sort_by_key(|input| input.id);
        Some(Self {
            units: parameters.units().clone(),
            inputs,
        })
    }
    pub fn matches(&self, parameters: &ParameterCollection<u64>, roots: &[u64]) -> bool {
        Self::capture(parameters, roots).as_ref() == Some(self)
    }
}
