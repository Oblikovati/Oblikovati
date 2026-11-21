// SPDX-License-Identifier: GPL-2.0-only
use super::assignments::AssignmentRecipe;
use oblikovati_api::{
    types::ASSET_DOCUMENT,
    wire::{AppearanceInfo, MaterialInfo},
};
use oblikovati_model::assets::{AppearanceAsset, DocumentAssets, MaterialAsset};
use serde::{Deserialize, Serialize};
use serde_yaml_ng::{Mapping, Value};
use std::collections::BTreeSet;
#[derive(Default, Serialize, Deserialize)]
#[serde(default)]
pub(super) struct AssetRecipe {
    #[serde(deserialize_with = "null_record")]
    pub appearances: Vec<Value>,
    #[serde(deserialize_with = "null_record")]
    pub materials: Vec<Value>,
    pub assignments: Option<AssignmentRecipe>,
}
impl AssetRecipe {
    pub fn restore(&self, schema: u32) -> Result<DocumentAssets, String> {
        let mut assets = DocumentAssets::default();
        for asset in decode_material_definitions(&self.materials, ASSET_DOCUMENT)? {
            assets.embed_material(&asset)?;
        }
        for asset in decode_appearance_definitions(&self.appearances, ASSET_DOCUMENT)? {
            assets.embed_appearance(&asset)?;
        }
        if let Some(assignments) = &self.assignments {
            assets.assignments = assignments.restore(schema)?;
        }
        Ok(assets)
    }
    pub fn capture(assets: &DocumentAssets) -> Result<Self, String> {
        Ok(Self {
            materials: assets
                .materials()
                .ordered()
                .map(|asset| stored_record(&asset.info(), true))
                .collect::<Result<_, _>>()?,
            appearances: assets
                .appearances()
                .ordered()
                .map(|asset| stored_record(&asset.info(), false))
                .collect::<Result<_, _>>()?,
            assignments: Some(AssignmentRecipe::capture(&assets.assignments)?),
        })
    }
}
pub(crate) fn decode_material_definitions(
    records: &[Value],
    source: oblikovati_api::types::AssetSource,
) -> Result<Vec<MaterialAsset>, String> {
    let mut decoded = Vec::new();
    let mut seen = BTreeSet::new();
    for record in records {
        let info: MaterialInfo = crate::assets::stored_asset_record(
            "document materials",
            record.clone(),
            &[
                ("name", "displayName"),
                ("appearance", "appearanceId"),
                ("isotropy", "isotropyClass"),
                ("orthotropic", "anisotropic"),
            ],
            source.clone(),
        )?;
        require_unique_asset(&mut seen, &info.id)?;
        decoded.push(MaterialAsset::from_info(info)?);
    }
    Ok(decoded)
}
pub(crate) fn decode_appearance_definitions(
    records: &[Value],
    source: oblikovati_api::types::AssetSource,
) -> Result<Vec<AppearanceAsset>, String> {
    let mut decoded = Vec::new();
    let mut seen = BTreeSet::new();
    for record in records {
        let info: AppearanceInfo = crate::assets::stored_appearance_record(
            "document appearances",
            record.clone(),
            source.clone(),
        )?;
        require_unique_asset(&mut seen, &info.id)?;
        decoded.push(AppearanceAsset::from_info(info)?);
    }
    Ok(decoded)
}
fn require_unique_asset(seen: &mut BTreeSet<String>, id: &str) -> Result<(), String> {
    if !seen.insert(id.into()) {
        return Err(format!(
            "embedded asset {id:?}: expected a unique identity within its asset family"
        ));
    }
    Ok(())
}
pub(crate) fn stored_record<T: Serialize>(record: &T, material: bool) -> Result<Value, String> {
    let value = serde_yaml_ng::to_value(record).map_err(|cause| cause.to_string())?;
    let mut fields = value
        .as_mapping()
        .cloned()
        .ok_or_else(|| format!("asset {value:?}: expected a typed record mapping"))?;
    fields.remove("source");
    rename_stored_field(&mut fields, "displayName", "name");
    if material {
        rename_material_fields(&mut fields);
    }
    Ok(Value::Mapping(fields))
}
fn rename_material_fields(fields: &mut Mapping) {
    for (api, stored) in [
        ("appearanceId", "appearance"),
        ("isotropyClass", "isotropy"),
        ("anisotropic", "orthotropic"),
    ] {
        rename_stored_field(fields, api, stored);
    }
}
fn rename_stored_field(fields: &mut Mapping, api: &str, stored: &str) {
    if let Some(value) = fields.remove(api) {
        fields.insert(stored.into(), value);
    }
}

pub(super) fn null_record<'de, T: Deserialize<'de> + Default, D: serde::Deserializer<'de>>(
    deserializer: D,
) -> Result<T, D::Error> {
    Option::<T>::deserialize(deserializer).map(Option::unwrap_or_default)
}
