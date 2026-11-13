// SPDX-License-Identifier: GPL-2.0-only
//! Refuse assembly content until occurrence reconstruction is available.
use super::{DocumentPackage, PackageCodecError, Value};
use oblikovati_api::types::DOCUMENT_ASSEMBLY;

impl DocumentPackage {
    pub(crate) fn require_supported_assembly(&self) -> Result<(), PackageCodecError> {
        if self.manifest.document_type != DOCUMENT_ASSEMBLY.0 {
            return Ok(());
        }
        for name in ["occurrences", "patterns"] {
            let Some(value) = self.root.get("model").and_then(|model| model.get(name)) else {
                continue;
            };
            if !value.is_null() && !value.as_sequence().is_some_and(Vec::is_empty) {
                return Err(unsupported_assembly_content(name, value));
            }
        }
        Ok(())
    }
}
fn unsupported_assembly_content(name: &str, value: &Value) -> PackageCodecError {
    PackageCodecError(format!(
        "assembly model.{name} {value:?}: expected an empty sequence until assembly occurrence reconstruction is supported"
    ))
}
