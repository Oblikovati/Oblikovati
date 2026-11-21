// SPDX-License-Identifier: GPL-2.0-only
use super::{MetadataCodecError, blob_length};
use std::collections::BTreeSet;

pub(super) fn require_unique(
    names: &mut BTreeSet<String>,
    name: &str,
    kind: &str,
) -> Result<(), MetadataCodecError> {
    if name.is_empty() || !names.insert(name.into()) {
        return Err(MetadataCodecError(format!(
            "metadata {kind} name {name:?}: expected a unique nonempty name"
        )));
    }
    Ok(())
}

pub(super) fn append_text(encoded: &mut Vec<u8>, text: &str) -> Result<(), MetadataCodecError> {
    append_blob(encoded, text.as_bytes())
}

pub(super) fn append_blob(encoded: &mut Vec<u8>, bytes: &[u8]) -> Result<(), MetadataCodecError> {
    encoded.extend_from_slice(&blob_length(bytes.len())?.to_le_bytes());
    encoded.extend_from_slice(bytes);
    Ok(())
}
