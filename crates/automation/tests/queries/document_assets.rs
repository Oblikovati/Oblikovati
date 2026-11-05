// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::ASSET_PROJECT;
use oblikovati_model::assets::DocumentAssets;
fn document_catalog(steel_name: &str) -> DocumentAssets {
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let mut assets = DocumentAssets::default();
    assets
        .embed_material(
            &catalog
                .materials()
                .get("steel")
                .unwrap()
                .duplicate("steel", steel_name, ASSET_PROJECT)
                .unwrap(),
        )
        .unwrap();
    assets
        .embed_material(
            &catalog
                .materials()
                .get("steel")
                .unwrap()
                .duplicate("portable", "Portable material", ASSET_PROJECT)
                .unwrap(),
        )
        .unwrap();
    assets
        .embed_appearance(
            &catalog
                .appearances()
                .get("steel")
                .unwrap()
                .duplicate("portable-look", "Portable look", ASSET_PROJECT)
                .unwrap(),
        )
        .unwrap();
    assets
}
fn create_document_catalog(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    name: &str,
) -> u64 {
    let package =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {parameters: []}\n")
            .unwrap()
            .with_assets(&document_catalog(name))
            .unwrap();
    files
        .create(
            Path::new(&format!("/fixtures/{name}.opd")),
            package,
            QueryObserver::default(),
            "test",
            source,
        )
        .unwrap()
        .session_id
}
fn steel_name(files: &DiscoveryWorkspace) -> String {
    Client::new(FileQueryHost::new(files))
        .materials()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap()
        .display_name
}
#[test]
fn typed_reads_use_the_active_document_without_overwriting_shared_catalog_definitions() {
    let (mut files, _) = observed_catalog_files(false);
    let mut source = QueryGuids(0);
    let first = create_document_catalog(&mut files, &mut source, "First");
    assert_eq!(steel_name(&files), "First");
    let second = create_document_catalog(&mut files, &mut source, "Second");
    assert_eq!(steel_name(&files), "Second");
    files.activate(first).unwrap();
    assert_eq!(steel_name(&files), "First");
    files.close_discarding_changes(first).unwrap();
    assert_eq!(steel_name(&files), "Second");
    files.close_discarding_changes(second).unwrap();
    assert_ne!(steel_name(&files), "First");
    assert_ne!(steel_name(&files), "Second");
    assert_eq!(
        Client::new(FileQueryHost::new(&files))
            .materials()
            .list()
            .unwrap()
            .materials
            .unwrap()
            .len(),
        126
    );
}
#[test]
fn document_lists_have_one_definition_per_id_and_keep_catalog_positions() {
    let (mut files, _) = observed_catalog_files(false);
    let mut source = QueryGuids(0);
    create_document_catalog(&mut files, &mut source, "Doc steel");
    let client = Client::new(FileQueryHost::new(&files));
    let materials = client.materials().list().unwrap().materials.unwrap();
    assert_eq!(materials.len(), 127);
    assert_eq!(
        materials.iter().filter(|asset| asset.id == "steel").count(),
        1
    );
    assert_eq!(
        materials
            .iter()
            .find(|asset| asset.id == "steel")
            .unwrap()
            .display_name,
        "Doc steel"
    );
    assert_eq!(materials.last().unwrap().id, "portable");
    assert_eq!(
        client
            .appearances()
            .list()
            .unwrap()
            .appearances
            .unwrap()
            .len(),
        95
    );
    assert_eq!(
        client
            .appearances()
            .get(&AssetRefArgs {
                id: "portable-look".into()
            })
            .unwrap()
            .source,
        "document"
    );
}
#[test]
fn saved_and_reopened_embedded_materials_remain_readable_through_the_file_host() {
    let (mut files, _) = observed_catalog_files(false);
    let mut source = QueryGuids(0);
    let id = create_document_catalog(&mut files, &mut source, "Portable");
    files.save(id, "test", &mut source).unwrap();
    files.close_discarding_changes(id).unwrap();
    let opened = files
        .open(
            Path::new("/fixtures/Portable.opd"),
            QueryObserver::default(),
            "test",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap();
    assert_eq!(
        files.workspace().active_document(),
        Some(opened.session_id())
    );
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    assert_eq!(
        client
            .materials()
            .get(&AssetRefArgs {
                id: "portable".into()
            })
            .unwrap()
            .display_name,
        "Portable material"
    );
    assert_eq!(
        client
            .appearances()
            .get(&AssetRefArgs {
                id: "portable-look".into()
            })
            .unwrap()
            .display_name,
        "Portable look"
    );
}

#[path = "document_asset_commands.rs"]
mod commands;
