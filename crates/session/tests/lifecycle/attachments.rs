// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::attachment::AttachmentContents;

#[test]
fn attachment_visibility_and_removal_participate_in_undo_redo_and_abort() {
    let mut session = session(None);
    session.begin("Attach").unwrap();
    session
        .edit(|snapshot, _| {
            snapshot
                .attachments
                .restore("file", "source", AttachmentContents::Generic, true)
        })
        .unwrap();
    session.end().unwrap();
    for _ in 0..2 {
        session.begin("Hide").unwrap();
        session
            .edit(|snapshot, _| snapshot.attachments.set_browser_visible("file", false))
            .unwrap();
        session.end().unwrap();
    }
    assert_eq!(session.undo_labels(), ["Attach", "Hide"]);
    session.undo().unwrap();
    assert!(
        session
            .snapshot()
            .attachments
            .get("file")
            .unwrap()
            .browser_visible()
    );
    session.redo().unwrap();
    let expected = session.snapshot().attachments.clone();
    session.begin("Remove").unwrap();
    session
        .edit(|snapshot, _| {
            snapshot.attachments.remove("file");
            Ok::<(), String>(())
        })
        .unwrap();
    session.abort().unwrap();
    assert_eq!(session.snapshot().attachments, expected);
}

#[test]
fn attachment_commit_veto_restores_original_collection() {
    let mut session = session(Some(EVENT_TRANSACTION_COMMITTED));
    session.begin("Attach").unwrap();
    session
        .edit(|snapshot, _| {
            snapshot
                .attachments
                .restore("file", "source", AttachmentContents::Generic, true)
        })
        .unwrap();
    assert!(session.end().is_err());
    assert!(session.snapshot().attachments.entries().is_empty());
    assert!(session.undo_labels().is_empty());
}
