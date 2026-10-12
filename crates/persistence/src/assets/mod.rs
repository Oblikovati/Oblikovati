// SPDX-License-Identifier: GPL-2.0-only
//! Catalog decoding stays in storage; admitted typed assets and their rules stay in the model.
mod decode;
pub(crate) use decode::stored_asset_record;
mod embedded;
mod legacy;
use oblikovati_model::assets::{AppearanceAsset, BuiltinAssetCatalog, MaterialAsset};
use serde::Deserialize;
use serde_yaml_ng::Value;
/// Named, injected bytes with no filesystem or process-directory dependency.
#[derive(Clone, Copy)]
pub struct CatalogSource<'a> {
    pub name: &'a str,
    pub text: &'a str,
}
#[derive(Default, Deserialize)]
#[serde(default, deny_unknown_fields)]
struct CatalogDocument {
    appearances: Vec<Value>,
    materials: Vec<Value>,
}
/// Decode the embedded source catalog, refusing malformed assets; example: `load_builtin_catalog()?`.
pub fn load_builtin_catalog() -> Result<BuiltinAssetCatalog, String> {
    decode_catalog_sources(embedded::SOURCES)
}
/// Decode a complete injected catalog in caller order; example: `decode_catalog_sources(&sources)?`.
pub fn decode_catalog_sources(
    sources: &[CatalogSource<'_>],
) -> Result<BuiltinAssetCatalog, String> {
    let mut materials = Vec::new();
    let mut appearances = Vec::new();
    for source in sources {
        append_catalog_source(*source, &mut materials, &mut appearances)?;
    }
    BuiltinAssetCatalog::from_records(materials, appearances)
}
fn append_catalog_source(
    source: CatalogSource<'_>,
    materials: &mut Vec<MaterialAsset>,
    appearances: &mut Vec<AppearanceAsset>,
) -> Result<(), String> {
    let parsed = read_catalog_document(source)?;
    for record in parsed.appearances {
        appearances.push(catalog_appearance(source.name, record)?);
    }
    for record in parsed.materials {
        materials.push(catalog_material(source.name, record)?);
    }
    Ok(())
}
fn read_catalog_document(source: CatalogSource<'_>) -> Result<CatalogDocument, String> {
    serde_yaml_ng::from_str(source.text).map_err(|cause| {
        format!(
            "catalog {:?}: expected an appearances/materials YAML document: {cause}",
            source.name
        )
    })
}
fn catalog_appearance(name: &str, record: Value) -> Result<AppearanceAsset, String> {
    let info = decode::api_catalog_record(name, record, &[("name", "displayName")])?;
    AppearanceAsset::from_info(info).map_err(|cause| format!("catalog {name:?}: {cause}"))
}
fn catalog_material(name: &str, record: Value) -> Result<MaterialAsset, String> {
    let info = decode::api_catalog_record(
        name,
        record,
        &[
            ("name", "displayName"),
            ("isotropy", "isotropyClass"),
            ("orthotropic", "anisotropic"),
            ("appearance", "appearanceId"),
        ],
    )?;
    MaterialAsset::from_info(info).map_err(|cause| format!("catalog {name:?}: {cause}"))
}

pub(crate) fn stored_appearance_record(
    name: &str,
    record: Value,
    source: oblikovati_api::types::AssetSource,
) -> Result<oblikovati_api::wire::AppearanceInfo, String> {
    if ["albedo", "metallic", "roughness", "emissive", "opacity"]
        .iter()
        .any(|field| record.get(*field).is_some())
    {
        return legacy::decode_legacy_appearance(name, record, source);
    }
    stored_asset_record(name, record, &[("name", "displayName")], source)
}
