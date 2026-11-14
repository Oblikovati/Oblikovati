// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assets::{DocumentAssets, MaterialAssignmentTarget};

fn assignment_workspace() -> (DiscoveryWorkspace, QueryGuids, u64) {
    let (mut files, _) = observed_catalog_files(false);
    let project = project_files(&files, "assignment", "Portable steel");
    files.files().packages.borrow_mut().insert(
        project.parent().unwrap().join("assets/materials.yaml"),
        b"materials: [{id: custom, name: Custom, density: 3.5, appearance: project-look}]".to_vec(),
    );
    files.load_project(&project).unwrap();
    let mut source = QueryGuids(0);
    let package = DocumentPackage::empty(DOCUMENT_PART, "Assigned");
    let id = files
        .create(
            Path::new("/fixtures/assigned.opd"),
            package,
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    files.save(id, "test", &mut source).unwrap();
    (files, source, id)
}
fn assign_project_material(
    files: &mut DiscoveryWorkspace,
    id: u64,
    material: &str,
) -> Result<(), String> {
    let project = files.active_project().unwrap().assets.clone();
    let catalog = files.builtin_catalog().unwrap().clone();
    files
        .commands(id)
        .unwrap()
        .run("Assign material", |snapshot, _| {
            snapshot.assets = snapshot.assets.with_material_assignment(
                &project,
                &catalog,
                &MaterialAssignmentTarget::Part,
                material,
            )?;
            Ok::<(), String>(())
        })
        .map_err(|cause| cause.to_string())
}
fn owned_assets(files: &DiscoveryWorkspace, id: u64) -> DocumentAssets {
    files
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .assets
        .clone()
}
#[test]
fn assignment_is_one_undo_step_and_restores_without_the_project() {
    let (mut files, mut source, id) = assignment_workspace();
    assign_project_material(&mut files, id, "custom").unwrap();
    let assigned = owned_assets(&files, id);
    assert_eq!(assigned.assignments.part_material_id(), Some("custom"));
    assert!(assigned.appearances().get("project-look").is_some());
    files.commands(id).unwrap().undo().unwrap();
    assert!(owned_assets(&files, id).is_empty());
    assert!(!files.document_info(id).unwrap().dirty);
    files.commands(id).unwrap().redo().unwrap();
    assert_eq!(owned_assets(&files, id), assigned);
    files.save(id, "test", &mut source).unwrap();
    reopen_without_project(&files, &assigned);
}
fn reopen_without_project(files: &DiscoveryWorkspace, expected: &DocumentAssets) {
    let path = Path::new("/fixtures/assigned.opd");
    let bytes = files.files().packages.borrow()[path].clone();
    let (mut reopened, _) = observed_catalog_files(false);
    reopened
        .files()
        .packages
        .borrow_mut()
        .insert(path.into(), bytes);
    let mut source = QueryGuids(100);
    let result = reopened
        .open(
            path,
            QueryObserver::default(),
            "test",
            &mut source,
            OpenActivation::Activate,
        )
        .unwrap();
    assert!(reopened.active_project().is_none());
    assert_eq!(&owned_assets(&reopened, result.session_id()), expected);
    let client = Client::new(FileQueryHost::new(&reopened));
    assert_eq!(
        client
            .materials()
            .get(&AssetRefArgs {
                id: "custom".into()
            })
            .unwrap()
            .source,
        "document"
    );
    assert_eq!(
        client
            .appearances()
            .get(&AssetRefArgs {
                id: "project-look".into()
            })
            .unwrap()
            .source,
        "document"
    );
}
#[test]
fn failed_assignment_and_aborted_batch_leave_saved_document_unchanged() {
    let (mut files, _, id) = assignment_workspace();
    assert!(assign_project_material(&mut files, id, "missing").is_err());
    assert!(owned_assets(&files, id).is_empty());
    assert!(!files.document_info(id).unwrap().dirty);
    files
        .commands(id)
        .unwrap()
        .begin("Assign and cancel")
        .unwrap();
    assign_project_material(&mut files, id, "custom").unwrap();
    assert!(!owned_assets(&files, id).is_empty());
    files.commands(id).unwrap().abort().unwrap();
    assert!(owned_assets(&files, id).is_empty());
    assert!(!files.document_info(id).unwrap().dirty);
}

#[test]
fn typed_part_assignment_uses_portable_owner_history_and_refuses_unverified_entities() {
    use oblikovati_automation::FileHost;
    let (mut files, mut source, id) = assignment_workspace();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .materials()
        .assign(&AssignMaterialArgs {
            material_id: "custom".into(),
            ..Default::default()
        })
        .unwrap();
    let assigned = owned_assets(&files, id);
    assert_eq!(assigned.assignments.part_material_id(), Some("custom"));
    assert!(assigned.appearances().get("project-look").is_some());
    files.commands(id).unwrap().undo().unwrap();
    assert!(owned_assets(&files, id).is_empty());
    files.commands(id).unwrap().redo().unwrap();
    files.save(id, "test", &mut source).unwrap();
    reopen_without_project(&files, &assigned);
    for key in ["bad", "éé", "00", "00000000000000000400000000000000"] {
        let error = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .materials()
            .assign(&AssignMaterialArgs {
                body_key: key.into(),
                material_id: "custom".into(),
            })
            .unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert_eq!(owned_assets(&files, id), assigned);
    }
}
#[test]
fn typed_part_appearance_rejects_bad_scope_and_rolls_back_in_an_aborted_batch() {
    use oblikovati_automation::FileHost;
    let (mut files, mut source, id) = assignment_workspace();
    for (scope, key, appearance) in [
        ("unknown", "", "project-look"),
        ("part", "unexpected", "project-look"),
        ("face", "", "project-look"),
        ("part", "", "missing"),
        ("part", "", ""),
    ] {
        assert!(
            Client::new(FileHost::new(&mut files, &mut source, "test"))
                .appearances()
                .assign(&AssignAppearanceArgs {
                    scope: scope.into(),
                    key: key.into(),
                    appearance_id: appearance.into()
                })
                .is_err()
        );
        assert!(owned_assets(&files, id).is_empty());
    }
    files
        .commands(id)
        .unwrap()
        .begin("Cancel appearance")
        .unwrap();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "part".into(),
            appearance_id: "project-look".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(
        owned_assets(&files, id).assignments.part_appearance_id(),
        Some("project-look")
    );
    files.commands(id).unwrap().abort().unwrap();
    assert!(owned_assets(&files, id).is_empty());
}
