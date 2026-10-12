// SPDX-License-Identifier: GPL-2.0-only

use serde::Deserialize;
use serde_yaml_ng::{Mapping, Value};
use std::collections::BTreeMap;
use std::{error::Error, fmt};

mod attachments;
pub use attachments::format_attachment_time;
mod attributes;
mod binary_text;
mod container;
mod copy_metadata;
mod empty;
mod interests;
mod parameters;
mod properties;
mod resource_edit;
mod resource_import;
pub use resource_import::ResourceImportError;
mod resource_read;
pub use resource_read::{ApplicationResources, ResourceReadError};
mod resources;
mod streams;
mod units;
pub use parameters::RestoredParameters;
mod save;
mod sketches;
mod work_planes;
pub use work_planes::{RestoredWorkGeometry, RestoredWorkGeometry as RestoredWorkPlanes};
mod sketch_settings;
mod snapshot;
pub use sketches::RestoredDocumentSketches;
pub use snapshot::{ParameterPackageTarget, SnapshotPackageError};
mod validation;
pub use oblikovati_model::resource::{DocumentResource as PackageResource, ResourceContents};
pub use save::{PackageSaveError, PackageSaveTarget};

/// File-level metadata. Document kind remains a numeric wire value until host validation.
#[derive(Clone, Debug, Deserialize, Eq, PartialEq)]
#[serde(rename_all = "camelCase")]
pub struct PackageManifest {
    pub schema_version: u32,
    pub document_type: u32,
    #[serde(default)]
    pub sub_type: String,
    #[serde(default)]
    pub display_name: String,
}

/// A versioned YAML package with native YAML sections retained behind this boundary.
/// Reading an envelope does not authorize executing or editing its recipe.
#[derive(Clone, Debug, PartialEq)]
pub struct DocumentPackage {
    manifest: PackageManifest,
    root: Mapping,
    streams: BTreeMap<String, Vec<u8>>,
    resources: BTreeMap<String, PackageResource>,
}

/// A refused package with an input-specific diagnostic.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct PackageCodecError(String);

impl DocumentPackage {
    /// Decode exactly one supported YAML mapping without dropping known sections.
    ///
    /// ```
    /// use oblikovati_persistence::package::DocumentPackage;
    /// let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")?;
    /// assert_eq!(package.manifest().document_type, 1);
    /// # Ok::<(), oblikovati_persistence::package::PackageCodecError>(())
    /// ```
    pub fn decode(encoded: &[u8]) -> Result<Self, PackageCodecError> {
        if encoded.starts_with(b"PK\x03\x04") {
            return Err(PackageCodecError(
                "legacy ZIP input: expected schema-2 YAML".into(),
            ));
        }
        let root: Mapping = serde_yaml_ng::from_slice(encoded).map_err(codec_error)?;
        validation::validate_root(&root)?;
        let manifest: PackageManifest =
            serde_yaml_ng::from_value(Value::Mapping(root.clone())).map_err(codec_error)?;
        if !matches!(manifest.schema_version, 2 | 3) {
            return Err(PackageCodecError(format!(
                "schemaVersion {}: expected supported schema 2 or 3",
                manifest.schema_version
            )));
        }
        Self::from_root(root, manifest)
    }

    fn from_root(root: Mapping, manifest: PackageManifest) -> Result<Self, PackageCodecError> {
        validation::validate_root(&root)?;
        let streams = resources::decode_streams(&root)?;
        let resources = resources::decode_resources(&root)?;
        Ok(Self {
            manifest,
            root,
            streams,
            resources,
        })
    }

    /// Encode retained section values as native YAML, without interpreting geometry.
    ///
    /// ```
    /// use oblikovati_persistence::package::DocumentPackage;
    /// let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")?;
    /// assert_eq!(DocumentPackage::decode(&package.encode()?)?, package);
    /// # Ok::<(), oblikovati_persistence::package::PackageCodecError>(())
    /// ```
    pub fn encode(&self) -> Result<Vec<u8>, PackageCodecError> {
        serde_yaml_ng::to_string(&self.root)
            .map(String::into_bytes)
            .map_err(codec_error)
    }

    /// Inspect metadata without bypassing package validation.
    /// Example: `let name = &package.manifest().display_name;`.
    pub fn manifest(&self) -> &PackageManifest {
        &self.manifest
    }

    /// Read decoded add-in data without changing the preserved file representation.
    /// Example: `let payload = package.streams().get("addin/state");`.
    pub fn streams(&self) -> &BTreeMap<String, Vec<u8>> {
        &self.streams
    }

    /// Inspect imported resources with encoding already classified and validated.
    /// Example: `let mesh = package.resources().get("mesh");`.
    pub fn resources(&self) -> &BTreeMap<String, PackageResource> {
        &self.resources
    }

    /// Extract a section for its domain decoder, retaining its native YAML shape.
    /// Example: `let recipe = package.section_yaml("model")?;`.
    pub fn section_yaml(&self, name: &str) -> Result<Option<Vec<u8>>, PackageCodecError> {
        self.root
            .get(Value::String(name.to_owned()))
            .map(|section| {
                serde_yaml_ng::to_string(section)
                    .map(String::into_bytes)
                    .map_err(codec_error)
            })
            .transpose()
    }
}

fn codec_error(error: serde_yaml_ng::Error) -> PackageCodecError {
    PackageCodecError(format!(
        "YAML input: expected one valid schema-2 mapping: {error}"
    ))
}

impl fmt::Display for PackageCodecError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(formatter)
    }
}

impl Error for PackageCodecError {}

pub(crate) mod assets;
