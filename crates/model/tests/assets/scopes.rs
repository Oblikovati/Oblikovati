// SPDX-License-Identifier: GPL-2.0-only
use super::support::{appearance_record, material_record};
use oblikovati_api::{
    contract::{Appearance, Material},
    types::{ASSET_BUILTIN, ASSET_DOCUMENT, ASSET_PROJECT},
};
use oblikovati_model::assets::{AppearanceAsset, AssetCollection, MaterialAsset, ScopedAssets};
#[test]
fn physical_scopes_prefer_owned_document_snapshots_to_changed_project_values() {
    let [builtin, mut project, document] = prepared_physical_scopes();
    let mut changed = project.get("physical").unwrap().clone();
    let mut edit = changed.info();
    edit.density = 1.2;
    changed.edit(edit).unwrap();
    project.insert(changed);
    let scopes = ScopedAssets {
        document: &document,
        project: &project,
        builtin: &builtin,
    };
    assert_eq!(scopes.resolve("physical").unwrap().density(), 7.85);
    assert_eq!(project.get("physical").unwrap().density(), 1.2);
    assert!(scopes.resolve("missing").unwrap_err().contains("missing"));
}
#[test]
fn visual_scopes_follow_document_project_builtin_precedence() {
    let mut builtin = AssetCollection::default();
    let mut project = AssetCollection::default();
    let mut document = AssetCollection::default();
    let base = AppearanceAsset::from_info(appearance_record(ASSET_BUILTIN)).unwrap();
    builtin.insert(base.clone());
    assert_eq!(
        resolved_visual_origin(&document, &project, &builtin),
        ASSET_BUILTIN
    );
    project.insert(base.duplicate("visual", "Project", ASSET_PROJECT).unwrap());
    assert_eq!(
        resolved_visual_origin(&document, &project, &builtin),
        ASSET_PROJECT
    );
    document.insert(
        base.duplicate("visual", "Document", ASSET_DOCUMENT)
            .unwrap(),
    );
    assert_eq!(
        resolved_visual_origin(&document, &project, &builtin),
        ASSET_DOCUMENT
    );
}
#[test]
fn scope_iteration_and_replacement_are_deterministic() {
    let base = AppearanceAsset::from_info(appearance_record(ASSET_BUILTIN)).unwrap();
    let mut collection = AssetCollection::default();
    for id in ["z", "a", "m"] {
        collection.insert(base.duplicate(id, id, ASSET_PROJECT).unwrap());
    }
    assert_eq!(
        collection.ordered().map(Appearance::id).collect::<Vec<_>>(),
        ["a", "m", "z"]
    );
    let old = collection
        .insert(base.duplicate("m", "Replacement", ASSET_DOCUMENT).unwrap())
        .unwrap();
    assert_eq!(
        collection
            .insertion_order()
            .map(Appearance::id)
            .collect::<Vec<_>>(),
        ["z", "a", "m"]
    );
    assert_eq!(old.display_name(), "m");
    assert_eq!(collection.get("m").unwrap().display_name(), "Replacement");
}

fn resolved_visual_origin(
    document: &AssetCollection<AppearanceAsset>,
    project: &AssetCollection<AppearanceAsset>,
    builtin: &AssetCollection<AppearanceAsset>,
) -> oblikovati_api::types::AssetSource {
    ScopedAssets {
        document,
        project,
        builtin,
    }
    .resolve("visual")
    .unwrap()
    .source()
}
#[test]
fn reloaded_document_snapshot_resolves_without_the_original_project() {
    let original = MaterialAsset::from_info(material_record(ASSET_PROJECT)).unwrap();
    let bytes = serde_json::to_vec(&original.info()).unwrap();
    drop(original);
    let reloaded = MaterialAsset::from_info(serde_json::from_slice(&bytes).unwrap()).unwrap();
    let mut document = AssetCollection::default();
    document.insert(reloaded);
    let absent_project = AssetCollection::default();
    let absent_builtins = AssetCollection::default();
    let scopes = ScopedAssets {
        document: &document,
        project: &absent_project,
        builtin: &absent_builtins,
    };
    assert_eq!(scopes.resolve("physical").unwrap().density(), 7.85);
}

fn prepared_physical_scopes() -> [AssetCollection<MaterialAsset>; 3] {
    let mut builtin = AssetCollection::default();
    let mut project = AssetCollection::default();
    let mut document = AssetCollection::default();
    let base = MaterialAsset::from_info(material_record(ASSET_BUILTIN)).unwrap();
    builtin.insert(base.clone());
    let shared = base
        .duplicate("physical", "Project", ASSET_PROJECT)
        .unwrap();
    project.insert(shared.clone());
    document.insert(shared);
    [builtin, project, document]
}
