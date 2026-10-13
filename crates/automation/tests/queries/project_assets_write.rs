// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assets::{AppearanceAsset, MaterialAsset};

fn editable_project() -> (DiscoveryWorkspace, std::path::PathBuf) {
    let (mut files, _) = observed_catalog_files(false);
    let project = project_files(&files, "editable", "Before");
    files.files().packages.borrow_mut().insert(
        Path::new("/fixtures/editable/assets/appearances.yaml").into(),
        b"appearances: [{id: project-look, name: Project look, albedo: '#804020', futureExtension: {tag: retained}}]".to_vec());
    files.load_project(&project).unwrap();
    (files, project)
}

fn edited_material(files: &DiscoveryWorkspace) -> MaterialAsset {
    let mut record = files
        .active_project()
        .unwrap()
        .assets
        .materials()
        .get("steel")
        .unwrap()
        .info();
    record.display_name = "Edited physical".into();
    record.density = 3.5;
    MaterialAsset::from_info(record).unwrap()
}

fn edited_appearance(files: &DiscoveryWorkspace) -> AppearanceAsset {
    let mut record = files
        .active_project()
        .unwrap()
        .assets
        .appearances()
        .get("project-look")
        .unwrap()
        .info();
    record.display_name = "Edited visual".into();
    record.coat.weight = 0.75;
    AppearanceAsset::from_info(record).unwrap()
}

fn library_bytes(files: &DiscoveryWorkspace, project: &Path, family: &str) -> Vec<u8> {
    let path = project
        .parent()
        .unwrap()
        .join("assets")
        .join(format!("{family}.yaml"));
    files.files().packages.borrow()[&path].clone()
}

#[test]
fn material_write_adopts_after_persistence_and_leaves_visual_file_unchanged() {
    let (mut files, project) = editable_project();
    let visual_bytes = library_bytes(&files, &project, "appearances");
    let material = edited_material(&files);
    files.store_project_material(material.clone()).unwrap();
    assert_eq!(material_name(&files, "steel"), "Edited physical");
    assert_eq!(library_bytes(&files, &project, "appearances"), visual_bytes);
    files.reload_project_assets(&project).unwrap();
    assert_eq!(
        files
            .active_project()
            .unwrap()
            .assets
            .materials()
            .get("steel"),
        Some(&material)
    );
}

#[test]
fn appearance_write_upgrades_legacy_groups_and_leaves_physical_file_unchanged() {
    let (mut files, project) = editable_project();
    let physical_bytes = library_bytes(&files, &project, "materials");
    let appearance = edited_appearance(&files);
    files.store_project_appearance(appearance.clone()).unwrap();
    assert_eq!(library_bytes(&files, &project, "materials"), physical_bytes);
    let yaml = String::from_utf8(library_bytes(&files, &project, "appearances")).unwrap();
    assert!(!yaml.contains("albedo:"));
    assert!(yaml.contains("tag: retained"));
    files.reload_project_assets(&project).unwrap();
    assert_eq!(
        files
            .active_project()
            .unwrap()
            .assets
            .appearances()
            .get("project-look"),
        Some(&appearance)
    );
}

#[test]
fn write_failure_preserves_registered_snapshot_and_bytes_then_can_retry() {
    let (mut files, project) = editable_project();
    let previous = files.projects().to_vec();
    let bytes = library_bytes(&files, &project, "materials");
    let material = edited_material(&files);
    files.files().refuse_write.set(true);
    assert!(files.store_project_material(material.clone()).is_err());
    assert_eq!(files.projects(), previous);
    assert_eq!(library_bytes(&files, &project, "materials"), bytes);
    files.files().refuse_write.set(false);
    files.store_project_material(material).unwrap();
    assert_eq!(material_name(&files, "steel"), "Edited physical");
}

#[test]
fn externally_changed_library_requires_reload_before_overwrite() {
    let (mut files, project) = editable_project();
    let candidate = edited_material(&files);
    project_files(&files, "editable", "External");
    let external = library_bytes(&files, &project, "materials");
    let refusal = files.store_project_material(candidate).unwrap_err();
    assert!(refusal.to_string().contains("reload"));
    assert_eq!(material_name(&files, "steel"), "Before");
    assert_eq!(library_bytes(&files, &project, "materials"), external);
    files.reload_project_assets(&project).unwrap();
    files
        .store_project_material(edited_material(&files))
        .unwrap();
}

#[test]
fn malformed_or_wrong_family_files_cannot_be_overwritten() {
    for invalid in [
        "materials: [",
        "materials: [{id: steel}, {id: steel}]",
        "appearances: [{id: misplaced}]",
    ] {
        let (mut files, project) = editable_project();
        let candidate = edited_material(&files);
        let previous = files.projects().to_vec();
        let path = project.parent().unwrap().join("assets/materials.yaml");
        files
            .files()
            .packages
            .borrow_mut()
            .insert(path, invalid.as_bytes().to_vec());
        assert!(
            files.store_project_material(candidate).is_err(),
            "{invalid}"
        );
        assert_eq!(files.projects(), previous);
        assert_eq!(
            library_bytes(&files, &project, "materials"),
            invalid.as_bytes()
        );
    }
}

#[test]
fn new_project_definition_creates_a_missing_library_and_survives_reload() {
    let (mut files, project) = editable_project();
    let candidate = edited_material(&files);
    files
        .files()
        .packages
        .borrow_mut()
        .remove(&project.parent().unwrap().join("assets/materials.yaml"));
    files.reload_project_assets(&project).unwrap();
    assert!(
        files
            .active_project()
            .unwrap()
            .assets
            .materials()
            .get("steel")
            .is_none()
    );
    files.store_project_material(candidate.clone()).unwrap();
    files.reload_project_assets(&project).unwrap();
    assert_eq!(
        files
            .active_project()
            .unwrap()
            .assets
            .materials()
            .get("steel"),
        Some(&candidate)
    );
}

#[test]
fn external_opaque_record_edits_are_retained_when_typed_values_are_unchanged() {
    let (mut files, project) = editable_project();
    let candidate = edited_material(&files);
    let path = project.parent().unwrap().join("assets/materials.yaml");
    let original = String::from_utf8(library_bytes(&files, &project, "materials")).unwrap();
    let extended = original.replace("id: steel,", "id: steel, futureExtension: {tag: external},");
    files
        .files()
        .packages
        .borrow_mut()
        .insert(path, extended.into_bytes());
    files.store_project_material(candidate).unwrap();
    let saved = String::from_utf8(library_bytes(&files, &project, "materials")).unwrap();
    assert!(saved.contains("tag: external"));
    files.reload_project_assets(&project).unwrap();
    assert_eq!(material_name(&files, "steel"), "Edited physical");
}

#[test]
fn storage_refuses_missing_project_location_and_wrong_asset_ownership() {
    let (mut files, _) = editable_project();
    let builtin = files
        .builtin_catalog()
        .unwrap()
        .materials()
        .get("steel")
        .unwrap()
        .clone();
    assert!(
        files
            .store_project_material(builtin)
            .unwrap_err()
            .to_string()
            .contains("project")
    );
    let asset = edited_material(&files);
    let (mut absent, _) = observed_catalog_files(false);
    assert!(absent.store_project_material(asset.clone()).is_err());
    let path = Path::new("/fixtures/no-library.opj");
    absent.files().packages.borrow_mut().insert(
        path.into(),
        ProjectConfiguration::default().encode().unwrap(),
    );
    absent.load_project(path).unwrap();
    assert!(
        absent
            .store_project_material(asset)
            .unwrap_err()
            .to_string()
            .contains("design-data")
    );
    assert!(
        absent
            .active_project()
            .unwrap()
            .assets
            .materials()
            .ordered()
            .next()
            .is_none()
    );
}

#[path = "asset_commands.rs"]
mod api_commands;
