// SPDX-License-Identifier: GPL-2.0-only
mod assignments;
mod merge;
mod records;
use super::{DocumentPackage, Mapping, PackageCodecError, Value};
pub(crate) use merge::merge_asset_entries;
use oblikovati_model::assets::DocumentAssets;
use records::AssetRecipe;
pub(crate) use records::stored_record;
pub(crate) use records::{decode_appearance_definitions, decode_material_definitions};
impl DocumentPackage {
    /// Restore portable definitions and exact assignment intent; example: `package.restore_assets()?`.
    pub fn restore_assets(&self) -> Result<DocumentAssets, PackageCodecError> {
        let section = asset_section(self);
        let recipe: AssetRecipe = serde_yaml_ng::from_value(section).map_err(asset_error)?;
        recipe
            .restore(self.manifest.schema_version)
            .map_err(asset_error)
    }
    /// Capture asset intent without dropping other recipe content; example: `package.with_assets(&snapshot.assets)?`.
    pub fn with_assets(&self, assets: &DocumentAssets) -> Result<Self, PackageCodecError> {
        let before = self.restore_assets()?;
        if &before == assets && !contains_legacy_appearances(self) {
            return Ok(self.clone());
        }
        let recipe = AssetRecipe::capture(assets).map_err(asset_error)?;
        let mut candidate = self.updated_asset_recipe(&before, assets, &recipe)?;
        if recipe
            .assignments
            .as_ref()
            .is_some_and(|assignments| assignments.has_occurrences())
        {
            candidate.manifest.schema_version = 3;
            candidate.root.insert("schemaVersion".into(), 3u32.into());
        }
        if candidate.restore_assets()? != *assets {
            return Err(asset_error(
                "asset export: expected an exact definition/assignment round trip",
            ));
        }
        Ok(candidate)
    }
    fn updated_asset_recipe(
        &self,
        before: &DocumentAssets,
        after: &DocumentAssets,
        recipe: &AssetRecipe,
    ) -> Result<Self, PackageCodecError> {
        let mut candidate = self.clone();
        let mut model = candidate
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        let section = updated_asset_section(self, before, after, recipe)?;
        if section.as_mapping().is_some_and(|fields| fields.is_empty()) {
            model.remove("materials");
        } else {
            model.insert("materials".into(), section);
        }
        candidate.root.insert("model".into(), Value::Mapping(model));
        Ok(candidate)
    }
}
fn asset_section(package: &DocumentPackage) -> Value {
    package
        .root
        .get("model")
        .and_then(|model| model.get("materials"))
        .filter(|section| !section.is_null())
        .cloned()
        .unwrap_or(Value::Mapping(Mapping::new()))
}
fn asset_error(cause: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!("document assets: {cause}"))
}

fn updated_asset_section(
    package: &DocumentPackage,
    before: &DocumentAssets,
    after: &DocumentAssets,
    recipe: &AssetRecipe,
) -> Result<Value, PackageCodecError> {
    let known = serde_yaml_ng::to_value(AssetRecipe::capture(before).map_err(asset_error)?)
        .map_err(asset_error)?;
    if after.is_empty() {
        return Ok(merge::merge_known_values(
            &asset_section(package),
            &known,
            &Value::Mapping(Mapping::new()),
        ));
    }
    let replacement = serde_yaml_ng::to_value(recipe).map_err(asset_error)?;
    Ok(merge::merge_asset_recipe(
        &asset_section(package),
        &known,
        &replacement,
    ))
}

fn contains_legacy_appearances(package: &DocumentPackage) -> bool {
    asset_section(package)
        .get("appearances")
        .and_then(Value::as_sequence)
        .is_some_and(|records| records.iter().any(|record| record.get("albedo").is_some()))
}
