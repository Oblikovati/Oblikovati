// SPDX-License-Identifier: GPL-2.0-only
use super::{AppearanceAsset, AssetCollection, AssetIdentity, MaterialAsset};
use oblikovati_api::{
    contract::{Appearance, Material},
    types::ASSET_PROJECT,
};
/// One project's immutable definitions, distinct from document snapshots and shipped assets.
#[derive(Clone, Debug, Default)]
pub struct ProjectAssetLibrary {
    materials: AssetCollection<MaterialAsset>,
    appearances: AssetCollection<AppearanceAsset>,
}
impl ProjectAssetLibrary {
    /// Admit project definitions with unique IDs and explicit project ownership; example: `ProjectAssetLibrary::from_records(materials, appearances)?`.
    pub fn from_records(
        materials: Vec<MaterialAsset>,
        appearances: Vec<AppearanceAsset>,
    ) -> Result<Self, String> {
        for asset in &materials {
            require_project_source(asset.asset_id(), asset.source())?;
        }
        for asset in &appearances {
            require_project_source(asset.asset_id(), asset.source())?;
        }
        Ok(Self {
            materials: AssetCollection::from_unique(materials)?,
            appearances: AssetCollection::from_unique(appearances)?,
        })
    }
    /// Borrow physical project definitions; example: `project.materials().get(id)`.
    pub fn materials(&self) -> &AssetCollection<MaterialAsset> {
        &self.materials
    }
    /// Borrow visual project definitions; example: `project.appearances().get(id)`.
    pub fn appearances(&self) -> &AssetCollection<AppearanceAsset> {
        &self.appearances
    }

    /// Prepare an owned physical replacement without changing this snapshot; example: `let next = project.with_material(asset)?;`.
    pub fn with_material(&self, asset: MaterialAsset) -> Result<Self, String> {
        require_project_source(asset.asset_id(), asset.source())?;
        let mut candidate = self.clone();
        candidate.materials.insert(asset);
        Ok(candidate)
    }

    /// Prepare an owned visual replacement without changing this snapshot; example: `let next = project.with_appearance(asset)?;`.
    pub fn with_appearance(&self, asset: AppearanceAsset) -> Result<Self, String> {
        require_project_source(asset.asset_id(), asset.source())?;
        let mut candidate = self.clone();
        candidate.appearances.insert(asset);
        Ok(candidate)
    }
}
impl PartialEq for ProjectAssetLibrary {
    fn eq(&self, other: &Self) -> bool {
        self.materials.ordered().eq(other.materials.ordered())
            && self.appearances.ordered().eq(other.appearances.ordered())
    }
}
fn require_project_source(
    id: &str,
    source: oblikovati_api::types::AssetSource,
) -> Result<(), String> {
    if source != ASSET_PROJECT {
        return Err(format!(
            "project asset {id:?} source {source:?}: expected project"
        ));
    }
    Ok(())
}
