// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::attachment::ExternalAttachmentFiles;
use std::{io, time::SystemTime};

/// Native foreign-file access for attachment capture and freshness checks.
pub struct NativeAttachmentFiles;

impl ExternalAttachmentFiles for NativeAttachmentFiles {
    fn modified(&mut self, path: &str) -> io::Result<Option<SystemTime>> {
        let metadata = match std::fs::metadata(path) {
            Ok(metadata) => metadata,
            Err(error) if error.kind() == io::ErrorKind::NotFound => return Ok(None),
            Err(error) => return Err(error),
        };
        if !metadata.is_file() {
            return Ok(None);
        }
        metadata.modified().map(Some)
    }

    fn read(&mut self, path: &str) -> io::Result<Vec<u8>> {
        std::fs::read(path)
    }
}
