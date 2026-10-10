// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::attachment::{ExternalAttachmentFiles, ResolvedAttachmentSource};
use std::io;

/// Native foreign-file access for attachment capture and freshness checks.
pub struct NativeAttachmentFiles;

impl ExternalAttachmentFiles for NativeAttachmentFiles {
    fn inspect_source(&mut self, path: &str) -> io::Result<Option<ResolvedAttachmentSource>> {
        let metadata = match std::fs::metadata(path) {
            Ok(metadata) => metadata,
            Err(error)
                if matches!(
                    error.kind(),
                    io::ErrorKind::NotFound | io::ErrorKind::NotADirectory
                ) =>
            {
                return Ok(None);
            }
            Err(error) => return Err(error),
        };
        if !metadata.is_file() {
            return Ok(None);
        }
        Ok(Some(ResolvedAttachmentSource {
            resolved_file_name: path.into(),
            modified: metadata.modified()?,
        }))
    }

    fn read(&mut self, path: &str) -> io::Result<Vec<u8>> {
        std::fs::read(path)
    }
}
