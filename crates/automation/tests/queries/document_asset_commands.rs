// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn material_update_uses_document_history_without_overwriting_project() {
    let (mut files, _) = observed_catalog_files(false);
    let mut source = QueryGuids(0);
    let id = create_document_catalog(&mut files, &mut source, "Original");
    files.save(id, "test", &mut source).unwrap();
    let original = material_record(&files);
    let mut edited = original.clone();
    edited.source = "project".into();
    edited.display_name = "Document edit".into();
    edited.density = 12.0;
    let response = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .materials()
        .update(&edited)
        .unwrap();
    assert_eq!(response.source, "document");
    assert!(files.document_info(id).unwrap().dirty);
    files.commands(id).unwrap().undo().unwrap();
    assert_eq!(material_record(&files), original);
    assert!(!files.document_info(id).unwrap().dirty);
    files.commands(id).unwrap().redo().unwrap();
    assert_eq!(material_record(&files), response);
    reopen_and_check(&mut files, &mut source, id, response);
}
fn material_record(files: &DiscoveryWorkspace) -> MaterialInfo {
    Client::new(FileQueryHost::new(files))
        .materials()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap()
}
fn reopen_and_check(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    id: u64,
    expected: MaterialInfo,
) {
    let path = files.path(id).unwrap().to_owned();
    files.save(id, "test", source).unwrap();
    files.close_discarding_changes(id).unwrap();
    let _opened = files
        .open(
            &path,
            QueryObserver::default(),
            "test",
            source,
            OpenActivation::Activate,
        )
        .unwrap();
    assert_eq!(material_record(files), expected);
}
#[test]
fn document_appearance_updates_obey_transaction_abort() {
    let (mut files, _) = observed_catalog_files(false);
    let mut source = QueryGuids(0);
    let id = create_document_catalog(&mut files, &mut source, "Document");
    let before = files
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .assets
        .clone();
    files.commands(id).unwrap().begin("Appearance").unwrap();
    let request = UpdateAppearanceArgs {
        id: "portable-look".into(),
        display_name: "Updated".into(),
        ..Default::default()
    };
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .appearances()
        .update(&request)
        .unwrap();
    assert_ne!(
        files.workspace().session(id).unwrap().snapshot().assets,
        before
    );
    files.commands(id).unwrap().abort().unwrap();
    assert_eq!(
        files.workspace().session(id).unwrap().snapshot().assets,
        before
    );
}
#[test]
fn project_creation_from_document_copies_the_visible_snapshot() {
    let (mut files, _) = observed_catalog_files(false);
    let project = super::super::project_assets::project_files(&files, "copies", "Project");
    files.load_project(&project).unwrap();
    let mut source = QueryGuids(0);
    create_document_catalog(&mut files, &mut source, "Document");
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let original = client
        .materials()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap();
    let copy = client
        .materials()
        .create(&DuplicateAssetArgs {
            base_id: "steel".into(),
            name: "Steel".into(),
        })
        .unwrap();
    assert_eq!(copy.id, "steel-2");
    assert_eq!(copy.density, original.density);
    assert_eq!(copy.source, "project");
    assert_eq!(
        client
            .materials()
            .get(&AssetRefArgs { id: "steel".into() })
            .unwrap(),
        original
    );
}
