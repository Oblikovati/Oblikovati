// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, Value, codec_error, parameter_error};
use oblikovati_model::parameter::ParameterCollection;
use serde::{Deserialize, Serialize};

#[derive(Default, Deserialize, Serialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
struct DerivedTableRecipe {
    id: i64,
    source_document: String,
    linked: Vec<String>,
    owned_by_feature: bool,
}

pub(super) fn restore_tables(
    model: Option<&Value>,
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
        parameters
            .restore_derived_table(
                record.id,
                &record.source_document,
                &record.linked,
                record.owned_by_feature,
            )
            .map_err(parameter_error)?;
    }
    Ok(())
}

pub(super) fn capture_tables(
    parameters: &ParameterCollection<u64>,
) -> Result<Value, PackageCodecError> {
    let mut records = Vec::new();
    for table in parameters.derived_tables() {
        let references = parameters
            .derived_references(table.id)
            .map_err(parameter_error)?;
        if references
            .iter()
            .any(|reference| reference.parameter != reference.source_parameter)
        {
            return Err(PackageCodecError(format!(
                "derived table {}: expected matching local/source names in the legacy recipe",
                table.id
            )));
        }
        records.push(DerivedTableRecipe {
            id: table.id,
            source_document: table.source_document.clone(),
            linked: table.linked.clone(),
            owned_by_feature: table.owned_by_feature,
        });
    }
    serde_yaml_ng::to_value(records).map_err(codec_error)
}
