// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{Error, types::VariantValue, wire::*};
use oblikovati_model::{
    parameter::ParameterCollection,
    property::{DocumentProperties, USER_DEFINED},
};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

pub(super) fn list<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: ListPropertiesArgs,
) -> Result<ListPropertiesResult, Error> {
    let properties = &workspace
        .session(request.document)
        .map_err(|error| Error(error.to_string()))?
        .snapshot()
        .properties;
    let entries: Vec<PropertyInfo> = properties
        .sets()
        .iter()
        .flat_map(|set| {
            set.properties().iter().map(|property| {
                properties
                    .info(&set.name, &property.name)
                    .expect("owned property")
            })
        })
        .collect();
    Ok(ListPropertiesResult {
        properties: (!entries.is_empty()).then_some(entries),
    })
}

pub(super) fn get<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: GetPropertyArgs,
) -> Result<PropertyResult, Error> {
    let snapshot = workspace
        .session(request.document)
        .map_err(|error| Error(error.to_string()))?
        .snapshot();
    if let Some(parameter) = exposed_parameter(&snapshot.parameters, &request.set, &request.name)
        && parameter.current().is_none()
    {
        return Err(Error(format!(
            "property {:?}/{:?}: exposed parameter is unhealthy ({:?}); expected a current value",
            request.set, request.name, parameter.health
        )));
    }
    read(&snapshot.properties, &request.set, &request.name)
}

fn read(properties: &DocumentProperties, set: &str, name: &str) -> Result<PropertyResult, Error> {
    properties
        .info(set, name)
        .map(|property| PropertyResult { property })
        .ok_or_else(|| {
            Error(format!(
                "property {set:?}/{name:?}: expected an existing property in the document"
            ))
        })
}

pub(super) fn set(
    properties: &mut DocumentProperties,
    parameters: &ParameterCollection<u64>,
    request: SetPropertyArgs,
) -> Result<PropertyResult, Error> {
    if let Some(property) = properties.get(&request.set, &request.name)
        && !property.from_parameter.is_empty()
    {
        return Err(Error(format!(
            "property {:?}/{:?} is owned by parameter {:?}: expected an edit to that parameter",
            request.set, request.name, property.from_parameter
        )));
    }
    if exposed_parameter(parameters, &request.set, &request.name).is_some() {
        return Err(Error(format!(
            "property {:?}/{:?} is owned by an exposed parameter: expected an edit to that parameter",
            request.set, request.name
        )));
    }
    // Go's default Variant case becomes an empty string; property units are not stored.
    let value = match request.value.value {
        VariantValue::Uninitialized => VariantValue::String(String::new()),
        value => value,
    };
    properties
        .put(&request.set, &request.name, value)
        .map_err(|error| Error(error.to_string()))?;
    read(properties, &request.set, &request.name)
}

fn exposed_parameter<'a>(
    parameters: &'a ParameterCollection<u64>,
    set: &str,
    name: &str,
) -> Option<&'a oblikovati_model::parameter::ParameterState<u64>> {
    if set != USER_DEFINED {
        return None;
    }
    let parameter = parameters.get(parameters.find(name)?)?;
    parameter
        .annotations
        .exposed_as_property
        .then_some(parameter)
}
