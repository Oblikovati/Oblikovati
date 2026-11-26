// SPDX-License-Identifier: GPL-2.0-only
mod assign;
use super::{AppearanceAsset, AssetAssignments, AssetCollection, MaterialAsset};
pub use assign::{AppearanceAssignmentTarget, MaterialAssignmentTarget};
use oblikovati_api::{
    contract::{Appearance, Material},
    types::{ASSET_BUILTIN, ASSET_DOCUMENT},
};
/// Application-owned information and Generic editor settings for one document appearance.
#[derive(Clone, Debug, Default, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default)]
pub struct AppearanceMetadata {
    pub description: String,
    pub keywords: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reflectivity: Option<AppearanceReflectivity>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub transparency: Option<AppearanceTransparency>,
}

/// Generic appearance reflection strengths as percentages.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default)]
pub struct AppearanceReflectivity {
    pub enabled: bool,
    pub direct: u8,
    pub oblique: u8,
}

impl Default for AppearanceReflectivity {
    fn default() -> Self {
        Self {
            enabled: false,
            direct: 50,
            oblique: 50,
        }
    }
}

/// A Generic appearance's transmission settings, retained when disabled.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(default, rename_all = "camelCase")]
pub struct AppearanceTransparency {
    pub enabled: bool,
    pub amount: u8,
    pub translucency: u8,
    pub refraction: RefractionPreset,
    pub custom_index_milli: u16,
}

impl Default for AppearanceTransparency {
    fn default() -> Self {
        Self {
            enabled: false,
            amount: 30,
            translucency: 0,
            refraction: RefractionPreset::Glass,
            custom_index_milli: 1520,
        }
    }
}

impl AppearanceTransparency {
    /// Read the selected index; example: `effect.index_of_refraction()`.
    pub fn index_of_refraction(self) -> f32 {
        let milli = match self.refraction {
            RefractionPreset::Air => 1000,
            RefractionPreset::Water => 1330,
            RefractionPreset::Alcohol => 1360,
            RefractionPreset::Quartz => 1460,
            RefractionPreset::Glass => 1520,
            RefractionPreset::Diamond => 2300,
            RefractionPreset::Custom => self.custom_index_milli,
        };
        f32::from(milli) / 1000.
    }
}

/// The Generic Appearance Editor's refraction presets.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Deserialize, serde::Serialize)]
#[serde(rename_all = "camelCase")]
pub enum RefractionPreset {
    Air,
    Water,
    Alcohol,
    Quartz,
    Glass,
    Diamond,
    Custom,
}

/// A portable document owns asset snapshots and assignment intent together.
#[derive(Clone, Debug, Default)]
pub struct DocumentAssets {
    materials: AssetCollection<MaterialAsset>,
    appearances: AssetCollection<AppearanceAsset>,
    appearance_metadata: std::collections::BTreeMap<String, AppearanceMetadata>,
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
    /// Read the authored information; example: `assets.appearance_metadata("custom")`.
    pub fn appearance_metadata(&self, id: &str) -> Option<&AppearanceMetadata> {
        self.appearance_metadata.get(id)
    }

    /// Iterate persisted information; example: `assets.appearance_metadata_entries().count()`.
    pub fn appearance_metadata_entries(&self) -> impl Iterator<Item = (&str, &AppearanceMetadata)> {
        self.appearance_metadata
            .iter()
            .map(|(id, metadata)| (id.as_str(), metadata))
    }

    /// Set or clear a document appearance's information; example: `assets.set_appearance_metadata(id, metadata)?`.
    pub fn set_appearance_metadata(
        &mut self,
        id: &str,
        metadata: AppearanceMetadata,
    ) -> Result<(), String> {
        if self.appearances.get(id).is_none() {
            return Err(format!(
                "appearance {id:?}: expected a document-owned definition for its information"
            ));
        }
        if metadata.description.contains('\0') || metadata.keywords.contains('\0') {
            return Err(format!(
                "appearance {id:?}: expected description and keywords without NUL characters"
            ));
        }
        if let Some(reflectivity) = metadata.reflectivity
            && (reflectivity.direct > 100 || reflectivity.oblique > 100)
        {
            return Err(format!(
                "appearance {id:?} reflectivity {reflectivity:?}: expected percentages from 0 to 100"
            ));
        }
        if let Some(transparency) = metadata.transparency
            && (transparency.amount > 100
                || transparency.translucency > 100
                || !(1..=5000).contains(&transparency.custom_index_milli))
        {
            return Err(format!(
                "appearance {id:?} transparency {transparency:?}: expected percentages from 0 to 100 and a custom index from 0.001 to 5.0"
            ));
        }
        if metadata == AppearanceMetadata::default() {
            self.appearance_metadata.remove(id);
        } else {
            self.appearance_metadata.insert(id.into(), metadata);
        }
        Ok(())
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
            && self.appearance_metadata == other.appearance_metadata
            && self.assignments == other.assignments
    }
}
