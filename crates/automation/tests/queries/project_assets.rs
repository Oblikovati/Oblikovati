// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_persistence::project_file::ProjectConfiguration;
#[path = "project_assets_reload.rs"]
mod reload;
#[path = "project_assets_write.rs"]
mod write;
pub(super) fn project_files(
    files: &DiscoveryWorkspace,
    name: &str,
    steel_name: &str,
) -> std::path::PathBuf {
    let root = std::path::PathBuf::from(format!("/fixtures/{name}"));
    let project = root.join("design.opj");
    let configuration = ProjectConfiguration {
        name: name.into(),
        design_data: "assets".into(),
        ..Default::default()
    };
    let mut storage = files.files().packages.borrow_mut();
    storage.insert(project.clone(), configuration.encode().unwrap());
    storage.insert(root.join("assets/materials.yaml"),format!("materials: [{{id: steel, name: {steel_name}, density: 7.87, appearance: steel}},{{id: project-only, name: Project only, density: 2.7}}]").into_bytes());
    storage.insert(
        root.join("assets/appearances.yaml"),
        b"appearances: [{id: project-look, name: Project look, albedo: '#804020', opacity: 1}]"
            .to_vec(),
    );
    project
}
fn material_name(files: &DiscoveryWorkspace, id: &str) -> String {
    Client::new(FileQueryHost::new(files))
        .materials()
        .get(&AssetRefArgs { id: id.into() })
        .unwrap()
        .display_name
}
#[test]
fn selecting_loaded_projects_changes_only_the_project_definition_scope() {
    let (mut files, _) = observed_catalog_files(false);
    let first = project_files(&files, "one", "First");
    let second = project_files(&files, "two", "Second");
    files.load_project(&first).unwrap();
    files.load_project(&second).unwrap();
    assert_eq!(material_name(&files, "steel"), "First");
    files.select_project(&second).unwrap();
    assert_eq!(material_name(&files, "steel"), "Second");
    let client = Client::new(FileQueryHost::new(&files));
    assert_eq!(
        client.materials().list().unwrap().materials.unwrap().len(),
        127
    );
    assert_eq!(
        client
            .appearances()
            .get(&AssetRefArgs {
                id: "project-look".into()
            })
            .unwrap()
            .source,
        "project"
    );
    assert_eq!(material_name(&files, "project-only"), "Project only");
    assert!(files.workspace().documents().is_empty());
}
#[test]
fn failed_second_library_load_keeps_the_previous_project_and_its_catalog() {
    let (mut files, _) = observed_catalog_files(false);
    let first = project_files(&files, "one", "First");
    files.load_project(&first).unwrap();
    let bad = project_files(&files, "bad", "Bad");
    files.files().packages.borrow_mut().insert(
        Path::new("/fixtures/bad/assets/materials.yaml").into(),
        b"materials: [".to_vec(),
    );
    let refusal = files.load_project(&bad).unwrap_err();
    assert!(refusal.to_string().contains("materials.yaml"));
    assert!(std::error::Error::source(&refusal).is_none());
    assert_eq!(files.projects().len(), 1);
    assert_eq!(files.active_project().unwrap().path, first);
    assert_eq!(material_name(&files, "steel"), "First");
}
#[test]
fn document_definition_stays_authoritative_over_selected_project_changes() {
    let (mut files, _) = observed_catalog_files(false);
    let first = project_files(&files, "one", "First");
    files.load_project(&first).unwrap();
    let mut source = QueryGuids(0);
    let project_material = files
        .active_project()
        .unwrap()
        .assets
        .materials()
        .get("steel")
        .unwrap()
        .clone();
    let mut assets = oblikovati_model::assets::DocumentAssets::default();
    assets.embed_material(&project_material).unwrap();
    let package =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {parameters: []}\n")
            .unwrap()
            .with_assets(&assets)
            .unwrap();
    let id = files
        .create(
            Path::new("/fixtures/portable.opd"),
            package,
            QueryObserver::default(),
            "test",
            &mut source,
        )
        .unwrap()
        .session_id;
    let before_reload = files.workspace().session(id).unwrap().snapshot().clone();
    let dirty_before_reload = files.workspace().needs_save(id).unwrap();
    project_files(&files, "one", "Reloaded");
    files.reload_project_assets(&first).unwrap();
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .same_metadata_recipe(&before_reload)
    );
    assert_eq!(
        files.workspace().needs_save(id).unwrap(),
        dirty_before_reload
    );
    assert_eq!(material_name(&files, "steel"), "First");
    let second = project_files(&files, "two", "Second");
    files.load_project(&second).unwrap();
    files.select_project(&second).unwrap();
    assert_eq!(material_name(&files, "steel"), "First");
    files.close_discarding_changes(id).unwrap();
    assert_eq!(material_name(&files, "steel"), "Second");
}

#[path = "asset_assignment_history.rs"]
mod assignment_history;
