// SPDX-License-Identifier: GPL-2.0-only
mod assign;
use super::{AppearanceAsset, AssetAssignments, AssetCollection, MaterialAsset};
pub use assign::{AppearanceAssignmentTarget, MaterialAssignmentTarget};
use oblikovati_api::{
    contract::{Appearance, Material},
    types::{ASSET_BUILTIN, ASSET_DOCUMENT},
};
/// A portable document owns asset snapshots and assignment intent together.
#[derive(Clone, Debug, Default)]
pub struct DocumentAssets {
    materials: AssetCollection<MaterialAsset>,
    appearances: AssetCollection<AppearanceAsset>,
    pub assignments: AssetAssignments,
}
impl DocumentAssets {
    /// Embed an independent non-builtin physical definition; example: `assets.embed_material(&material)?`.
    pub fn embed_material(&mut self, asset: &MaterialAsset) -> Result<bool, String> {
        if asset.source() == ASSET_BUILTIN {
            return Ok(false);
        }
        let mut record = asset.info();
        record.source = ASSET_DOCUMENT.0.into_owned();
        self.materials.insert(MaterialAsset::from_info(record)?);
        Ok(true)
    }
    /// Embed an independent non-builtin visual definition; example: `assets.embed_appearance(&appearance)?`.
    pub fn embed_appearance(&mut self, asset: &AppearanceAsset) -> Result<bool, String> {
        if asset.source() == ASSET_BUILTIN {
            return Ok(false);
        }
        let mut record = asset.info();
        record.source = ASSET_DOCUMENT.0.into_owned();
        self.appearances.insert(AppearanceAsset::from_info(record)?);
        Ok(true)
    }
    /// Borrow document physical definitions; example: `assets.materials().get(id)`.
    pub fn materials(&self) -> &AssetCollection<MaterialAsset> {
        &self.materials
    }
    /// Borrow document visual definitions; example: `assets.appearances().get(id)`.
    pub fn appearances(&self) -> &AssetCollection<AppearanceAsset> {
        &self.appearances
    }
    /// Test whether the document has no authored asset content; example: `assets.is_empty()` for recipe omission.
    pub fn is_empty(&self) -> bool {
        self.materials.ordered().next().is_none()
            && self.appearances.ordered().next().is_none()
            && self.assignments == AssetAssignments::default()
    }
}
impl PartialEq for DocumentAssets {
    fn eq(&self, other: &Self) -> bool {
        // Document definitions are a set; category display order belongs to the shared catalog.
        self.materials.ordered().eq(other.materials.ordered())
            && self.appearances.ordered().eq(other.appearances.ordered())
            && self.assignments == other.assignments
    }
}
