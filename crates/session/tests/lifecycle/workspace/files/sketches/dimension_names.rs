// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
fn reserve_name(host: &mut Host, document: u64, sketch: DocumentSketchId) -> String {
    host.commands(document)
        .unwrap()
        .run("Reserve dimension name", |snapshot, ids| {
            ids.allocate_dimension_name(sketch, &snapshot.parameters)
        })
        .unwrap()
}
fn assert_saved_name(host: &Host, file: &str, sketch: DocumentSketchId, expected: &str) {
    let bytes = host.files().contents.borrow()[&path(file)].clone();
    let package = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = package.restore_sketches().unwrap();
    let parameters = package.restore_document_parameters().unwrap().parameters;
    assert_eq!(
        restored
            .dimension_names
            .allocate(sketch, restored.sketch_ids.high_water(), &parameters)
            .unwrap(),
        expected
    );
}
fn failed_name(host: &mut Host, document: u64, sketch: DocumentSketchId) {
    let result = host.commands(document).unwrap().run(
        "Refused dimension",
        |snapshot, ids| -> Result<(), String> {
            let name = ids
                .allocate_dimension_name(sketch, &snapshot.parameters)
                .unwrap();
            assert_eq!(name, "d0");
            Err("creation reply refused".into())
        },
    );
    assert!(result.is_err());
    assert!(host.workspace().needs_save(document).unwrap());
}
#[test]
fn failed_reply_and_group_abort_dirty_the_owner_and_survive_save_and_reopen() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    failed_name(&mut host, document, sketch);
    host.commands(document)
        .unwrap()
        .begin("Abort dimensions")
        .unwrap();
    assert_eq!(reserve_name(&mut host, document, sketch), "d1");
    host.commands(document).unwrap().abort().unwrap();
    host.save(document, "alpha", &mut source).unwrap();
    assert!(!host.workspace().needs_save(document).unwrap());
    assert_saved_name(&host, "a.obk", sketch, "d2");
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_eq!(reserve_name(&mut host, reopened, sketch), "d2");
}
#[test]
fn deleted_sketch_names_survive_undo_redo_and_saved_copies() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    assert_eq!(reserve_name(&mut host, document, sketch), "d0");
    host.commands(document)
        .unwrap()
        .delete_sketch(sketch)
        .unwrap();
    host.commands(document).unwrap().undo().unwrap();
    host.commands(document).unwrap().redo().unwrap();
    assert_deleted_name_copy(&mut host, &mut source, document, sketch);
}

#[test]
fn malformed_name_marks_refuse_open_before_registration_or_guid_allocation() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1700);
    let bytes = b"schemaVersion: 3\ndocumentType: 1\nmodel:\n  parameters: []\n  planarSketches:\n    sketchHighWater: 1\n    entityHighWater: 0\n    dimensionNameOrdinals: {0: 1}\n    recipe: {version: 1, sketches: []}\n";
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), bytes.to_vec());
    let result = host.open(
        &path("a.obk"),
        RecordingObserver::default(),
        "alpha",
        &mut source,
        OpenActivation::Activate,
    );
    assert!(result.unwrap_err().to_string().contains("recipe ID 0"));
    assert!(host.workspace().documents().is_empty());
    assert_eq!(source.0, 1700);
}

fn assert_deleted_name_copy(
    host: &mut Host,
    source: &mut SequenceGuids,
    document: u64,
    sketch: DocumentSketchId,
) {
    host.save_copy(document, &path("names-copy.obk"), None, "alpha", source)
        .unwrap();
    assert_saved_name(host, "names-copy.obk", sketch, "d1");
    let copied = open_file(host, "names-copy.obk", source, OpenActivation::Background);
    assert_eq!(reserve_name(host, copied, sketch), "d1");
    assert_copied_sketch_absent(host, copied);
}

fn assert_copied_sketch_absent(host: &Host, document: u64) {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    assert!(snapshot.sketches.sketches().is_empty());
}
