// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, Value, codec_error, parameter_error};
use oblikovati_model::parameter::{DerivedTableHealth, ParameterCollection};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

#[derive(Default, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
struct DerivedTableRecipe {
    id: i64,
    source_document: String,
    linked: Vec<String>,
    owned_by_feature: bool,
    #[serde(skip_serializing_if = "BTreeMap::is_empty")]
    local_names: BTreeMap<String, String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    failure: Option<LinkFailure>,
}

pub(super) fn restore_tables(
    model: Option<&Value>,
    schema: u32,
    parameters: &mut ParameterCollection<u64>,
) -> Result<(), PackageCodecError> {
    let section = model
        .and_then(Value::as_mapping)
        .and_then(|model| model.get("derivedParameterTables"));
    let records: Vec<DerivedTableRecipe> = match section {
        None | Some(Value::Null) => return Ok(()),
        Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
    };
    for record in records {
        if schema < 3 && (!record.local_names.is_empty() || record.failure.is_some()) {
            return Err(PackageCodecError(format!(
                "derived table {} localNames in schema {schema}: expected schema 3",
                record.id
            )));
        }
        parameters
            .restore_derived_table_names(
                record.id,
                &record.source_document,
                &record.linked,
                record.owned_by_feature,
                &record.local_names,
            )
            .map_err(parameter_error)?;
        if let Some(failure) = record.failure {
            let missing = match &failure {
                LinkFailure::Unavailable => None,
                LinkFailure::Missing { parameters } => Some(parameters.as_slice()),
            };
            parameters
                .restore_derived_failure(record.id, missing)
                .map_err(parameter_error)?;
        }
    }
    Ok(())
}

pub(super) fn capture_tables(
    parameters: &ParameterCollection<u64>,
) -> Result<(Value, bool), PackageCodecError> {
    let mut records = Vec::new();
    for table in parameters.derived_tables() {
        let references = parameters
            .derived_references(table.id)
            .map_err(parameter_error)?;
        let local_names = references
            .into_iter()
            .filter(|reference| reference.parameter != reference.source_parameter)
            .map(|reference| (reference.source_parameter, reference.parameter))
            .collect();
        records.push(DerivedTableRecipe {
            id: table.id,
            source_document: table.source_document.clone(),
            linked: table.linked.clone(),
            owned_by_feature: table.owned_by_feature,
            local_names,
            failure: match &table.health {
                DerivedTableHealth::Unavailable => Some(LinkFailure::Unavailable),
                DerivedTableHealth::Missing(parameters) => Some(LinkFailure::Missing {
                    parameters: parameters.clone(),
                }),
                _ => None,
            },
        });
    }
    let extended = records
        .iter()
        .any(|record| !record.local_names.is_empty() || record.failure.is_some());
    Ok((
        serde_yaml_ng::to_value(records).map_err(codec_error)?,
        extended,
    ))
}

#[derive(Deserialize, Serialize)]
#[serde(tag = "state", rename_all = "camelCase", deny_unknown_fields)]
enum LinkFailure {
    Unavailable,
    Missing { parameters: Vec<String> },
}
