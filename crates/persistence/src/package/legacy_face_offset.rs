// SPDX-License-Identifier: GPL-2.0-only
//! Inspect supported Go face-offset records without adopting or rewriting them.
use super::{DocumentPackage, PackageCodecError, binary_text::decode_raw_url_base64};
use serde::Deserialize;
use std::collections::BTreeSet;

/// A source work plane awaiting an explicit face-key migration and adoption.
#[derive(Debug, PartialEq)]
pub struct LegacyFaceOffsetCandidate {
    pub creation_sequence: u64,
    pub face_key: Vec<u8>,
    pub offset: f64,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct StoredFaceOffset {
    collection: String,
    kind: String,
    seq: u64,
    refs: Vec<String>,
    #[serde(default)]
    offset: f64,
}

impl DocumentPackage {
    /// Prepare face-offset records without changing the package.
    /// Example: `let planes = package.prepare_legacy_face_offsets()?;`.
    pub fn prepare_legacy_face_offsets(
        &self,
    ) -> Result<Vec<LegacyFaceOffsetCandidate>, PackageCodecError> {
        let Some(value) = self
            .root
            .get("model")
            .and_then(|model| model.get("workFeatures"))
        else {
            return Ok(Vec::new());
        };
        if value.is_null() {
            return Ok(Vec::new());
        }
        let rows: Vec<StoredFaceOffset> =
            serde_yaml_ng::from_value(value.clone()).map_err(failure)?;
        prepare_offsets(rows)
    }
}

fn prepare_offsets(
    rows: Vec<StoredFaceOffset>,
) -> Result<Vec<LegacyFaceOffsetCandidate>, PackageCodecError> {
    let mut sequences = BTreeSet::new();
    rows.into_iter()
        .map(|row| {
            if row.seq == 0 || !sequences.insert(row.seq) {
                return Err(failure(format!(
                    "sequence {}: expected a unique nonzero sequence",
                    row.seq
                )));
            }
            prepare_offset(row)
        })
        .collect()
}

fn prepare_offset(row: StoredFaceOffset) -> Result<LegacyFaceOffsetCandidate, PackageCodecError> {
    if row.collection != "plane" || row.kind != "plane-offset" || !row.offset.is_finite() {
        return Err(failure(format!(
            "collection {:?}, kind {:?}, offset {:?}: expected a finite plane-offset",
            row.collection, row.kind, row.offset
        )));
    }
    let [reference] = row.refs.as_slice() else {
        return Err(failure(format!(
            "refs {:?}: expected exactly one face reference",
            row.refs
        )));
    };
    let face_key = decode_face_reference(reference)?;
    Ok(LegacyFaceOffsetCandidate {
        creation_sequence: row.seq,
        face_key,
        offset: row.offset,
    })
}

fn decode_face_reference(reference: &str) -> Result<Vec<u8>, PackageCodecError> {
    let encoded = reference
        .strip_prefix("face/")
        .ok_or_else(|| failure(format!("reference {reference:?}: expected face/<base64>")))?;
    let key = decode_raw_url_base64("legacy face reference", encoded)?;
    if key.is_empty() {
        return Err(failure(format!(
            "reference {reference:?}: expected a nonempty face key"
        )));
    }
    Ok(key)
}

fn failure(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!(
        "legacy model.workFeatures: expected supported face-offset records: {error}"
    ))
}
