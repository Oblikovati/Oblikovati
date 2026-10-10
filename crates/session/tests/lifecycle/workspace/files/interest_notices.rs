// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::interest::DocumentInterests;

fn interest_host(present: &[&str]) -> Host {
    let host = host(RecordingWorkspace {
        present_clients: present.iter().map(|name| (*name).into()).collect(),
        ..Default::default()
    });
    let mut interests = DocumentInterests::default();
    for (client, name, strength) in [
        ("cam", "paths", INTERESTED),
        ("cam", "tools", INTERESTED),
        ("withdrawn", "mesh", INTEREST_NONE),
        ("fea", "loads", INTERESTED),
    ] {
        interests
            .put(DocumentInterestRecord {
                client_id: client.into(),
                name: name.into(),
                interest_type: strength,
                ..Default::default()
            })
            .unwrap();
    }
    let package = DocumentPackage::decode(&fixture(1))
        .unwrap()
        .with_display_name("Custom title")
        .with_interests(&interests)
        .unwrap();
    host.files()
        .contents
        .borrow_mut()
        .insert(path("interests.opd"), package.encode().unwrap());
    host
}

#[test]
fn open_reports_absent_interests_after_binding_without_dirtying_or_changing_content() {
    let mut host = interest_host(&["fea"]);
    let before = host.files().contents.borrow()[&path("interests.opd")].clone();
    let id = open_file(
        &mut host,
        "interests.opd",
        &mut SequenceGuids(100),
        OpenActivation::Activate,
    );
    let observer = host.workspace().observer();
    assert_eq!(
        observer.interest_notices,
        [(id, "Custom title".into(), vec!["cam".into(), "cam".into()])]
    );
    assert!(!host.workspace().needs_save(id).unwrap());
    assert_eq!(
        host.workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .interests
            .records()
            .len(),
        4
    );
    assert_eq!(
        host.files().contents.borrow()[&path("interests.opd")],
        before
    );
    open_file(
        &mut host,
        "interests.opd",
        &mut SequenceGuids(200),
        OpenActivation::Activate,
    );
    assert_eq!(host.workspace().observer().interest_notices.len(), 1);
}

#[test]
fn present_clients_suppress_notice_but_matching_is_case_sensitive() {
    let mut host = interest_host(&["cam", "fea"]);
    open_file(
        &mut host,
        "interests.opd",
        &mut SequenceGuids(100),
        OpenActivation::Activate,
    );
    assert!(host.workspace().observer().interest_notices.is_empty());
    let mut host = interest_host(&["CAM", "fea"]);
    open_file(
        &mut host,
        "interests.opd",
        &mut SequenceGuids(100),
        OpenActivation::Activate,
    );
    assert_eq!(
        host.workspace().observer().interest_notices[0].2,
        ["cam", "cam"]
    );
}

#[test]
fn failed_open_does_not_emit_an_interest_notice() {
    let mut host = interest_host(&[]);
    host.files()
        .contents
        .borrow_mut()
        .insert(path("interests.opd"), b"invalid: document".to_vec());
    assert!(
        host.open(
            &path("interests.opd"),
            RecordingObserver::default(),
            "test",
            &mut SequenceGuids(100),
            OpenActivation::Activate
        )
        .is_err()
    );
    assert!(host.workspace().observer().interest_notices.is_empty());
}

#[test]
fn connected_clients_suppress_missing_interest_notice_until_last_connection_leaves() {
    let mut host = interest_host(&["fea"]);
    let first = host.connected_clients_mut().register("cam").unwrap();
    let second = host.connected_clients_mut().register("cam").unwrap();
    host.connected_clients_mut().unregister(first).unwrap();
    let id = open_file(
        &mut host,
        "interests.opd",
        &mut SequenceGuids(100),
        OpenActivation::Activate,
    );
    assert!(host.workspace().observer().interest_notices.is_empty());
    host.close_discarding_changes(id).unwrap();
    host.connected_clients_mut().unregister(second).unwrap();
    open_file(
        &mut host,
        "interests.opd",
        &mut SequenceGuids(200),
        OpenActivation::Activate,
    );
    assert_eq!(
        host.workspace().observer().interest_notices[0].2,
        ["cam", "cam"]
    );
}
