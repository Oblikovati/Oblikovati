// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, Value, codec_error};
use oblikovati_model::property::{
    DESIGN_TRACKING, DOCUMENT_SUMMARY, DocumentProperties, SUMMARY_INFORMATION, USER_DEFINED,
};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;
mod exposure;
mod sets;
mod value;

#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct PropertyRecipe {
    set: String,
    name: String,
    #[serde(rename = "type")]
    kind: String,
    value: String,
}

impl DocumentPackage {
    /// Restore typed document properties without changing a live collection.
    /// Example: `let properties = package.restore_properties()?;`.
    pub fn restore_properties(&self) -> Result<DocumentProperties, PackageCodecError> {
        let section = self
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .and_then(|model| model.get("properties"));
        let records: Vec<PropertyRecipe> = match section {
            None | Some(Value::Null) => Vec::new(),
            Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
        };
        let mut properties = DocumentProperties::default();
        let declared = sets::restore_sets(self, &mut properties)?;
        let mut keys = BTreeSet::new();
        for record in records {
            if declared && properties.set(&record.set).is_none() {
                return Err(PackageCodecError(format!(
                    "property set {:?}: expected a declared custom set or a standard set",
                    record.set
                )));
            }
            restore_property(record, &mut properties, &mut keys)?;
        }
        Ok(properties)
    }

    /// Replace only the property recipe after verifying a lossless reconstruction.
    /// Example: `let updated = package.with_properties(&properties)?;`.
    pub fn with_properties(
        &self,
        properties: &DocumentProperties,
    ) -> Result<Self, PackageCodecError> {
        self.restore_properties()?;
        let mut candidate = self.clone();
        sets::capture_sections(&mut candidate, properties, false)?;
        if candidate.restore_properties()? != *properties {
            return Err(PackageCodecError(
                "property export: expected an exact typed collection round trip".into(),
            ));
        }
        Ok(candidate)
    }
}

fn restore_property(
    record: PropertyRecipe,
    properties: &mut DocumentProperties,
    keys: &mut BTreeSet<(String, String)>,
) -> Result<(), PackageCodecError> {
    if !keys.insert((record.set.clone(), record.name.clone())) {
        return Err(PackageCodecError(format!(
            "property {:?}/{:?}: expected one record per set/name pair",
            record.set, record.name
        )));
    }
    let value = value::decode(&record)?;
    properties
        .put(&record.set, &record.name, value)
        .map_err(|error| PackageCodecError(error.to_string()))
}

fn capture_properties(
    properties: &DocumentProperties,
    include_exposure: bool,
) -> Result<Vec<PropertyRecipe>, PackageCodecError> {
    let mut records = Vec::new();
    for set in properties.sets() {
        for property in set.properties() {
            records.push(capture_property(&set.name, property, include_exposure)?);
        }
    }
    Ok(records)
}

fn capture_property(
    set: &str,
    property: &oblikovati_model::property::DocumentProperty,
    include_exposure: bool,
) -> Result<PropertyRecipe, PackageCodecError> {
    if !include_exposure && !property.from_parameter.is_empty() {
        return Err(PackageCodecError(format!(
            "property {set:?}/{:?}: expected authored provenance; parameter exposure requires host reconstruction",
            property.name
        )));
    }
    let (kind, value) = value::encode(&property.value)?;
    Ok(PropertyRecipe {
        set: set.into(),
        name: property.name.clone(),
        kind: kind.into(),
        value,
    })
}
