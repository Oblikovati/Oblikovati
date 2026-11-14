// SPDX-License-Identifier: GPL-2.0-only
use crate::load::PackageFileReader;
use oblikovati_api::types::ASSET_PROJECT;
use oblikovati_model::assets::ProjectAssetLibrary;
use serde::Deserialize;
use serde_yaml_ng::Value;
use std::{io, path::Path};
#[derive(Default, Deserialize)]
#[serde(default, deny_unknown_fields)]
pub(super) struct ProjectAssetFile {
    #[serde(deserialize_with = "null_entries")]
    pub materials: Vec<Value>,
    #[serde(deserialize_with = "null_entries")]
    pub appearances: Vec<Value>,
}
/// Load both libraries before admission; example: `load_project_asset_library(project.bound.design_data.as_deref(), files)?`.
pub fn load_project_asset_library(
    directory: Option<&Path>,
    files: &mut dyn PackageFileReader,
) -> Result<ProjectAssetLibrary, String> {
    let Some(directory) = directory else {
        return Ok(ProjectAssetLibrary::default());
    };
    let appearance_path = directory.join("appearances.yaml");
    let visuals = read_project_assets(&appearance_path, files)?;
    let material_path = directory.join("materials.yaml");
    let physicals = read_project_assets(&material_path, files)?;
    if !visuals.materials.is_empty() || !physicals.appearances.is_empty() {
        return Err(format!(
            "project library {directory:?}: expected appearances only in appearances.yaml and materials only in materials.yaml"
        ));
    }
    let appearances =
        crate::package::assets::decode_appearance_definitions(&visuals.appearances, ASSET_PROJECT)
            .map_err(|cause| format!("project asset file {appearance_path:?}: {cause}"))?;
    let materials =
        crate::package::assets::decode_material_definitions(&physicals.materials, ASSET_PROJECT)
            .map_err(|cause| format!("project asset file {material_path:?}: {cause}"))?;
    ProjectAssetLibrary::from_records(materials, appearances)
}
pub(super) fn read_project_assets(
    path: &Path,
    files: &mut dyn PackageFileReader,
) -> Result<ProjectAssetFile, String> {
    let bytes = match files.read(path) {
        Ok(bytes) => bytes,
        Err(cause) if cause.kind() == io::ErrorKind::NotFound => {
            return Ok(ProjectAssetFile::default());
        }
        Err(cause) => {
            return Err(format!(
                "project asset file {path:?}: expected readable YAML or an absent library: {cause}"
            ));
        }
    };
    serde_yaml_ng::from_slice(&bytes).map_err(|cause| {
        format!("project asset file {path:?}: expected typed material/appearance lists: {cause}")
    })
}
fn null_entries<'de, D: serde::Deserializer<'de>>(deserializer: D) -> Result<Vec<Value>, D::Error> {
    Option::<Vec<Value>>::deserialize(deserializer).map(Option::unwrap_or_default)
}
