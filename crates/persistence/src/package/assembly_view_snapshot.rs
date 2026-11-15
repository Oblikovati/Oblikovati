// SPDX-License-Identifier: GPL-2.0-only
//! Assembly display snapshots are not modelling geometry.
use super::{DocumentPackage, PackageCodecError};
use oblikovati_model::{assembly::AssemblyOccurrenceRecipe, identity::RecipeDigest};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

const SECTION: &str = "cache/assembly-view-v1";

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
struct AssemblyViewEnvelope {
    content: String,
    digest: String,
}

/// One occurrence edge and the exact source used for its presentation.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct AssemblySourceRecord {
    pub occurrence_path: Vec<usize>,
    pub reference: String,
    pub resolved_path: PathBuf,
    pub identity: String,
    pub digest: String,
}

/// A display triangle, in assembly coordinates and linear color.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct SavedAssemblyTriangle {
    pub positions: [[f32; 3]; 3],
    pub color: [f32; 4],
}

/// A versioned display-only assembly snapshot and its source provenance.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct SavedAssemblyView {
    pub version: u32,
    pub occurrences: Vec<AssemblyOccurrenceRecipe>,
    pub sources: Vec<AssemblySourceRecord>,
    pub catalog_digest: String,
    pub chord_error: f64,
    pub angular_error: f64,
    pub triangles: Vec<SavedAssemblyTriangle>,
}
impl SavedAssemblyView {
    /// Current format; example: `view.version == SavedAssemblyView::VERSION`.
    pub const VERSION: u32 = 1;
    /// Validate the display data and manifest shape before a save is attempted.
    /// Example: `view.validate_content()?` before staging a cache.
    pub fn validate_content(&self) -> Result<(), PackageCodecError> {
        validate_content(self)
    }

    /// Check that the saved mesh meets the current display limits.
    /// Example: `view.validate_display_quality(0.05, 0.5)?` before display.
    pub fn validate_display_quality(
        &self,
        chord_limit: f64,
        angular_limit: f64,
    ) -> Result<(), PackageCodecError> {
        for (name, actual, limit) in [
            ("chord", self.chord_error, chord_limit),
            ("angular", self.angular_error, angular_limit),
        ] {
            if !limit.is_finite()
                || limit <= 0.
                || !actual.is_finite()
                || actual <= 0.
                || actual > limit
            {
                return Err(PackageCodecError(format!(
                    "assembly view {name} error {actual}: expected a positive finite value at most {limit} (positive and finite)"
                )));
            }
        }
        Ok(())
    }
}
impl DocumentPackage {
    /// Store display data with its authored assembly; example: `package.with_saved_assembly_view(&view)?`.
    pub fn with_saved_assembly_view(
        &self,
        view: &SavedAssemblyView,
    ) -> Result<Self, PackageCodecError> {
        validate_view(self, view)?;
        let content = serde_yaml_ng::to_string(view).map_err(|error| {
            PackageCodecError(format!(
                "assembly view: expected serializable cache: {error}"
            ))
        })?;
        let envelope = AssemblyViewEnvelope {
            digest: RecipeDigest::of_recipe(content.as_bytes()).to_string(),
            content,
        };
        let encoded = serde_yaml_ng::to_string(&envelope).map_err(|error| {
            PackageCodecError(format!(
                "assembly view envelope: expected serializable content: {error}"
            ))
        })?;
        Ok(self.with_stream(SECTION, encoded.as_bytes()))
    }
    /// Read a locally validated display snapshot; source files must still be checked before use.
    /// Example: `package.restore_saved_assembly_view()?` before validating its source graph.
    pub fn restore_saved_assembly_view(
        &self,
    ) -> Result<Option<SavedAssemblyView>, PackageCodecError> {
        let Some(bytes) = self.streams().get(SECTION) else {
            return Ok(None);
        };
        let envelope: AssemblyViewEnvelope = serde_yaml_ng::from_slice(bytes).map_err(|error| {
            PackageCodecError(format!(
                "assembly view stream {SECTION:?}: expected a versioned envelope: {error}"
            ))
        })?;
        if !is_sha256(&envelope.digest)
            || RecipeDigest::of_recipe(envelope.content.as_bytes()).to_string() != envelope.digest
        {
            return Err(PackageCodecError(format!(
                "assembly view stream {SECTION:?}: expected its content checksum"
            )));
        }
        let view = serde_yaml_ng::from_str(&envelope.content).map_err(|error| {
            PackageCodecError(format!(
                "assembly view stream {SECTION:?}: expected versioned content: {error}"
            ))
        })?;
        validate_view(self, &view)?;
        Ok(Some(view))
    }
    pub(super) fn without_saved_assembly_view(&self) -> Self {
        self.without_stream(SECTION)
    }
}
fn validate_view(
    package: &DocumentPackage,
    view: &SavedAssemblyView,
) -> Result<(), PackageCodecError> {
    let occurrences = package
        .assembly_occurrences()?
        .ok_or_else(|| PackageCodecError("assembly view: expected an assembly document".into()))?;
    let mut expected = view.occurrences.clone();
    for item in &mut expected {
        item.session_id = 0;
    }
    if view.version != SavedAssemblyView::VERSION || expected != occurrences {
        return Err(PackageCodecError(format!(
            "assembly view version {} and occurrences: expected version {} and the saved assembly recipe",
            view.version,
            SavedAssemblyView::VERSION
        )));
    }
    validate_content(view)
}

fn validate_content(view: &SavedAssemblyView) -> Result<(), PackageCodecError> {
    if view.version != SavedAssemblyView::VERSION
        || !view.chord_error.is_finite()
        || view.chord_error <= 0.
        || !view.angular_error.is_finite()
        || view.angular_error <= 0.
        || !is_sha256(&view.catalog_digest)
    {
        return Err(PackageCodecError(format!(
            "assembly view version {}, mesh and catalog {:?}: expected a supported version, finite positive mesh limits and a SHA-256 digest",
            view.version, view.catalog_digest
        )));
    }
    for occurrence in &view.occurrences {
        occurrence.validate().map_err(PackageCodecError)?;
    }
    validate_sources(&view.occurrences, &view.sources)?;
    for (index, triangle) in view.triangles.iter().enumerate() {
        if triangle.positions.iter().flatten().any(|v| !v.is_finite())
            || triangle
                .color
                .iter()
                .any(|v| !v.is_finite() || !(0.0..=1.0).contains(v))
        {
            return Err(PackageCodecError(format!(
                "assembly view triangle {index}: expected finite positions and RGBA colors in 0..=1"
            )));
        }
    }
    Ok(())
}

fn validate_sources(
    occurrences: &[AssemblyOccurrenceRecipe],
    sources: &[AssemblySourceRecord],
) -> Result<(), PackageCodecError> {
    let mut paths = std::collections::BTreeSet::new();
    let mut previous: Option<&[usize]> = None;
    for source in sources {
        let path = source.occurrence_path.as_slice();
        if path.is_empty()
            || path.len() > 128
            || previous.is_some_and(|item| item >= path)
            || !paths.insert(path.to_vec())
        {
            return Err(PackageCodecError(format!(
                "assembly source path {path:?}: expected a unique ordered occurrence path"
            )));
        }
        if path.len() > 1 && !paths.contains(&path[..path.len() - 1]) {
            return Err(PackageCodecError(format!(
                "assembly source path {path:?}: expected its parent source"
            )));
        }
        if !source.resolved_path.is_absolute()
            || source
                .identity
                .parse::<oblikovati_model::identity::DocumentGuid>()
                .is_err()
            || !is_sha256(&source.digest)
            || source.reference.is_empty()
        {
            return Err(PackageCodecError(format!(
                "assembly source {source:?}: expected an absolute path, identity, reference and SHA-256 digest"
            )));
        }
        previous = Some(path);
    }
    for (ordinal, occurrence) in occurrences
        .iter()
        .enumerate()
        .filter(|(_, item)| !item.virtual_)
    {
        let path = [ordinal];
        let source = sources
            .iter()
            .find(|source| source.occurrence_path == path)
            .ok_or_else(|| {
                PackageCodecError(format!(
                    "assembly occurrence {ordinal}: expected a source record"
                ))
            })?;
        if source.reference != occurrence.component {
            return Err(PackageCodecError(format!(
                "assembly occurrence {ordinal} source {:?}: expected {:?}",
                source.reference, occurrence.component
            )));
        }
    }
    if sources.iter().any(|source| {
        source.occurrence_path.len() == 1
            && occurrences
                .get(source.occurrence_path[0])
                .is_none_or(|item| item.virtual_)
    }) {
        return Err(PackageCodecError(
            "assembly source path: expected an authored nonvirtual occurrence".into(),
        ));
    }
    Ok(())
}
fn is_sha256(value: &str) -> bool {
    value.len() == 64
        && value
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
}

#[cfg(test)]
#[path = "../../tests/application/assembly_view_snapshot.rs"]
mod tests;
