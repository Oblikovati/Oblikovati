// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentGuid, FileIdentity, RecipeDigest};
use serde::{Deserialize, Serialize};
use std::{error::Error, fmt};

/// The existing file identity mapping, independent of the package encoding.
#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
pub struct FileIdentityRecord {
    #[serde(skip_serializing_if = "String::is_empty")]
    pub internal_name: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub revision_id: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub database_revision_id: String,
    #[serde(skip_serializing_if = "counter_is_zero")]
    pub save_counter: u64,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub version_created: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub version_saved: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub model_digest: String,
}

/// A persisted identity field whose value cannot be restored safely.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct IdentityRecordError {
    field: &'static str,
    value: String,
    expected: &'static str,
}

impl FileIdentityRecord {
    /// Restore a validated identity, or retain the fresh identity for old files.
    ///
    /// ```
    /// use oblikovati_model::identity::{FileIdentity, FileIdentityRecord, SystemGuidSource};
    /// let fresh = FileIdentity::new("0.1.0", &mut SystemGuidSource)?;
    /// assert_eq!(FileIdentityRecord::default().restore(&fresh)?, fresh);
    /// # Ok::<(), Box<dyn std::error::Error>>(())
    /// ```
    pub fn restore(&self, fresh: &FileIdentity) -> Result<FileIdentity, IdentityRecordError> {
        // Go ignores the whole identity block when internalName is absent.
        if self.internal_name.is_empty() {
            return Ok(fresh.clone());
        }
        Ok(FileIdentity {
            internal_name: parse_guid("internalName", &self.internal_name)?,
            revision_id: optional_guid("revisionId", &self.revision_id)?,
            database_revision_id: optional_guid("databaseRevisionId", &self.database_revision_id)?,
            save_counter: self.save_counter,
            version_created: self.version_created.clone(),
            version_saved: self.version_saved.clone(),
            model_digest: parse_digest(&self.model_digest)?,
        })
    }
}

impl From<&FileIdentity> for FileIdentityRecord {
    fn from(identity: &FileIdentity) -> Self {
        Self {
            internal_name: identity.internal_name.to_string(),
            revision_id: optional_text(identity.revision_id),
            database_revision_id: optional_text(identity.database_revision_id),
            save_counter: identity.save_counter,
            version_created: identity.version_created.clone(),
            version_saved: identity.version_saved.clone(),
            model_digest: optional_text(identity.model_digest),
        }
    }
}

fn optional_text<Value: fmt::Display>(value: Option<Value>) -> String {
    value.map(|present| present.to_string()).unwrap_or_default()
}

fn counter_is_zero(counter: &u64) -> bool {
    *counter == 0
}

fn parse_guid(field: &'static str, value: &str) -> Result<DocumentGuid, IdentityRecordError> {
    value.parse().map_err(|_| IdentityRecordError {
        field,
        value: value.to_owned(),
        expected: "32 hexadecimal digits or an 8-4-4-4-12 GUID",
    })
}

fn optional_guid(
    field: &'static str,
    value: &str,
) -> Result<Option<DocumentGuid>, IdentityRecordError> {
    if value.is_empty() {
        return Ok(None);
    }
    parse_guid(field, value).map(Some)
}

fn parse_digest(value: &str) -> Result<Option<RecipeDigest>, IdentityRecordError> {
    if value.is_empty() {
        return Ok(None);
    }
    if value.len() != 64 || !value.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        return Err(IdentityRecordError {
            field: "modelDigest",
            value: value.to_owned(),
            expected: "64 SHA-256 hexadecimal digits",
        });
    }
    let mut bytes = [0; 32];
    for (index, byte) in bytes.iter_mut().enumerate() {
        // ASCII shape validation above makes each two-byte slice valid hex.
        *byte =
            u8::from_str_radix(&value[index * 2..index * 2 + 2], 16).expect("validated hex pair");
    }
    Ok(Some(RecipeDigest(bytes)))
}

impl fmt::Display for IdentityRecordError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            formatter,
            "identity field {} {:?}: expected {}",
            self.field, self.value, self.expected
        )
    }
}

impl Error for IdentityRecordError {}
