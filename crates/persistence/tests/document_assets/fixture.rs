// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{ASSET_DOCUMENT, ASSET_PROJECT};
use oblikovati_model::assets::{AssignmentReference, DocumentAssets, MaterialAsset};
use oblikovati_persistence::{assets::load_builtin_catalog, package::DocumentPackage};
pub fn owned_assets() -> DocumentAssets {
    let catalog = load_builtin_catalog().unwrap();
    let material = catalog
        .materials()
        .get("steel")
        .unwrap()
        .duplicate("custom-steel", "Custom steel", ASSET_PROJECT)
        .unwrap();
    let appearance = catalog
        .appearances()
        .get("steel")
        .unwrap()
        .duplicate("custom-look", "Custom look", ASSET_DOCUMENT)
        .unwrap();
    let mut record = material.info();
    record.appearance_id = "custom-look".into();
    let mut assets = DocumentAssets::default();
    assets
        .embed_material(&MaterialAsset::from_info(record).unwrap())
        .unwrap();
    assets.embed_appearance(&appearance).unwrap();
    assets
        .assignments
        .set_part_material("custom-steel")
        .unwrap();
    assets
        .assignments
        .set_part_appearance("custom-look")
        .unwrap();
    assets
}
pub fn empty_package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {parameters: []}\n")
        .unwrap()
}
pub fn target(key: &str, occurrence: &str) -> AssignmentReference {
    AssignmentReference::new(
        key,
        if occurrence.is_empty() {
            vec![]
        } else {
            vec![occurrence.into()]
        },
    )
    .unwrap()
}
pub fn package_with_model(model: &str) -> DocumentPackage {
    DocumentPackage::decode(
        format!("schemaVersion: 2\ndocumentType: 1\nmodel:\n{model}\n").as_bytes(),
    )
    .unwrap()
}
pub fn yaml_package(value: &serde_yaml_ng::Value) -> DocumentPackage {
    DocumentPackage::decode(serde_yaml_ng::to_string(value).unwrap().as_bytes()).unwrap()
}
pub fn raw_package(package: &DocumentPackage) -> serde_yaml_ng::Value {
    serde_yaml_ng::from_slice(&package.encode().unwrap()).unwrap()
}
