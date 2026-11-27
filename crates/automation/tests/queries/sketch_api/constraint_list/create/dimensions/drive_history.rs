// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn group_abort_and_undo_redo_restore_the_complete_dimension_edit() {
    let (mut files, document, mut source) = owned(DOCUMENT_PART);
    let original = dimensions(&files);
    files
        .commands(document)
        .unwrap()
        .begin("Dimension group")
        .unwrap();
    drive_call(&mut files, &mut source, &edit_request()).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(dimensions(&files), original);
    assert_eq!(target(&files, document), 1.0);
    drive_call(&mut files, &mut source, &edit_request()).unwrap();
    let edited = dimensions(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(dimensions(&files), original);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(dimensions(&files), edited);
    assert_eq!(target(&files, document), 3.0);
}
#[test]
fn stored_dimension_settings_reopen_and_copy_without_losing_drive_limits() {
    let (mut files, document, mut source) = owned(DOCUMENT_PART);
    drive_call(&mut files, &mut source, &edit_request()).unwrap();
    let edited = dimensions(&files);
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(dimensions(&files), edited);
    copy_settings(&mut files, reopened, &mut source);
    assert_eq!(dimensions(&files), edited);
    let copied = files.workspace().active_document().unwrap();
    let request = DriveDimensionArgs {
        expression: "10 cm".into(),
        ..Default::default()
    };
    drive_call(&mut files, &mut source, &request).unwrap();
    assert_eq!(target(&files, copied), 3.0);
    assert_eq!(dimensions(&files), edited);
}
#[test]
fn commit_veto_preserves_all_settings_and_the_original_parameter() {
    let (mut files, document, mut source) = owned(DOCUMENT_PART);
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let reopened = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, reopened);
    let original = dimensions(&files);
    expect_api_decline(
        drive_call(&mut files, &mut source, &edit_request()),
        "locked",
    );
    assert_eq!(dimensions(&files), original);
    assert_eq!(target(&files, reopened), 1.0);
    let after = owner_revision(&files, reopened);
    assert_eq!((after.1, after.2), (before.1, before.2));
}
#[test]
fn protected_user_targets_cannot_be_mutated_by_a_dimension_drive() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let rows = dimensions(&files);
    expect_api_decline(
        drive_call(&mut files, &mut source, &edit_request()),
        "owned Model scalar",
    );
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(dimensions(&files), rows);
}

fn copy_settings(files: &mut DiscoveryWorkspace, reopened: u64, source: &mut QueryGuids) {
    crate::file_discovery::sketch_api::entities::activate_copy(
        files,
        reopened,
        source,
        Path::new("/fixtures/dimension-drive-copy.obk"),
    );
}
