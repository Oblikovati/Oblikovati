// SPDX-License-Identifier: GPL-2.0-only
//! Inspect and capture occurrence recipes while live assembly support is developed.
use super::{DocumentPackage, Mapping, PackageCodecError, Value};
use oblikovati_api::types::DOCUMENT_ASSEMBLY;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

impl DocumentPackage {
    /// Read the Go assembly's first-placement option; example: `package.ground_first_component()?`.
    pub fn ground_first_component(&self) -> Result<bool, PackageCodecError> {
        if self.manifest.document_type != DOCUMENT_ASSEMBLY.0 {
            return Err(PackageCodecError(format!(
                "document type {}: expected an assembly",
                self.manifest.document_type
            )));
        }
        let Some(options) = self
            .root
            .get("model")
            .and_then(|model| model.get("options"))
        else {
            return Ok(true);
        };
        if options.is_null() {
            return Ok(true);
        }
        let mapping = options.as_mapping().ok_or_else(|| {
            PackageCodecError(format!("assembly options {options:?}: expected a mapping"))
        })?;
        let Some(value) = mapping.get("placeAndGroundFirstComponentAtOrigin") else {
            return Ok(false);
        };
        value.as_bool().ok_or_else(|| {
            PackageCodecError(format!(
                "assembly placeAndGroundFirstComponentAtOrigin {value:?}: expected a boolean"
            ))
        })
    }
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
        self.capture_assembly_occurrences(records, None)
    }
    pub(super) fn with_owned_assembly_occurrences(
        &self,
        records: &[AssemblyOccurrenceRecipe],
        references: &std::collections::BTreeSet<String>,
    ) -> Result<Self, PackageCodecError> {
        self.capture_assembly_occurrences(records, Some(references))
    }
    fn capture_assembly_occurrences(
        &self,
        records: &[AssemblyOccurrenceRecipe],
        references: Option<&std::collections::BTreeSet<String>>,
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
        self.require_safe_occurrence_change(&previous, records, references)?;
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
        references: Option<&std::collections::BTreeSet<String>>,
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
        validate_reference_changes(previous, records, references)
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
fn validate_reference_changes(
    previous: &[AssemblyOccurrenceRecipe],
    records: &[AssemblyOccurrenceRecipe],
    authorized: Option<&std::collections::BTreeSet<String>>,
) -> Result<(), PackageCodecError> {
    let before = component_references(previous);
    let after = component_references(records);
    if before == after {
        return Ok(());
    }
    let Some(authorized) = authorized else {
        return Err(PackageCodecError("assembly occurrence component references changed: expected the reference graph owner to perform this edit".into()));
    };
    for added in after.difference(&before) {
        if !authorized.contains(*added) {
            return Err(PackageCodecError(format!(
                "assembly component reference {added:?}: expected an owned reference to its current file"
            )));
        }
    }
    Ok(())
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
