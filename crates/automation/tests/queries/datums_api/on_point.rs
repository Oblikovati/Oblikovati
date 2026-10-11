// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn associate(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    reference: &str,
) -> CreateWorkPointResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_points()
        .create(&CreateWorkPointArgs {
            kind: WORK_POINT_ON_POINT.0.into_owned(),
            refs: Some(vec![reference.into()]),
            ..Default::default()
        })
        .unwrap()
}
#[test]
fn typed_associations_follow_current_points_and_feed_directed_axes() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let first = create_point(&mut files, &mut source, [0.0, 0.0, 4.0], false);
    let second = associate(&mut files, &mut source, &first.ref_);
    let third = associate(&mut files, &mut source, &second.ref_);
    assert_eq!(point_rows(&files, false)[3].kind, WORK_POINT_ON_POINT.0);
    let axis = create_two_points(&mut files, &mut source, WORK_REF_CENTER, &third.ref_, false);
    assert!(axis.healthy);
    change_point(&mut files, document, 1, [0.0, 3.0, 0.0]);
    assert_eq!(
        point_rows(&files, false)[3].position,
        Some(vec![0.0, 3.0, 0.0])
    );
    assert_eq!(
        axis_rows(&files, false)[3].direction,
        Some(vec![0.0, 1.0, 0.0])
    );
    assert_suppressed_associations(&mut files, document);
}
fn suppress_driver(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Suppress point", |snapshot, _| {
            snapshot.work_geometry.set_suppressed(FeatureId(1), true)
        })
        .unwrap();
}
#[test]
fn associations_preserve_source_identity_through_undo_storage_and_copy() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = associate(&mut files, &mut source, WORK_REF_CENTER);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(point_rows(&files, false).len(), 1);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(point_rows(&files, false)[1].position, Some(vec![0.0; 3]));
    super::history::verify_stored_key(&mut files, document, &mut source, &point.ref_);
}
#[test]
fn malformed_association_references_decline_before_owner_mutation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    for refs in invalid_association_refs() {
        let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
            .work_points()
            .create(&CreateWorkPointArgs {
                kind: WORK_POINT_ON_POINT.0.into_owned(),
                refs,
                ..Default::default()
            });
        assert!(result.is_err());
    }
    assert_eq!(owner_revision(&files, document), before);
}

fn assert_suppressed_associations(files: &mut DiscoveryWorkspace, document: u64) {
    suppress_driver(files, document);
    assert!(
        point_rows(files, false)[2..]
            .iter()
            .all(|row| !row.healthy && row.position.is_none())
    );
    assert!(!axis_rows(files, false)[3].healthy);
}

fn invalid_association_refs() -> [Option<Vec<String>>; 5] {
    [
        None,
        Some(vec![]),
        Some(vec![WORK_REF_CENTER.into(); 2]),
        Some(vec![WORK_REF_X_AXIS.into()]),
        Some(vec!["datum/v1/99".into()]),
    ]
}
