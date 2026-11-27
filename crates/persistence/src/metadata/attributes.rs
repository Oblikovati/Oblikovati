// SPDX-License-Identifier: GPL-2.0-only

use super::collection::{append_blob, append_text, require_unique};
use super::{MetadataCodecError, blob_length, encode_value, reader::ValueReader};
use oblikovati_model::attribute::{AttributeAnchor, AttributeTarget, DocumentAttributes};
use std::collections::BTreeSet;

/// Encode ordered targets and attributes using the Go layout.
/// Example: `let encoded = encode_attributes(&attributes)?;`.
pub fn encode_attributes(attributes: &DocumentAttributes) -> Result<Vec<u8>, MetadataCodecError> {
    let mut encoded = blob_length(attributes.anchors().len())?
        .to_le_bytes()
        .to_vec();
    for anchor in attributes.anchors() {
        encode_anchor(&mut encoded, anchor)?;
    }
    Ok(encoded)
}

fn encode_anchor(
    encoded: &mut Vec<u8>,
    anchor: &AttributeAnchor,
) -> Result<(), MetadataCodecError> {
    append_blob(encoded, anchor.target.encoded())?;
    encoded.extend_from_slice(&blob_length(anchor.sets().len())?.to_le_bytes());
    for set in anchor.sets() {
        append_text(encoded, &set.name)?;
        encoded.extend_from_slice(&blob_length(set.attributes().len())?.to_le_bytes());
        for attribute in set.attributes() {
            append_text(encoded, &attribute.name)?;
            encoded.extend_from_slice(&encode_value(&attribute.value)?);
        }
    }
    Ok(())
}

/// Restore a complete collection after validating all identities and entries.
/// Example: `let attributes = decode_attributes(&encoded)?;`.
pub fn decode_attributes(encoded: &[u8]) -> Result<DocumentAttributes, MetadataCodecError> {
    let mut reader = ValueReader::new(encoded);
    let mut attributes = DocumentAttributes::default();
    let mut targets = BTreeSet::new();
    for _ in 0..reader.count(24)? {
        let target = AttributeTarget::from_encoded(reader.blob()?).map_err(codec_error)?;
        if !targets.insert(target.clone()) {
            return Err(MetadataCodecError(format!(
                "attribute target {target:?}: expected unique reference identity"
            )));
        }
        attributes.ensure_target(&target);
        restore_anchor(&mut reader, &mut attributes, &target)?;
    }
    if !reader.remaining().is_empty() {
        return Err(MetadataCodecError(format!(
            "attribute collection has {} trailing bytes: expected complete input consumption",
            reader.remaining().len()
        )));
    }
    Ok(attributes)
}

fn restore_anchor(
    reader: &mut ValueReader<'_>,
    attributes: &mut DocumentAttributes,
    target: &AttributeTarget,
) -> Result<(), MetadataCodecError> {
    let mut sets = BTreeSet::new();
    for _ in 0..reader.count(8)? {
        let set = reader.text()?;
        require_unique(&mut sets, &set, "set")?;
        attributes.ensure_set(target, &set).map_err(codec_error)?;
        restore_set(reader, attributes, target, &set)?;
    }
    Ok(())
}

fn restore_set(
    reader: &mut ValueReader<'_>,
    attributes: &mut DocumentAttributes,
    target: &AttributeTarget,
    set: &str,
) -> Result<(), MetadataCodecError> {
    let mut names = BTreeSet::new();
    for _ in 0..reader.count(6)? {
        let name = reader.text()?;
        require_unique(&mut names, &name, "attribute")?;
        attributes
            .put(target, set, &name, reader.value()?)
            .map_err(codec_error)?;
    }
    Ok(())
}

fn codec_error(error: impl std::fmt::Display) -> MetadataCodecError {
    MetadataCodecError(format!("binary attributes: {error}"))
}
