// SPDX-License-Identifier: GPL-2.0-only
//! Physical and visual assets use the released API values; no renderer or mass evaluation lives here.
// Keep the two asset families on the same identity-preserving edit path.
macro_rules! asset_snapshot_methods {
    ($record_type:ty) => {
        /// Copy the released API record for a client or storage adapter; example: `asset.info()`.
        pub fn info(&self) -> $record_type {
            self.record.clone()
        }
        /// Replace editable properties while retaining identity and origin; example: `asset.edit(info)?`.
        pub fn edit(&mut self, mut record: $record_type) -> Result<bool, String> {
            if !super::validation::editable_asset_source(&self.source()) {
                return Ok(false);
            }
            record.id = self.record.id.clone();
            record.source = self.record.source.clone();
            let replacement = Self::from_info(record)?;
            *self = replacement;
            Ok(true)
        }
        /// Create an independent named copy with an injected stable identity; example: `asset.duplicate("custom", "Custom", ASSET_PROJECT)?`.
        pub fn duplicate(
            &self,
            id: &str,
            name: &str,
            source: oblikovati_api::types::AssetSource,
        ) -> Result<Self, String> {
            if name.is_empty() {
                return Err(format!(
                    "asset duplicate name {name:?}: expected nonempty text"
                ));
            }
            let mut record = self.record.clone();
            record.id = id.into();
            record.display_name = name.into();
            record.source = source.0.into_owned();
            Self::from_info(record)
        }
    };
}
mod appearance;
mod catalog;
mod catalog_validation;
mod material;
mod scopes;
mod validation;
pub use appearance::AppearanceAsset;
pub use catalog::{BuiltinAssetCatalog, DEFAULT_APPEARANCE_ID};
pub use material::MaterialAsset;
pub use scopes::{AssetCollection, AssetIdentity, ScopedAssets};
