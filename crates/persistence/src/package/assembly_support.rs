// SPDX-License-Identifier: GPL-2.0-only
//! Inspect and capture occurrence recipes while live assembly support is developed.
use super::{DocumentPackage, Mapping, PackageCodecError, Value};
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
    /// Prepare occurrence edits while retaining the rest of the package; example: `package.with_assembly_occurrences(&records)?` before an atomic save.
    pub fn with_assembly_occurrences(
        &self,
        records: &[AssemblyOccurrenceRecipe],
    ) -> Result<Self, PackageCodecError> {
        let previous = self.assembly_occurrences()?.ok_or_else(|| {
            PackageCodecError(format!(
                "document type {}: expected an assembly",
                self.manifest.document_type
            ))
        })?;
        for record in records {
            record.validate().map_err(PackageCodecError)?;
        }
        if previous == records {
            return Ok(self.clone());
        }
        self.require_safe_occurrence_change(&previous, records)?;
        let encoded = serde_yaml_ng::to_value(records).map_err(|cause| PackageCodecError(format!("assembly occurrences {records:?}: expected serializable occurrence records: {cause}")))?;
        let mut candidate = self.clone();
        let mut model = candidate
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_else(Mapping::new);
        model.insert("occurrences".into(), encoded);
        candidate.root.insert("model".into(), Value::Mapping(model));
        Ok(candidate)
    }
    fn require_safe_occurrence_change(
        &self,
        previous: &[AssemblyOccurrenceRecipe],
        records: &[AssemblyOccurrenceRecipe],
    ) -> Result<(), PackageCodecError> {
        for name in ["patterns", "features"] {
            if let Some(value) = self.root.get("model").and_then(|model| model.get(name))
                && !value.is_null()
                && !value.as_sequence().is_some_and(Vec::is_empty)
            {
                return Err(PackageCodecError(format!(
                    "assembly model.{name} {value:?}: expected no dependent entries before editing occurrences"
                )));
            }
        }
        if component_references(previous) != component_references(records) {
            return Err(PackageCodecError("assembly occurrence component references changed: expected the reference graph owner to perform this edit".into()));
        }
        Ok(())
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
fn component_references(records: &[AssemblyOccurrenceRecipe]) -> std::collections::BTreeSet<&str> {
    records
        .iter()
        .filter(|record| !record.virtual_ && !record.component.is_empty())
        .map(|record| record.component.as_str())
        .collect()
}
fn unsupported_assembly_content(name: &str, value: &Value) -> PackageCodecError {
    PackageCodecError(format!(
        "assembly model.{name} {value:?}: expected an empty sequence until assembly occurrence reconstruction is supported"
    ))
}
