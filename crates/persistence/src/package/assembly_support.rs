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
    pub(crate) fn assembly_occurrences_for_open(
        &self,
    ) -> Result<Option<Vec<AssemblyOccurrenceRecipe>>, PackageCodecError> {
        let mut records = self.assembly_occurrences()?;
        if let Some(records) = &mut records {
            oblikovati_model::assembly::assign_open_occurrence_ids(records)
                .map_err(PackageCodecError)?;
        }
        Ok(records)
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
        if same_authored_occurrences(&previous, records) {
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
    /// Refuse live reconstruction until components can be resolved; example: `package.require_supported_assembly()?` before registration.
    pub fn require_supported_assembly(&self) -> Result<(), PackageCodecError> {
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
    /// Check whether saved assembly content can be reconstructed on open.
    /// Example: `package.require_loadable_assembly()?` before registration.
    pub fn require_loadable_assembly(&self) -> Result<(), PackageCodecError> {
        let Some(records) = self.assembly_occurrences()? else {
            return Ok(());
        };
        for name in ["patterns", "features"] {
            let Some(value) = self.root.get("model").and_then(|model| model.get(name)) else {
                continue;
            };
            if !value.is_null() && !value.as_sequence().is_some_and(Vec::is_empty) {
                return Err(unsupported_assembly_content(name, value));
            }
        }
        for record in records {
            if record.adaptive
                || record.flexible
                || record.substitute
                || !record.child_transforms.is_empty()
            {
                return Err(PackageCodecError(format!(
                    "assembly occurrence {:?}: expected no adaptive, flexible, substitute, or child transform state",
                    record.name
                )));
            }
        }
        Ok(())
    }
}
fn same_authored_occurrences(
    left: &[AssemblyOccurrenceRecipe],
    right: &[AssemblyOccurrenceRecipe],
) -> bool {
    left.len() == right.len()
        && left.iter().zip(right).all(|(a, b)| {
            let mut other = b.clone();
            other.session_id = a.session_id;
            *a == other
        })
}
fn component_references(records: &[AssemblyOccurrenceRecipe]) -> std::collections::BTreeSet<&str> {
    oblikovati_model::assembly::component_references(records)
        .into_iter()
        .collect()
}
fn unsupported_assembly_content(name: &str, value: &Value) -> PackageCodecError {
    PackageCodecError(format!(
        "assembly model.{name} {value:?}: expected an empty sequence until assembly occurrence reconstruction is supported"
    ))
}
