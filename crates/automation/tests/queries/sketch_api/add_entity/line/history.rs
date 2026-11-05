// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn line_inference_survives_save_copy_and_keeps_new_reference_namespaces() {
    let (mut files, document, mut source) = fresh();
    let mut request = line_input([[0.0; 2], [4.0, 0.01]]);
    request.centerline = true;
    add_call(&mut files, &mut source, &request).unwrap();
    let original = rows(&files);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), original);
    activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/inferred-line-copy.obk"),
    );
    let copy = rows(&files);
    assert_copy_namespaces(&original, &copy);
    verify_keys(&files, &copy);
}
#[test]
fn grouped_line_creation_undo_and_abort_do_not_reuse_reserved_entities() {
    let (mut files, document, mut source) = fresh();
    files
        .commands(document)
        .unwrap()
        .begin("Author profile")
        .unwrap();
    let first = add_line(&mut files, &mut source, [[0.0; 2], [4.0, 0.01]]);
    files.commands(document).unwrap().end().unwrap();
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Author profile"
    );
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files).len(), 1);
    let second = aborted_line(&mut files, document, &mut source);
    assert!(second.entity_id > first.entity_id);
    assert_eq!(rows(&files).len(), 1);
}

fn aborted_line(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
) -> AddSketchEntityResult {
    files
        .commands(document)
        .unwrap()
        .begin("Discard profile")
        .unwrap();
    let second = add_line(files, source, [[0.0; 2], [4.0, 0.01]]);
    files.commands(document).unwrap().abort().unwrap();
    second
}
