// SPDX-License-Identifier: GPL-2.0-only
use super::fixture::*;
use oblikovati_model::assets::AppearanceAsset;
#[test]
fn known_edits_and_clears_preserve_opaque_extension_fields() {
    let assets = owned_assets();
    let mut raw = raw_package(&empty_package().with_assets(&assets).unwrap());
    raw["model"]["materials"]["future"] = "retained".into();
    raw["model"]["materials"]["appearances"][0]["geometry"]["futureNormal"] = "retained".into();
    let original = yaml_package(&raw);
    let mut edited = original.restore_assets().unwrap();
    let mut look = edited.appearances().get("custom-look").unwrap().info();
    look.geometry.normal = Some(std::boxed::Box::new(oblikovati_api::types::Vector {
        x: 0.,
        y: 0.,
        z: 1.,
    }));
    edited
        .embed_appearance(&AppearanceAsset::from_info(look).unwrap())
        .unwrap();
    let changed = original.with_assets(&edited).unwrap();
    assert_eq!(
        raw_package(&changed)["model"]["materials"]["appearances"][0]["geometry"]["futureNormal"],
        "retained"
    );
    let cleared = changed.with_assets(&Default::default()).unwrap();
    assert_eq!(
        raw_package(&cleared)["model"]["materials"]["future"],
        "retained"
    );
    assert!(cleared.restore_assets().unwrap().is_empty());
}
#[test]
fn omitted_optional_values_are_removed_without_reviving_old_vectors() {
    let mut assets = owned_assets();
    let mut info = assets.appearances().get("custom-look").unwrap().info();
    info.geometry.normal = Some(std::boxed::Box::new(oblikovati_api::types::Vector {
        x: 0.,
        y: 0.,
        z: 1.,
    }));
    assets
        .embed_appearance(&AppearanceAsset::from_info(info).unwrap())
        .unwrap();
    let original = empty_package().with_assets(&assets).unwrap();
    let mut cleared = assets;
    let mut info = cleared.appearances().get("custom-look").unwrap().info();
    info.geometry.normal = None;
    cleared
        .embed_appearance(&AppearanceAsset::from_info(info).unwrap())
        .unwrap();
    let saved = original.with_assets(&cleared).unwrap();
    assert_eq!(saved.restore_assets().unwrap(), cleared);
}
