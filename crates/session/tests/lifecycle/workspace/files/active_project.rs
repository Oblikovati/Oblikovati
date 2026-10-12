// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_persistence::project_file::ProjectConfiguration;

fn insert_project(host: &Host, name: &str, workspace: &str) {
    let configuration = ProjectConfiguration {
        name: name.into(),
        workspace: workspace.into(),
        templates: "templates".into(),
        ..Default::default()
    };
    host.files()
        .contents
        .borrow_mut()
        .insert(path(name), configuration.encode().unwrap());
}

#[test]
fn canonical_project_registration_selects_first_and_switches_without_document_changes() {
    let mut host = host(RecordingWorkspace::default());
    assert!(host.active_project().is_none());
    assert!(
        host.resolve_project_reference(Path::new("a.obk"))
            .unwrap_err()
            .to_string()
            .contains("active project")
    );
    insert_project(&host, "a.opj", "alpha");
    insert_project(&host, "b.opj", "beta");
    host.load_project(&path("a.opj")).unwrap();
    host.load_project(&path("b.opj")).unwrap();
    assert_eq!(host.active_project().unwrap().configuration.name, "a.opj");
    assert_eq!(host.projects().len(), 2);
    let reads = host.files().reads;
    host.load_project(&path("a.opj")).unwrap();
    assert_eq!(host.files().reads, reads);
    let id = open_file(
        &mut host,
        "a.obk",
        &mut SequenceGuids(1000),
        OpenActivation::Activate,
    );
    host.select_project(&path("b.opj")).unwrap();
    assert_eq!(host.active_project().unwrap().path, path("b.opj"));
    assert_eq!(host.workspace().active_document(), Some(id));
    assert!(!host.workspace().needs_save(id).unwrap());
    assert!(host.select_project(&path("unregistered.opj")).is_err());
    assert_eq!(host.active_project().unwrap().path, path("b.opj"));
}

#[test]
fn failed_load_keeps_registration_and_selection_intact() {
    let mut host = host(RecordingWorkspace::default());
    insert_project(&host, "a.opj", "alpha");
    host.load_project(&path("a.opj")).unwrap();
    host.files()
        .contents
        .borrow_mut()
        .insert(path("bad.opj"), b"schemaVersion: 999".to_vec());
    let error = host.load_project(&path("bad.opj")).unwrap_err();
    assert!(error.to_string().contains("999"));
    assert!(std::error::Error::source(&error).is_some());
    assert!(host.load_project(&path("absent.opj")).is_err());
    assert_eq!(host.projects().len(), 1);
    assert_eq!(host.active_project().unwrap().path, path("a.opj"));
}

#[test]
fn active_project_resolves_references_and_creates_from_its_template() {
    let mut host = host(RecordingWorkspace::default());
    insert_project(&host, "a.opj", "alpha");
    insert_project(&host, "b.opj", "beta");
    host.files()
        .contents
        .borrow_mut()
        .insert(path("alpha/pin.opd"), fixture(10));
    host.files()
        .contents
        .borrow_mut()
        .insert(path("beta/pin.opd"), fixture(11));
    host.files()
        .contents
        .borrow_mut()
        .insert(path("templates/part.opd"), fixture(12));
    host.load_project(&path("a.opj")).unwrap();
    host.load_project(&path("b.opj")).unwrap();
    assert_eq!(
        host.resolve_project_reference(Path::new("pin.opd"))
            .unwrap(),
        Some(path("alpha/pin.opd"))
    );
    host.select_project(&path("b.opj")).unwrap();
    assert_eq!(
        host.resolve_project_reference(Path::new("pin.opd"))
            .unwrap(),
        Some(path("beta/pin.opd"))
    );
    assert!(
        host.resolve_project_reference(Path::new("absent.opd"))
            .unwrap()
            .is_none()
    );
    let metadata = NewFileMetadata {
        file_name: path("created.opd").to_str().unwrap().into(),
        ..Default::default()
    };
    let created = host
        .create_from_active_project_template(
            DOCUMENT_PART,
            &metadata,
            RecordingObserver::default(),
            "test",
            &mut SequenceGuids(1000),
        )
        .unwrap();
    assert_eq!(
        width(
            host.workspace()
                .session(created.session_id)
                .unwrap()
                .snapshot()
        ),
        1.0
    );
    assert!(host.workspace().needs_save(created.session_id).unwrap());
}

#[test]
fn aliases_share_one_canonical_project_and_do_not_reload_configuration() {
    let mut disk = WorkspaceFiles::default();
    disk.aliases
        .insert(path("alias.opj"), path("canonical.opj"));
    let configuration = ProjectConfiguration {
        name: "Canonical".into(),
        ..Default::default()
    };
    disk.contents
        .borrow_mut()
        .insert(path("canonical.opj"), configuration.encode().unwrap());
    let mut host = Host::new(RecordingWorkspace::default(), disk);
    assert_eq!(
        host.load_project(&path("alias.opj")).unwrap().path,
        path("canonical.opj")
    );
    host.files()
        .contents
        .borrow_mut()
        .insert(path("canonical.opj"), b"invalid: changed".to_vec());
    assert_eq!(
        host.load_project(&path("canonical.opj"))
            .unwrap()
            .configuration,
        configuration
    );
    assert_eq!(host.projects().len(), 1);
    assert_eq!(host.files().reads, 1);
}

#[test]
fn missing_asset_files_are_empty_and_asset_parse_failure_does_not_register_a_project() {
    let mut host = host(RecordingWorkspace::default());
    let configuration = ProjectConfiguration {
        name: "asset project".into(),
        design_data: "assets".into(),
        ..Default::default()
    };
    host.files()
        .contents
        .borrow_mut()
        .insert(path("assets.opj"), configuration.encode().unwrap());
    let project = host.load_project(&path("assets.opj")).unwrap();
    assert!(project.assets.materials().ordered().next().is_none());
    host.files()
        .contents
        .borrow_mut()
        .insert(path("bad-assets.opj"), configuration.encode().unwrap());
    host.files()
        .contents
        .borrow_mut()
        .insert(path("assets/materials.yaml"), b"materials: [".to_vec());
    assert!(
        host.load_project(&path("bad-assets.opj"))
            .unwrap_err()
            .to_string()
            .contains("materials.yaml")
    );
    assert_eq!(host.projects().len(), 1);
    assert_eq!(host.active_project().unwrap().path, path("assets.opj"));
}
