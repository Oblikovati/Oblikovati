// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    offset::{create_offset, edit_driver, scalar_id},
    planes::{fixed, plane_rows},
};

fn flip(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    index: i64,
) -> Result<FlipWorkPlaneResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .work_planes()
        .flip_normal(&FlipWorkPlaneArgs { index })
}

#[test]
fn typed_flip_keeps_key_origin_and_owned_scalar_and_double_flip_restores_normal() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create_offset(&mut files, &mut source, "width / 2");
    let id = scalar_id(&files, document);
    let reply = flip(&mut files, &mut source, created.index).unwrap();
    assert_eq!(reply.plane.ref_, created.ref_);
    assert_eq!(reply.plane.origin, Some(vec![0.0, 0.0, 0.5]));
    assert_eq!(reply.plane.normal, Some(vec![0.0, 0.0, -1.0]));
    assert_eq!(scalar_id(&files, document), id);
    edit_driver(&mut files, document, "6 cm");
    assert_eq!(
        plane_rows(&files, true)[3].origin,
        Some(vec![0.0, 0.0, 3.0])
    );
    assert_eq!(
        flip(&mut files, &mut source, 3).unwrap().plane.normal,
        Some(vec![0.0, 0.0, 1.0])
    );
}

#[test]
fn hidden_construction_index_flips_the_full_collection_entry() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let hidden = fixed(&mut files, &mut source, true, false);
    let shown = fixed(&mut files, &mut source, false, true);
    let reply = flip(&mut files, &mut source, hidden.index).unwrap();
    assert_eq!(reply.plane.ref_, hidden.ref_);
    assert_eq!(reply.plane.normal, Some(vec![0.0, 0.0, -1.0]));
    assert_eq!(plane_rows(&files, false)[3].ref_, shown.ref_);
    assert_eq!(
        plane_rows(&files, false)[3].normal,
        Some(vec![0.0, 0.0, 1.0])
    );
}

#[test]
fn origin_and_invalid_index_refusals_leave_the_owner_unchanged() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    fixed(&mut files, &mut source, false, true);
    let before = owner_revision(&files, document);
    for index in [-1, 0, 1, 2, 4, i64::MAX] {
        assert!(flip(&mut files, &mut source, index).is_err());
    }
    assert_eq!(owner_revision(&files, document), before);
}

#[test]
fn undo_and_redo_preserve_orientation_edits() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    fixed(&mut files, &mut source, false, true);
    flip(&mut files, &mut source, 3).unwrap();
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        plane_rows(&files, true)[3].normal,
        Some(vec![0.0, 0.0, 1.0])
    );
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(
        plane_rows(&files, true)[3].normal,
        Some(vec![0.0, 0.0, -1.0])
    );
}

#[test]
fn grouped_flip_abort_restores_the_previous_orientation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    fixed(&mut files, &mut source, false, true);
    flip(&mut files, &mut source, 3).unwrap();
    files
        .commands(document)
        .unwrap()
        .begin("Flip group")
        .unwrap();
    flip(&mut files, &mut source, 3).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(
        plane_rows(&files, true)[3].normal,
        Some(vec![0.0, 0.0, -1.0])
    );
}

#[test]
fn flipped_owned_offset_survives_reopen_and_copy() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let created = create_offset(&mut files, &mut source, "width / 2");
    flip(&mut files, &mut source, 3).unwrap();
    files.save(document, "test", &mut source).unwrap();
    let document =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    let path = Path::new("/fixtures/flipped-copy.obk");
    files
        .save_copy(document, path, None, "test", &mut source)
        .unwrap();
    let copied = super::history::open_copied_datum(&mut files, &mut source, path);
    edit_driver(&mut files, copied, "8 cm");
    let row = &plane_rows(&files, true)[3];
    assert_eq!(row.ref_, created.ref_);
    assert_eq!(row.origin, Some(vec![0.0, 0.0, 4.0]));
    assert_eq!(row.normal, Some(vec![0.0, 0.0, -1.0]));
}

#[test]
fn failed_offset_can_store_a_flip_and_recover_without_stale_geometry() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    create_offset(&mut files, &mut source, "width / 2");
    edit_driver(&mut files, document, "missing");
    let reply = flip(&mut files, &mut source, 3).unwrap();
    assert!(!reply.plane.healthy && reply.plane.normal.is_none() && reply.plane.origin.is_none());
    files.save(document, "test", &mut source).unwrap();
    let document =
        reopen_with_observer(&mut files, document, &mut source, QueryObserver::default());
    edit_driver(&mut files, document, "4 cm");
    assert_eq!(
        plane_rows(&files, true)[3].normal,
        Some(vec![0.0, 0.0, -1.0])
    );
}

#[test]
fn vetoed_flip_does_not_publish_its_encoded_candidate() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    fixed(&mut files, &mut source, false, true);
    files.save(document, "test", &mut source).unwrap();
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    assert_datum_decline(flip(&mut files, &mut source, 3), "locked");
    assert_eq!(
        plane_rows(&files, true)[3].normal,
        Some(vec![0.0, 0.0, 1.0])
    );
    assert_eq!(owner_revision(&files, document).1, before.1);
}
