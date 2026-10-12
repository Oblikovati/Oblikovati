// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::ASSET_BUILTIN;
use serde::de::DeserializeOwned;
use serde_yaml_ng::{Mapping, Value};
pub(super) fn api_catalog_record<T: DeserializeOwned>(
    name: &str,
    record: Value,
    aliases: &[(&str, &str)],
) -> Result<T, String> {
    let mut mapping = require_catalog_mapping(name, record)?;
    for (stored, api) in aliases {
        rename_catalog_field(name, &mut mapping, stored, api)?;
    }
    if let Some(source) = mapping.get(key("source")) {
        return Err(format!(
            "catalog {name:?} source {source:?}: expected scope owned by the catalog, without a source override"
        ));
    }
    mapping.insert(key("source"), key(ASSET_BUILTIN.0.as_ref()));
    let identity = mapping.get(key("id")).cloned().unwrap_or(Value::Null);
    serde_yaml_ng::from_value(Value::Mapping(mapping)).map_err(|cause| {
        format!(
            "catalog {name:?} asset {identity:?}: expected the released API property shape: {cause}"
        )
    })
}
fn require_catalog_mapping(name: &str, record: Value) -> Result<Mapping, String> {
    match record {
        Value::Mapping(mapping) => Ok(mapping),
        offending => Err(format!(
            "catalog {name:?} record {offending:?}: expected an asset mapping"
        )),
    }
}
fn rename_catalog_field(
    name: &str,
    mapping: &mut Mapping,
    stored: &str,
    api: &str,
) -> Result<(), String> {
    let Some(value) = mapping.remove(key(stored)) else {
        return Ok(());
    };
    if mapping.contains_key(key(api)) {
        return Err(format!(
            "catalog {name:?} fields {stored:?} and {api:?}: expected only one stored property name"
        ));
    }
    mapping.insert(key(api), value);
    Ok(())
}
fn key(value: &str) -> Value {
    Value::String(value.into())
}
