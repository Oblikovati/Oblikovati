// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::FileHost;

#[test]
fn typed_creation_uses_stable_unique_ids_and_preserves_base_properties() {
    let (mut files, project) = editable_project();
    let mut source = QueryGuids(0);
    let host = FileHost::new(&mut files, &mut source, "test");
    let client = Client::new(host);
    let base = client
        .materials()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap();
    for id in ["custom-steel", "custom-steel-2"] {
        let copy = client
            .materials()
            .create(&DuplicateAssetArgs {
                base_id: base.id.clone(),
                name: "Custom Steel".into(),
            })
            .unwrap();
        assert_eq!(copy.id, id);
        assert_eq!(copy.source, "project");
        assert_eq!(copy.density, base.density);
        assert_eq!(copy.magnetic, base.magnetic);
    }
    files.reload_project_assets(&project).unwrap();
    assert_eq!(material_name(&files, "custom-steel-2"), "Custom Steel");
}

#[test]
fn typed_project_updates_replace_properties_and_keep_authoritative_source() {
    let (mut files, project) = editable_project();
    let mut source = QueryGuids(0);
    let mut request = edited_material(&files).info();
    request.source = "builtin".into();
    let response = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .materials()
        .update(&request)
        .unwrap();
    assert_eq!(response.source, "project");
    assert_eq!(response.density, 3.5);
    files.reload_project_assets(&project).unwrap();
    assert_eq!(material_name(&files, "steel"), "Edited physical");
}

#[test]
fn typed_visual_copy_and_update_round_trip_all_groups() {
    let (mut files, project) = editable_project();
    let mut source = QueryGuids(0);
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let copy = client
        .appearances()
        .create(&CreateAppearanceArgs {
            base_id: "project-look".into(),
            name: "Coated".into(),
        })
        .unwrap();
    let mut update = appearance_update(copy);
    update.coat.weight = 0.65;
    update.emission.luminance = 3.0;
    let changed = client.appearances().update(&update).unwrap();
    assert_eq!(changed.coat, update.coat);
    assert_eq!(changed.emission, update.emission);
    files.reload_project_assets(&project).unwrap();
    assert_eq!(
        files
            .active_project()
            .unwrap()
            .assets
            .appearances()
            .get("coated")
            .unwrap()
            .info(),
        changed
    );
}

fn appearance_update(record: AppearanceInfo) -> UpdateAppearanceArgs {
    UpdateAppearanceArgs {
        id: record.id,
        display_name: record.display_name,
        base: record.base,
        specular: record.specular,
        transmission: record.transmission,
        subsurface: record.subsurface,
        coat: record.coat,
        fuzz: record.fuzz,
        thin_film: record.thin_film,
        emission: record.emission,
        geometry: record.geometry,
    }
}

#[test]
fn builtins_are_readonly_and_do_not_require_a_project_or_write() {
    let (mut files, _) = observed_catalog_files(false);
    let mut source = QueryGuids(0);
    files.files().refuse_write.set(true);
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let original = client
        .materials()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap();
    let mut edit = original.clone();
    edit.density = -1.0;
    assert_eq!(client.materials().update(&edit).unwrap(), original);
    let original = client
        .appearances()
        .get(&AssetRefArgs { id: "steel".into() })
        .unwrap();
    assert_eq!(
        client
            .appearances()
            .update(&appearance_update(original.clone()))
            .unwrap(),
        original
    );
}

#[test]
fn failed_api_write_keeps_memory_disk_and_the_next_generated_identity() {
    let (mut files, project) = editable_project();
    let mut source = QueryGuids(0);
    let before = files.projects().to_vec();
    let bytes = library_bytes(&files, &project, "materials");
    files.files().refuse_write.set(true);
    let args = DuplicateAssetArgs {
        base_id: "steel".into(),
        name: "Retry".into(),
    };
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .materials()
            .create(&args)
            .is_err()
    );
    assert_eq!(files.projects(), before);
    assert_eq!(library_bytes(&files, &project, "materials"), bytes);
    files.files().refuse_write.set(false);
    assert_eq!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .materials()
            .create(&args)
            .unwrap()
            .id,
        "retry"
    );
}

#[test]
fn invalid_requests_and_missing_project_leave_catalog_unchanged() {
    let (mut files, _) = observed_catalog_files(false);
    let mut source = QueryGuids(0);
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    for (base, name) in [("absent", "Copy"), ("steel", ""), ("steel", "Copy")] {
        assert!(
            client
                .materials()
                .create(&DuplicateAssetArgs {
                    base_id: base.into(),
                    name: name.into()
                })
                .is_err()
        );
    }
    assert!(
        client
            .materials()
            .update(&MaterialInfo {
                id: "absent".into(),
                ..Default::default()
            })
            .is_err()
    );
    assert!(
        client
            .appearances()
            .update(&UpdateAppearanceArgs {
                id: "absent".into(),
                ..Default::default()
            })
            .is_err()
    );
    assert_eq!(
        client.materials().list().unwrap().materials.unwrap().len(),
        126
    );
}

#[test]
fn invalid_project_edit_and_malformed_wire_do_not_commit() {
    let (mut files, _) = editable_project();
    let previous = files.projects().to_vec();
    let mut source = QueryGuids(0);
    let mut invalid = edited_material(&files).info();
    invalid.density = -1.0;
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    assert!(client.materials().update(&invalid).is_err());
    let host = FileHost::new(&mut files, &mut source, "test");
    for method in oblikovati_automation::ASSET_COMMAND_METHODS {
        assert!(host.call(method, b"[").is_err());
    }
    assert_eq!(files.projects(), previous);
}
