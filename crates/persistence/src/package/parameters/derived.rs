// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, Value, codec_error, parameter_error};
use oblikovati_model::parameter::ParameterCollection;
use serde::Deserialize;

#[derive(Default, Deserialize)]
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
