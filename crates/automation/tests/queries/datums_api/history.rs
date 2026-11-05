// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn aborted_creation_retires_its_key_and_new_creation_survives_undo_storage_and_copy() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files
        .commands(document)
        .unwrap()
        .begin("Datum group")
        .unwrap();
    let retired = create_point(&mut files, &mut source, [1.0, 2.0, 3.0], false);
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(point_rows(&files, false).len(), 1);
    let kept = create_point(&mut files, &mut source, [1.0, 2.0, 3.0], false);
    assert_eq!(retired.index, kept.index);
    assert_ne!(retired.ref_, kept.ref_);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(point_rows(&files, false).len(), 1);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(point_rows(&files, false)[1].ref_, kept.ref_);
    verify_stored_key(&mut files, document, &mut source, &kept.ref_);
}
pub(super) fn verify_stored_key(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    key: &str,
) {
    files.save(document, "test", source).unwrap();
    let document = reopen_with_observer(files, document, source, QueryObserver::default());
    assert_eq!(point_rows(files, false)[1].ref_, key);
    verify_copied_key(files, document, source, key);
}
fn verify_copied_key(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    key: &str,
) {
    let copy = Path::new("/fixtures/datums-copy.obk");
    files
        .save_copy(document, copy, None, "test", source)
        .unwrap();
    let copied = open_copied_datum(files, source, copy);
    assert_eq!(point_rows(files, false)[1].ref_, key);
    let next = create_point(files, source, [4.0, 5.0, 6.0], false);
    assert_ne!(next.ref_, key);
    assert_eq!(next.index, 2);
    assert!(!files.workspace().needs_save(document).unwrap());
    assert!(files.workspace().needs_save(copied).unwrap());
}
#[test]
fn encoded_creation_is_not_published_when_the_commit_is_vetoed() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    files.save(document, "test", &mut source).unwrap();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .work_points()
        .create(&CreateWorkPointArgs {
            at: Some(vec![0.0; 3]),
            ..Default::default()
        });
    assert_datum_decline(result, "locked");
    assert_eq!(point_rows(&files, false).len(), 1);
    assert_eq!(owner_revision(&files, document).1, before.1);
}
#[test]
fn visibility_is_independent_of_construction_and_survives_recipe_storage() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_point(&mut files, &mut source, [1.0, 2.0, 3.0], true);
    files
        .commands(document)
        .unwrap()
        .run("Hide datum", |snapshot, _| {
            snapshot
                .work_geometry
                .set_datum_visible(FeatureId(1), false)
        })
        .unwrap();
    assert_visibility(&files);
    files.save(document, "test", &mut source).unwrap();
    reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    assert_visibility(&files);
}
fn assert_visibility(files: &DiscoveryWorkspace) {
    assert_eq!(point_rows(files, false).len(), 1);
    let point = &point_rows(files, true)[1];
    assert!(!point.visible);
    assert!(point.construction);
    assert_eq!(point.position, Some(vec![1.0, 2.0, 3.0]));
}

pub(super) fn open_copied_datum(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    copy: &Path,
) -> u64 {
    files
        .open(
            copy,
            QueryObserver::default(),
            "test",
            source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id()
}
