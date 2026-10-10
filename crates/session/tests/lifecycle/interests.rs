// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn interest() -> DocumentInterestRecord {
    DocumentInterestRecord {
        client_id: "cam".into(),
        name: "toolpaths".into(),
        data_version: 3,
        ..Default::default()
    }
}

#[test]
fn interest_history_restores_owned_records_and_elides_identical_updates() {
    let mut session = session(None);
    for _ in 0..2 {
        session.begin("Register CAM").unwrap();
        session
            .edit(|snapshot, _| snapshot.interests.put(interest()))
            .unwrap();
        session.end().unwrap();
    }
    assert_eq!(session.undo_labels(), ["Register CAM"]);
    session.undo().unwrap();
    assert!(session.snapshot().interests.records().is_empty());
    session.redo().unwrap();
    let saved = session.snapshot().interests.clone();
    session.begin("Remove CAM").unwrap();
    session
        .edit(|snapshot, _| {
            assert!(snapshot.interests.remove("cam", "toolpaths"));
            Ok::<(), String>(())
        })
        .unwrap();
    session.abort().unwrap();
    assert_eq!(session.snapshot().interests, saved);
}

#[test]
fn commit_veto_restores_interest_records_and_history() {
    let mut session = session(Some(EVENT_TRANSACTION_COMMITTED));
    session.begin("Register CAM").unwrap();
    session
        .edit(|snapshot, _| snapshot.interests.put(interest()))
        .unwrap();
    assert!(session.end().is_err());
    assert!(session.snapshot().interests.records().is_empty());
    assert!(session.undo_labels().is_empty());
}
