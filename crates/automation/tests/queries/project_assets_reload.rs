// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn replace_library(files: &DiscoveryWorkspace, project: &Path, name: &str, bytes: &[u8]) {
    let library = project.parent().unwrap().join("assets").join(name);
    files
        .files()
        .packages
        .borrow_mut()
        .insert(library, bytes.to_vec());
}

fn appearance_name(files: &DiscoveryWorkspace) -> String {
    Client::new(FileQueryHost::new(files))
        .appearances()
        .get(&AssetRefArgs {
            id: "project-look".into(),
        })
        .unwrap()
        .display_name
}

fn replace_appearance(files: &DiscoveryWorkspace, project: &Path) {
    replace_library(
        files,
        project,
        "appearances.yaml",
        b"appearances: [{id: project-look, name: Updated, albedo: '#102030'}]",
    );
}

#[test]
fn reload_adopts_both_libraries_without_replacing_project_registration() {
    let (mut files, _) = observed_catalog_files(false);
    let project = project_files(&files, "one", "Before");
    files.load_project(&project).unwrap();
    project_files(&files, "one", "After");
    replace_appearance(&files, &project);
    assert_eq!(material_name(&files, "steel"), "Before");
    files.reload_project_assets(&project).unwrap();
    assert_eq!(material_name(&files, "steel"), "After");
    assert_eq!(appearance_name(&files), "Updated");
    assert_eq!(files.projects().len(), 1);
    assert_eq!(files.active_project().unwrap().path, project);
    assert!(files.workspace().documents().is_empty());
}

#[test]
fn failed_reload_keeps_both_previous_libraries_and_selection() {
    let (mut files, _) = observed_catalog_files(false);
    let project = project_files(&files, "one", "Before");
    files.load_project(&project).unwrap();
    replace_appearance(&files, &project);
    replace_library(&files, &project, "materials.yaml", b"materials: [");
    let previous = files.projects().to_vec();
    let refusal = files.reload_project_assets(&project).unwrap_err();
    assert!(refusal.to_string().contains("materials.yaml"));
    assert_eq!(files.projects(), previous);
    assert_eq!(material_name(&files, "steel"), "Before");
    assert_eq!(appearance_name(&files), "Project look");
    assert_eq!(files.active_project().unwrap().path, project);
}

#[test]
fn reload_of_inactive_project_does_not_select_it() {
    let (mut files, _) = observed_catalog_files(false);
    let first = project_files(&files, "one", "First");
    let second = project_files(&files, "two", "Second");
    files.load_project(&first).unwrap();
    files.load_project(&second).unwrap();
    project_files(&files, "two", "Changed");
    files.reload_project_assets(&second).unwrap();
    assert_eq!(material_name(&files, "steel"), "First");
    assert_eq!(files.active_project().unwrap().path, first);
    files.select_project(&second).unwrap();
    assert_eq!(material_name(&files, "steel"), "Changed");
}

#[test]
fn unknown_reload_identity_does_no_io() {
    let (mut files, _) = observed_catalog_files(false);
    let reads = files.files().reads;
    assert!(
        files
            .reload_project_assets(Path::new("/unknown.opj"))
            .is_err()
    );
    assert_eq!(files.files().reads, reads);
}
