// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, Value, codec_error};
use oblikovati_model::attachment::{AttachmentContents, DocumentAttachments, FileAttachment};
use serde::{Deserialize, Serialize};
mod record;
mod timestamp;

#[derive(Default, Deserialize, Serialize)]
#[serde(default, deny_unknown_fields, rename_all = "camelCase")]
struct StoredAttachment {
    name: String,
    kind: String,
    full_file_name: String,
    resource_id: String,
    payload: String,
    last_known_file_time: String,
    browser_visible: bool,
}

impl DocumentPackage {
    /// Restore attachments without rereading sources or generating resource identities.
    /// Example: `let attachments = package.restore_attachments()?;`.
    pub fn restore_attachments(&self) -> Result<DocumentAttachments, PackageCodecError> {
        let records: Vec<StoredAttachment> = match self.root.get("attachments") {
            None | Some(Value::Null) => Vec::new(),
            Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
        };
        let mut attachments = DocumentAttachments::default();
        for record in records {
            let contents = record.contents()?;
            attachments
                .restore(
                    &record.name,
                    &record.full_file_name,
                    contents,
                    record.browser_visible,
                )
                .map_err(|cause| PackageCodecError(cause.to_string()))?;
        }
        Ok(attachments)
    }

    /// Replace validated attachment state, preserving all other package sections.
    /// Example: `let saved = package.with_attachments(&attachments)?;`.
    pub fn with_attachments(
        &self,
        attachments: &DocumentAttachments,
    ) -> Result<Self, PackageCodecError> {
        self.restore_attachments()?;
        let records = attachments
            .entries()
            .iter()
            .map(StoredAttachment::capture)
            .collect::<Result<Vec<_>, _>>()?;
        let mut candidate = self.clone();
        candidate.root.remove("attachments");
        if !records.is_empty() {
            candidate.root.insert(
                "attachments".into(),
                serde_yaml_ng::to_value(records).map_err(codec_error)?,
            );
        }
        Ok(candidate)
    }
}
