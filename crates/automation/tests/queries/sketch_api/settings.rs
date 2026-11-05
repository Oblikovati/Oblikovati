// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch_settings::{SketchSettings, default_sketch_settings};
fn settings(files: &DiscoveryWorkspace, document: u64) -> Option<SketchSettings> {
    files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketch_settings
        .clone()
}
fn store(files: &mut DiscoveryWorkspace, document: u64, settings: Option<SketchSettings>) {
    files
        .commands(document)
        .unwrap()
        .run("Sketch defaults", |snapshot, _| {
            snapshot.sketch_settings = settings;
            Ok::<(), String>(())
        })
        .unwrap();
}
#[test]
fn explicit_defaults_are_dirty_and_undo_restores_absence() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files.save(document, "test", &mut source).unwrap();
    assert!(!owner_revision(&files, document).2);
    store(&mut files, document, Some(default_sketch_settings()));
    assert!(owner_revision(&files, document).2);
    assert_eq!(owner_revision(&files, document).1, ["Sketch defaults"]);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(settings(&files, document), None);
    assert!(!owner_revision(&files, document).2);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(settings(&files, document), Some(default_sketch_settings()));
}
#[test]
fn save_reopen_and_copy_keep_all_values_for_part_assembly_and_drawing() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        let mut custom = default_sketch_settings();
        custom.infer_constraints = false;
        custom.x_snap_spacing = 0.75;
        store(&mut files, document, Some(custom.clone()));
        let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
        assert_eq!(settings(&files, document), Some(custom.clone()));
        assert!(!owner_revision(&files, document).2);
        entities::activate_copy(
            &mut files,
            document,
            &mut source,
            Path::new("/fixtures/settings-copy.obk"),
        );
        let copy = files.workspace().active_document().unwrap();
        assert_eq!(settings(&files, copy), Some(custom));
    }
}
#[test]
fn noops_keep_one_step_and_group_abort_restores_all_fields() {
    let (mut files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    for _ in 0..2 {
        store(&mut files, document, Some(default_sketch_settings()));
    }
    assert_eq!(owner_revision(&files, document).1, ["Sketch defaults"]);
    files
        .commands(document)
        .unwrap()
        .begin("Clear defaults")
        .unwrap();
    store(&mut files, document, None);
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(settings(&files, document), Some(default_sketch_settings()));
}
#[test]
fn commit_veto_preserves_settings_and_saved_state() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let result = files
        .commands(document)
        .unwrap()
        .run("Sketch defaults", |snapshot, _| {
            snapshot.sketch_settings = Some(default_sketch_settings());
            Ok::<(), String>(())
        });
    assert!(result.is_err());
    assert_eq!(settings(&files, document), None);
    assert!(!owner_revision(&files, document).2);
    assert!(owner_revision(&files, document).1.is_empty());
}
#[test]
fn clearing_saved_record_is_captured_and_independent_documents_keep_their_defaults() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    store(&mut files, document, Some(default_sketch_settings()));
    files.save(document, "test", &mut source).unwrap();
    entities::activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/settings-copy.obk"),
    );
    let copy = files.workspace().active_document().unwrap();
    store(&mut files, copy, Some(SketchSettings::default()));
    assert_eq!(settings(&files, document), Some(default_sketch_settings()));
    assert_eq!(settings(&files, copy), Some(SketchSettings::default()));
    store(&mut files, document, None);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(settings(&files, document), None);
    assert_eq!(settings(&files, copy), Some(SketchSettings::default()));
}
#[test]
fn malformed_template_settings_are_refused_before_creation_registers_an_owner() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let package = DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\nsketchSettings: {xSnapSpacing: .nan}\n",
    )
    .unwrap();
    let result = files.create(
        Path::new("/fixtures/sketch.obk"),
        package,
        Default::default(),
        "test",
        &mut source,
    );
    assert!(result.unwrap_err().to_string().contains("xSnapSpacing"));
    assert!(files.workspace().documents().is_empty());
    assert_eq!(source.0, 0);
}
