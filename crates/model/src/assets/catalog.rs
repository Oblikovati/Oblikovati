// SPDX-License-Identifier: GPL-2.0-only
use super::catalog_validation::{validate_builtin_appearance, validate_builtin_material};
use super::{AppearanceAsset, AssetCollection, AssetIdentity, MaterialAsset};
/// Stable neutral ID from the Go material subsystem; not a renderer's guessed color.
pub const DEFAULT_APPEARANCE_ID: &str = "default";
/// Complete immutable shipped assets, admitted together so missing links never escape.
#[derive(Clone, Debug)]
pub struct BuiltinAssetCatalog {
    materials: AssetCollection<MaterialAsset>,
    appearances: AssetCollection<AppearanceAsset>,
}
impl BuiltinAssetCatalog {
    /// Admit one complete catalog or refuse it; example: `BuiltinAssetCatalog::from_records(materials, appearances)?`.
    pub fn from_records(
        materials: Vec<MaterialAsset>,
        appearances: Vec<AppearanceAsset>,
    ) -> Result<Self, String> {
        for asset in &materials {
            validate_builtin_material(asset)?;
        }
        for asset in &appearances {
            validate_builtin_appearance(asset)?;
        }
        let result = Self {
            materials: unique_builtin_records(materials)?,
            appearances: unique_builtin_records(appearances)?,
        };
        result.require_catalog_links()?;
        Ok(result)
    }
    /// Borrow physical snapshots for resolution; example: `catalog.materials().get("steel")`.
    pub fn materials(&self) -> &AssetCollection<MaterialAsset> {
        &self.materials
    }
    /// Borrow visual snapshots for resolution; example: `catalog.appearances().get("default")`.
    pub fn appearances(&self) -> &AssetCollection<AppearanceAsset> {
        &self.appearances
    }
    fn require_catalog_links(&self) -> Result<(), String> {
        if self.appearances.get(DEFAULT_APPEARANCE_ID).is_none() {
            return Err(format!(
                "catalog appearance {DEFAULT_APPEARANCE_ID:?}: expected the neutral shipped asset"
            ));
        }
        for asset in self.materials.ordered() {
            require_material_appearance(asset, &self.appearances)?;
        }
        Ok(())
    }
}
fn unique_builtin_records<T: AssetIdentity>(records: Vec<T>) -> Result<AssetCollection<T>, String> {
    let mut scope = AssetCollection::default();
    for record in records {
        if scope.get(record.asset_id()).is_some() {
            return Err(format!(
                "catalog asset {:?}: expected a unique identity within its asset family",
                record.asset_id()
            ));
        }
        scope.insert(record);
    }
    Ok(scope)
}
fn require_material_appearance(
    asset: &MaterialAsset,
    appearances: &AssetCollection<AppearanceAsset>,
) -> Result<(), String> {
    use oblikovati_api::contract::Material;
    if appearances.get(&asset.appearance_id()).is_none() {
        return Err(format!(
            "catalog material {:?} appearance {:?}: expected a shipped appearance identity",
            asset.id(),
            asset.appearance_id()
        ));
    }
    Ok(())
}
