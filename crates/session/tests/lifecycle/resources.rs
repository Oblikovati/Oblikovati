// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::resource::DocumentResource;

#[test]
fn resource_edits_support_noop_detection_undo_redo_and_abort() {
    let mut session = session(None);
    for _ in 0..2 {
        session.begin("Import").unwrap();
        session
            .edit(|snapshot, _| {
                snapshot.resources.insert(
                    "mesh".into(),
                    DocumentResource::imported("ObjFile", "mesh.obj", b"v 0 0 0\n"),
                );
                Ok::<(), String>(())
            })
            .unwrap();
        session.end().unwrap();
    }
    assert_eq!(session.undo_labels(), ["Import"]);
    session.undo().unwrap();
    assert!(session.snapshot().resources.is_empty());
    session.redo().unwrap();
    let expected = session.snapshot().resources.clone();
    session.begin("Remove").unwrap();
    session
        .edit(|snapshot, _| {
            snapshot.resources.clear();
            Ok::<(), String>(())
        })
        .unwrap();
    session.abort().unwrap();
    assert_eq!(session.snapshot().resources, expected);
}

#[test]
fn resource_commit_veto_discards_candidate_payloads() {
    let mut session = session(Some(EVENT_TRANSACTION_COMMITTED));
    session.begin("Import").unwrap();
    session
        .edit(|snapshot, _| {
            snapshot.resources.insert(
                "font".into(),
                DocumentResource::imported("TrueTypeFont", "font.ttf", &[0, 255]),
            );
            Ok::<(), String>(())
        })
        .unwrap();
    assert!(session.end().is_err());
    assert!(session.snapshot().resources.is_empty());
    assert!(session.undo_labels().is_empty());
}
