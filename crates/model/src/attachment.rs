// SPDX-License-Identifier: GPL-2.0-only

use crate::identity::{DocumentGuid, GuidSource, IdentityTransitionError};
use oblikovati_api::types::*;
use std::{error::Error, fmt, io, time::SystemTime};

mod capture;
mod collection;
pub use collection::DocumentAttachments;

/// Injected foreign-file access; absence is distinct from an access or decoding error.
pub trait ExternalAttachmentFiles {
    /// Return a regular file's modification time, or None when it is absent.
    /// Example: `let modified = files.modified("loads.csv")?;`.
    fn modified(&mut self, path: &str) -> io::Result<Option<SystemTime>>;
    /// Read an owned payload for embedding. Example: `let bytes = files.read("table.csv")?;`.
    fn read(&mut self, path: &str) -> io::Result<Vec<u8>>;
}

/// The attachment kind determines which captured state is meaningful.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum AttachmentContents {
    Generic,
    Linked {
        last_known: Option<SystemTime>,
    },
    Embedded {
        resource_id: DocumentGuid,
        bytes: Vec<u8>,
    },
}

/// An owned foreign-file attachment with an immutable name and source path.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FileAttachment {
    name: String,
    full_file_name: String,
    contents: AttachmentContents,
    browser_visible: bool,
}

/// One probe result supplies both freshness and the resolved source name.
#[derive(Clone, Debug, PartialEq)]
pub struct AttachmentStatus {
    pub status: ReferenceStatus,
    pub resolved_file_name: Option<String>,
}

/// A refused attachment operation with input context and retained I/O/identity cause.
#[derive(Debug)]
pub enum AttachmentError {
    Invalid(String),
    File { path: String, cause: io::Error },
    Identity(IdentityTransitionError),
}

impl FileAttachment {
    /// Inspect the unique collection key. Example: `assert_eq!(attachment.name(), "loads");`.
    pub fn name(&self) -> &str {
        &self.name
    }
    /// Inspect the recorded source, without resolving or reading it. Example: `attachment.full_file_name()`.
    pub fn full_file_name(&self) -> &str {
        &self.full_file_name
    }
    /// Borrow captured state without sharing mutable payload ownership. Example: `attachment.contents()`.
    pub fn contents(&self) -> &AttachmentContents {
        &self.contents
    }
    /// Inspect browser visibility. Example: `assert!(attachment.browser_visible());`.
    pub fn browser_visible(&self) -> bool {
        self.browser_visible
    }
    /// Return the canonical API kind. Example: `assert_eq!(attachment.kind(), ATTACHMENT_LINKED);`.
    pub fn kind(&self) -> AttachmentKind {
        match self.contents {
            AttachmentContents::Generic => ATTACHMENT_GENERIC,
            AttachmentContents::Linked { .. } => ATTACHMENT_LINKED,
            AttachmentContents::Embedded { .. } => ATTACHMENT_EMBEDDED,
        }
    }

    /// Classify one source observation; embedded content never probes its original path.
    /// Example: `let state = attachment.probe(Some(&mut files))?;`.
    pub fn probe(
        &self,
        files: Option<&mut dyn ExternalAttachmentFiles>,
    ) -> Result<AttachmentStatus, AttachmentError> {
        if matches!(self.contents, AttachmentContents::Embedded { .. }) {
            return Ok(status(REFERENCE_UP_TO_DATE, None));
        }
        let Some(files) = files else {
            return Ok(status(REFERENCE_UNKNOWN, None));
        };
        let modified = files
            .modified(&self.full_file_name)
            .map_err(|cause| file_error(&self.full_file_name, cause))?;
        let Some(modified) = modified else {
            return Ok(status(REFERENCE_MISSING, None));
        };
        let stale = matches!(self.contents, AttachmentContents::Linked { last_known: Some(previous) } if modified > previous);
        Ok(status(
            if stale {
                REFERENCE_OUT_OF_DATE
            } else {
                REFERENCE_UP_TO_DATE
            },
            Some(self.full_file_name.clone()),
        ))
    }
}

fn status(status: ReferenceStatus, resolved_file_name: Option<String>) -> AttachmentStatus {
    AttachmentStatus {
        status,
        resolved_file_name,
    }
}

fn file_error(path: &str, cause: io::Error) -> AttachmentError {
    AttachmentError::File {
        path: path.into(),
        cause,
    }
}

impl fmt::Display for AttachmentError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Invalid(message) => message.fmt(formatter),
            Self::File { path, cause } => write!(
                formatter,
                "attachment source {path:?}: expected readable regular-file metadata or bytes: {cause}"
            ),
            Self::Identity(cause) => write!(
                formatter,
                "embedded attachment: expected a fresh resource GUID: {cause}"
            ),
        }
    }
}
impl Error for AttachmentError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Invalid(_) => None,
            Self::File { cause, .. } => Some(cause),
            Self::Identity(cause) => Some(cause),
        }
    }
}
