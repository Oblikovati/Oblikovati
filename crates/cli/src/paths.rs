// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::*;
use std::{
    ffi::OsStr,
    path::{Path, PathBuf},
};

pub(super) fn kind_extension(kind: DocumentType) -> Result<&'static str, String> {
    match kind {
        DOCUMENT_PART => Ok(PART_FILE_EXTENSION),
        DOCUMENT_ASSEMBLY => Ok(ASSEMBLY_FILE_EXTENSION),
        DOCUMENT_DRAWING => Ok(DRAWING_FILE_EXTENSION),
        DOCUMENT_PRESENTATION => Ok(PRESENTATION_FILE_EXTENSION),
        _ => Err(format!(
            "document type {kind:?}: expected part, assembly, drawing, or presentation"
        )),
    }
}
pub(super) fn with_extension(path: &OsStr, extension: &str) -> PathBuf {
    let current = PathBuf::from(path);
    if matching_extension(&current, extension) {
        return current;
    }
    let mut stamped = path.to_os_string();
    stamped.push(extension);
    PathBuf::from(stamped)
}
fn matching_extension(path: &Path, extension: &str) -> bool {
    // Go filepath.Ext includes a leading dot; Path::extension omits dotfile extensions.
    let name = path.file_name().unwrap_or_default().as_encoded_bytes();
    let Some(dot) = name.iter().rposition(|byte| *byte == b'.') else {
        return false;
    };
    name[dot..].eq_ignore_ascii_case(extension.as_bytes())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn unsupported_document_kind_has_a_value_specific_decline() {
        let error = kind_extension(DocumentType(42)).unwrap_err();
        assert!(error.contains("42") && error.contains("expected part"));
    }
}
