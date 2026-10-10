// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{Error, types::VariantValue, wire::*};
use oblikovati_model::property::DocumentProperties;
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
    let properties = &workspace
        .session(request.document)
        .map_err(|error| Error(error.to_string()))?
        .snapshot()
        .properties;
    read(properties, &request.set, &request.name)
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
