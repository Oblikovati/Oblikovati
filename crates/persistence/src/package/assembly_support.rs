// SPDX-License-Identifier: GPL-2.0-only
//! Refuse assembly content until occurrence reconstruction is available.
use super::{DocumentPackage, PackageCodecError, Value};
use oblikovati_api::types::DOCUMENT_ASSEMBLY;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

impl DocumentPackage {
    /// Read saved occurrence recipes without opening their references; example: `package.assembly_occurrences()?`.
    pub fn assembly_occurrences(
        &self,
    ) -> Result<Option<Vec<AssemblyOccurrenceRecipe>>, PackageCodecError> {
        if self.manifest.document_type != DOCUMENT_ASSEMBLY.0 {
            return Ok(None);
        }
        let Some(value) = self
            .root
            .get("model")
            .and_then(|model| model.get("occurrences"))
        else {
            return Ok(Some(Vec::new()));
        };
        if value.is_null() {
            return Ok(Some(Vec::new()));
        }
        let records: Vec<AssemblyOccurrenceRecipe> = serde_yaml_ng::from_value(value.clone())
            .map_err(|cause| {
                PackageCodecError(format!(
                    "assembly model.occurrences {value:?}: expected Go occurrence records: {cause}"
                ))
            })?;
        for record in &records {
            record.validate().map_err(PackageCodecError)?;
        }
        Ok(Some(records))
    }
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
