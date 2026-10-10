// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::package::binary_text::decode_base64;
use base64::{Engine, engine::general_purpose::STANDARD};

impl StoredAttachment {
    pub(super) fn contents(&self) -> Result<AttachmentContents, PackageCodecError> {
        match self.kind.as_str() {
            "generic"
                if self.resource_id.is_empty()
                    && self.payload.is_empty()
                    && self.last_known_file_time.is_empty() =>
            {
                Ok(AttachmentContents::Generic)
            }
            "linked" if self.resource_id.is_empty() && self.payload.is_empty() => {
                Ok(AttachmentContents::Linked {
                    last_known: timestamp::decode(&self.last_known_file_time)?,
                })
            }
            "embedded" if self.last_known_file_time.is_empty() => {
                Ok(AttachmentContents::Embedded {
                    resource_id: self.resource_id.parse().map_err(|cause| {
                        PackageCodecError(format!(
                            "attachment {:?} resourceId {:?}: expected a GUID: {cause}",
                            self.name, self.resource_id
                        ))
                    })?,
                    bytes: decode_base64(&format!("attachment {:?}", self.name), &self.payload)?,
                })
            }
            _ => Err(PackageCodecError(format!(
                "attachment {:?} kind {:?}: expected generic, linked with only a timestamp, or embedded with only a GUID and payload",
                self.name, self.kind
            ))),
        }
    }

    pub(super) fn capture(attachment: &FileAttachment) -> Result<Self, PackageCodecError> {
        let mut record = Self {
            name: attachment.name().into(),
            full_file_name: attachment.full_file_name().into(),
            browser_visible: attachment.browser_visible(),
            ..Default::default()
        };
        record.capture_contents(attachment.contents())?;
        Ok(record)
    }

    fn capture_contents(&mut self, contents: &AttachmentContents) -> Result<(), PackageCodecError> {
        match contents {
            AttachmentContents::Generic => self.kind = "generic".into(),
            AttachmentContents::Linked { last_known } => {
                self.kind = "linked".into();
                self.last_known_file_time = timestamp::encode(*last_known)?;
            }
            AttachmentContents::Embedded { resource_id, bytes } => {
                self.kind = "embedded".into();
                self.resource_id = resource_id.to_string();
                self.payload = STANDARD.encode(bytes);
            }
        }
        Ok(())
    }
}
