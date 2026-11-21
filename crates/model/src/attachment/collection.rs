// SPDX-License-Identifier: GPL-2.0-only
use super::*;

/// Ordered document attachments with unique names and owned payloads.
#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct DocumentAttachments {
    entries: Vec<FileAttachment>,
}

impl DocumentAttachments {
    /// Borrow insertion order. Example: `let count = attachments.entries().len();`.
    pub fn entries(&self) -> &[FileAttachment] {
        &self.entries
    }
    /// Find an exact attachment name. Example: `let loads = attachments.get("loads");`.
    pub fn get(&self, name: &str) -> Option<&FileAttachment> {
        self.entries.iter().find(|entry| entry.name == name)
    }
    /// Capture a source and publish only after all validation, I/O, and allocation succeed.
    /// Example: `attachments.attach("loads", ATTACHMENT_LINKED, "loads.csv", Some(&mut files), &mut guids)?;`.
    pub fn attach(
        &mut self,
        name: &str,
        kind: AttachmentKind,
        path: &str,
        files: Option<&mut dyn ExternalAttachmentFiles>,
        source: &mut dyn GuidSource,
    ) -> Result<(), AttachmentError> {
        self.validate_name(name, path)?;
        let contents = capture::capture(kind, path, files, source)?;
        self.restore(name, path, contents, true)
    }
    /// Restore owned captured state without probing files or allocating an identity.
    /// Example: `attachments.restore("loads", "loads.csv", contents, false)?;`.
    pub fn restore(
        &mut self,
        name: &str,
        path: &str,
        contents: AttachmentContents,
        visible: bool,
    ) -> Result<(), AttachmentError> {
        self.validate_name(name, path)?;
        self.validate_resource_id(&contents)?;
        self.entries.push(FileAttachment {
            name: name.into(),
            full_file_name: path.into(),
            contents,
            browser_visible: visible,
        });
        Ok(())
    }
    /// Remove one attachment and its payload. Example: `let removed = attachments.remove("loads");`.
    pub fn remove(&mut self, name: &str) -> bool {
        let Some(index) = self.entries.iter().position(|entry| entry.name == name) else {
            return false;
        };
        self.entries.remove(index);
        true
    }
    /// Change visibility without altering identity or source state.
    /// Example: `attachments.set_browser_visible("loads", false)?;`.
    pub fn set_browser_visible(
        &mut self,
        name: &str,
        visible: bool,
    ) -> Result<(), AttachmentError> {
        let entry = self
            .entries
            .iter_mut()
            .find(|entry| entry.name == name)
            .ok_or_else(|| {
                AttachmentError::Invalid(format!("attachment {name:?}: expected an existing name"))
            })?;
        entry.browser_visible = visible;
        Ok(())
    }
    fn validate_name(&self, name: &str, path: &str) -> Result<(), AttachmentError> {
        if name.is_empty() || path.is_empty() || self.get(name).is_some() {
            return Err(AttachmentError::Invalid(format!(
                "attachment {name:?} source {path:?}: expected a unique nonempty name and nonempty source"
            )));
        }
        Ok(())
    }
    fn validate_resource_id(&self, contents: &AttachmentContents) -> Result<(), AttachmentError> {
        let AttachmentContents::Embedded { resource_id, .. } = contents else {
            return Ok(());
        };
        if self.entries.iter().any(|entry| matches!(&entry.contents, AttachmentContents::Embedded { resource_id: existing, .. } if existing == resource_id)) {
            return Err(AttachmentError::Identity(IdentityTransitionError::RepeatedGuid(*resource_id)));
        }
        Ok(())
    }
}
