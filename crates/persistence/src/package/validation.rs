// SPDX-License-Identifier: GPL-2.0-only

use super::{Mapping, PackageCodecError, Value};

const SCALAR_FIELDS: &[&str] = &["schemaVersion", "documentType", "subType", "displayName"];
const MAPPING_FIELDS: &[&str] = &[
    "identity",
    "displaySettings",
    "sketchSettings",
    "bodyNames",
    "bodyColorStyles",
    "resources",
    "model",
    "data",
];
const SEQUENCE_FIELDS: &[&str] = &[
    "references",
    "attachments",
    "interests",
    "pointClouds",
    "attributes",
];

pub(super) fn validate_root(root: &Mapping) -> Result<(), PackageCodecError> {
    for (key, value) in root {
        let Some(name) = key.as_str() else {
            return Err(PackageCodecError(format!(
                "root key {key:?}: expected a named schema-2 field"
            )));
        };
        validate_field(name, value)?;
    }
    Ok(())
}

fn validate_field(name: &str, value: &Value) -> Result<(), PackageCodecError> {
    if SCALAR_FIELDS.contains(&name) {
        return Ok(()); // Typed manifest decoding validates these scalar fields.
    }
    if MAPPING_FIELDS.contains(&name) && (value.is_mapping() || value.is_null()) {
        return Ok(());
    }
    if SEQUENCE_FIELDS.contains(&name) && (value.is_sequence() || value.is_null()) {
        return Ok(());
    }
    Err(PackageCodecError(format!(
        "section {name:?} with {value:?}: expected a supported schema-2 section and its mapping/sequence shape"
    )))
}
