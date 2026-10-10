// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{Error, wire::*};
use oblikovati_model::{
    parameter::{ParameterCollection, ParameterHealth, ParameterState, ParameterValue, kind_name},
    quantity::{Dimension, Quantity},
};

type Parameters = ParameterCollection<u64>;

pub(super) fn list(parameters: &Parameters) -> Result<ListParametersResult, Error> {
    let rows = parameters
        .ids()
        .iter()
        .map(|id| information(parameters, *id))
        .collect::<Result<_, _>>()?;
    Ok(ListParametersResult {
        parameters: Some(rows),
    })
}

pub(super) fn get(
    parameters: &Parameters,
    request: ParameterNameArgs,
) -> Result<ParameterInfo, Error> {
    information(parameters, resolve(parameters, &request.name)?)
}

pub(super) fn neighbors(
    parameters: &Parameters,
    request: ParameterNameArgs,
    drivers: bool,
) -> Result<ParameterNamesResult, Error> {
    let id = resolve(parameters, &request.name)?;
    let links = if drivers {
        parameters.drivers(id)
    } else {
        parameters.readers(id)
    };
    let names = names(
        parameters,
        &links.map_err(|error| Error(error.to_string()))?,
    )?;
    Ok(ParameterNamesResult { names: Some(names) })
}

fn resolve(parameters: &Parameters, name: &str) -> Result<u64, Error> {
    parameters.find(name).ok_or_else(|| {
        Error(format!(
            "parameter name {name:?}: expected a parameter in the active document"
        ))
    })
}

fn state(parameters: &Parameters, id: u64) -> Result<&ParameterState<u64>, Error> {
    parameters.get(id).ok_or_else(|| {
        Error(format!(
            "parameter ID {id}: expected a registered parameter"
        ))
    })
}

fn information(parameters: &Parameters, id: u64) -> Result<ParameterInfo, Error> {
    let record = state(parameters, id)?;
    let kind = kind_name(record.kind).ok_or_else(|| {
        Error(format!(
            "parameter kind {:?}: expected a supported API kind",
            record.kind
        ))
    })?;
    let health = match &record.health {
        ParameterHealth::Healthy => String::new(),
        ParameterHealth::Failed(failure) => failure.to_string(),
    };
    Ok(ParameterInfo {
        name: record.name.clone(),
        kind: kind.into(),
        expression: record.source.clone(),
        value: formatted(parameters, record)?,
        health,
    })
}

fn formatted(parameters: &Parameters, record: &ParameterState<u64>) -> Result<String, Error> {
    // Inspection preserves the last-good nominal value, even when recomputation failed.
    let quantity = match &record.last_good_value {
        Some(ParameterValue::Text(value)) => return Ok(value.clone()),
        Some(ParameterValue::Boolean(value)) => return Ok(value.to_string()),
        Some(ParameterValue::Numeric(quantity)) => *quantity,
        None => Quantity::new(0.0, Dimension::Unitless), // Go's zero value before first successful evaluation.
    };
    parameters
        .units()
        .format_precise(quantity)
        .map_err(|error| Error(error.to_string()))
}

pub(super) fn names(parameters: &Parameters, ids: &[u64]) -> Result<Vec<String>, Error> {
    ids.iter()
        .map(|id| state(parameters, *id).map(|record| record.name.clone()))
        .collect()
}
