// SPDX-License-Identifier: GPL-2.0-only

use super::collection::{append_text, require_unique};
use super::{MetadataCodecError, blob_length, encode_value, reader::ValueReader};
use oblikovati_model::property::{DocumentProperties, DocumentProperty};
use std::collections::BTreeSet;

/// Encode ordered property sets and provenance using the Go binary layout.
/// Example: `let bytes = encode_properties(&properties)?;`.
pub fn encode_properties(properties: &DocumentProperties) -> Result<Vec<u8>, MetadataCodecError> {
    let mut encoded = blob_length(properties.sets().len())?.to_le_bytes().to_vec();
    for set in properties.sets() {
        append_text(&mut encoded, &set.name)?;
        encoded.extend_from_slice(&blob_length(set.properties().len())?.to_le_bytes());
        for property in set.properties() {
            append_text(&mut encoded, &property.name)?;
            append_text(&mut encoded, &property.from_parameter)?;
            encoded.extend_from_slice(&encode_value(&property.value)?);
        }
    }
    Ok(encoded)
}

/// Restore a complete property collection, refusing duplicates and trailing bytes.
/// Example: `let properties = decode_properties(&encoded)?;`.
pub fn decode_properties(encoded: &[u8]) -> Result<DocumentProperties, MetadataCodecError> {
    let mut reader = ValueReader::new(encoded);
    let mut properties = DocumentProperties::default();
    let mut names = Vec::new();
    let mut unique = BTreeSet::new();
    for _ in 0..reader.count(8)? {
        let name = reader.text()?;
        require_unique(&mut unique, &name, "set")?;
        properties.ensure_set(&name).map_err(model_error)?;
        restore_set(&mut reader, &mut properties, &name)?;
        names.push(name);
    }
    validate_collection(&reader, &properties, &names)?;
    Ok(properties)
}

fn restore_set(
    reader: &mut ValueReader<'_>,
    properties: &mut DocumentProperties,
    set: &str,
) -> Result<(), MetadataCodecError> {
    let mut unique = BTreeSet::new();
    for _ in 0..reader.count(10)? {
        let name = reader.text()?;
        require_unique(&mut unique, &name, "property")?;
        let from_parameter = reader.text()?;
        let value = reader.value()?;
        properties
            .restore_property(
                set,
                DocumentProperty {
                    name,
                    value,
                    from_parameter,
                },
            )
            .map_err(model_error)?;
    }
    Ok(())
}

fn validate_collection(
    reader: &ValueReader<'_>,
    properties: &DocumentProperties,
    names: &[String],
) -> Result<(), MetadataCodecError> {
    let canonical: Vec<_> = properties.sets().iter().map(|set| &set.name).collect();
    if canonical != names.iter().collect::<Vec<_>>() {
        return Err(MetadataCodecError(format!(
            "property set order {names:?}: expected all standard sets before custom sets, {canonical:?}"
        )));
    }
    if !reader.remaining().is_empty() {
        return Err(MetadataCodecError(format!(
            "property collection has {} trailing bytes: expected complete input consumption",
            reader.remaining().len()
        )));
    }
    Ok(())
}

fn model_error(error: impl std::fmt::Display) -> MetadataCodecError {
    MetadataCodecError(format!("binary property collection: {error}"))
}
