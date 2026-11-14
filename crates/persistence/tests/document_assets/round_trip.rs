// SPDX-License-Identifier: GPL-2.0-only
use super::fixture::*;
use oblikovati_api::{
    contract::{Appearance, Material},
    types::ASSET_DOCUMENT,
};
use oblikovati_model::assets::{AssetCollection, ScopedAssets};
use oblikovati_persistence::package::DocumentPackage;
#[test]
fn owned_definition_groups_and_assignment_intent_survive_without_the_project() {
    let assets = owned_assets();
    let saved = empty_package().with_assets(&assets).unwrap();
    let restored = DocumentPackage::decode(&saved.encode().unwrap())
        .unwrap()
        .restore_assets()
        .unwrap();
    assert_eq!(restored, assets);
    assert_eq!(
        restored.materials().get("custom-steel").unwrap().source(),
        ASSET_DOCUMENT
    );
    assert_eq!(
        restored.appearances().get("custom-look").unwrap().source(),
        ASSET_DOCUMENT
    );
    let empty_physics = AssetCollection::default();
    let empty_visuals = AssetCollection::default();
    let materials = ScopedAssets {
        document: restored.materials(),
        project: &empty_physics,
        builtin: &empty_physics,
    };
    let visuals = ScopedAssets {
        document: restored.appearances(),
        project: &empty_visuals,
        builtin: &empty_visuals,
    };
    assert_eq!(
        restored
            .assignments
            .resolve_appearance(&visuals, &materials, &target("body", ""), None)
            .unwrap()
            .id(),
        "custom-look"
    );
}
#[test]
fn each_assignment_family_and_occurrence_path_round_trips_deterministically() {
    let mut assets = owned_assets();
    let left = target("body", "left");
    let right = target("body", "right");
    assets
        .assignments
        .set_body_material(&left, "custom-steel")
        .unwrap();
    assets
        .assignments
        .set_body_appearance(&right, "custom-look")
        .unwrap();
    assets
        .assignments
        .set_face_appearance(&target("face", "left"), "custom-look")
        .unwrap();
    let saved = empty_package().with_assets(&assets).unwrap();
    assert_eq!(saved.manifest().schema_version, 3);
    let restored = DocumentPackage::decode(&saved.encode().unwrap())
        .unwrap()
        .restore_assets()
        .unwrap();
    assert_eq!(restored, assets);
    assert_eq!(
        restored.assignments.effective_material_id(&right),
        Some("custom-steel")
    );
    assert_eq!(
        saved.encode().unwrap(),
        empty_package()
            .with_assets(&assets)
            .unwrap()
            .encode()
            .unwrap()
    );
}
#[test]
fn legacy_flat_part_references_restore_without_an_occurrence_path() {
    let package = package_with_model(
        "  materials:\n    assignments:\n      partMaterial: steel\n      bodyMaterial: {bodykey: aluminum-6061}\n      bodyAppearance: {bodykey: oak}\n      faceAppearance: {facekey: abs-black}",
    );
    let restored = package.restore_assets().unwrap();
    assert_eq!(
        restored
            .assignments
            .effective_material_id(&target("bodykey", "")),
        Some("aluminum-6061")
    );
    assert_eq!(
        restored
            .assignments
            .face_appearances()
            .next()
            .unwrap()
            .0
            .occurrence_path()
            .len(),
        0
    );
    assert_eq!(package.with_assets(&restored).unwrap(), package);
}
#[test]
fn absence_null_and_cleared_assets_do_not_create_an_authored_section() {
    let original = empty_package();
    assert!(original.restore_assets().unwrap().is_empty());
    assert_eq!(original.with_assets(&Default::default()).unwrap(), original);
    let null =
        package_with_model("  materials: {materials: null, appearances: null, assignments: null}");
    assert!(null.restore_assets().unwrap().is_empty());
    let cleared = original
        .with_assets(&owned_assets())
        .unwrap()
        .with_assets(&Default::default())
        .unwrap();
    assert!(raw_package(&cleared)["model"].get("materials").is_none());
    let nullable =
        package_with_model("  materials: {assignments: {partMaterial: null, bodyMaterial: null}}");
    assert!(nullable.restore_assets().unwrap().is_empty());
}
