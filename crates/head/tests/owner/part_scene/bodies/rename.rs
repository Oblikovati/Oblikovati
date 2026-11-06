// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn renamed(owner: &mut TestOwner, index: i64, name: &str) -> BodyInfo {
    Client::new(owner.document_host())
        .body()
        .rename(&BodyRenameArgs {
            body_index: index,
            name: name.into(),
        })
        .unwrap()
        .body
}

#[test]
fn body_rename_returns_current_summary_and_supports_undo_redo_and_clear() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let mut expected = listed(&mut owner)[0].clone();
    expected.name = " Bracket α ".into();
    assert_eq!(renamed(&mut owner, 0, &expected.name), expected);
    assert_eq!(listed(&mut owner), [expected.clone()]);
    assert!(owner.files.workspace().needs_save(document).unwrap());
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(listed(&mut owner)[0].name, "Solid1");
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(listed(&mut owner)[0], expected);
    assert_eq!(renamed(&mut owner, 0, "").name, "Solid1");
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(listed(&mut owner)[0], expected);
}

#[test]
fn body_name_tracks_the_persistent_key_across_recompute_and_index_changes() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let original = renamed(&mut owner, 1, "Independent body");
    set_parameter(&mut owner, "width", "6 cm");
    assert_eq!(listed(&mut owner)[1].key, original.key);
    assert_eq!(listed(&mut owner)[1].name, original.name);
    set_parameter(&mut owner, "depth", "0 cm");
    let remaining = listed(&mut owner);
    assert_eq!(remaining.len(), 1);
    assert_eq!(
        (remaining[0].index, &remaining[0].key, &remaining[0].name),
        (0, &original.key, &original.name)
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(listed(&mut owner)[1].name, original.name);
}

#[test]
fn body_rename_is_saved_reopened_and_cleared_by_the_typed_route() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let named = renamed(&mut owner, 0, "Saved body");
    let reopened = save_reopen_body(&mut owner, document);
    assert_eq!(listed(&mut owner)[0], named);
    assert!(!owner.files.workspace().needs_save(reopened).unwrap());
    assert_eq!(renamed(&mut owner, 0, "").name, "Solid1");
    save_reopen_body(&mut owner, reopened);
    assert_eq!(listed(&mut owner)[0].name, "Solid1");
    assert_eq!(listed(&mut owner)[0].key, named.key);
}

pub(super) fn save_reopen_body(owner: &mut TestOwner, document: u64) -> u64 {
    let path = owner
        .files
        .path(document)
        .unwrap()
        .to_str()
        .unwrap()
        .to_owned();
    Client::new(owner.document_host())
        .documents()
        .save(&SaveDocumentArgs { document })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path,
            ..Default::default()
        })
        .unwrap()
        .id
}

#[test]
fn rejected_body_indices_do_not_change_names_history_or_dirty_state() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let before = listed(&mut owner);
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    for body_index in [-1, 1, i64::MAX] {
        let error = Client::new(owner.document_host())
            .body()
            .rename(&BodyRenameArgs {
                body_index,
                name: "Rejected".into(),
            })
            .unwrap_err()
            .to_string();
        assert!(error.contains(&body_index.to_string()) && error.contains("0..1"));
    }
    assert_eq!(listed(&mut owner), before);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(!owner.files.workspace().needs_save(document).unwrap());
}

#[test]
fn body_rename_joins_and_aborts_an_explicit_transaction() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    owner
        .files
        .commands(document)
        .unwrap()
        .begin("Body metadata")
        .unwrap();
    renamed(&mut owner, 0, "Temporary");
    assert_eq!(listed(&mut owner)[0].name, "Temporary");
    owner.files.commands(document).unwrap().abort().unwrap();
    assert_eq!(listed(&mut owner)[0].name, "Solid1");
    assert!(!owner.files.workspace().needs_save(document).unwrap());
}

#[test]
fn body_rename_save_failure_preserves_disk_and_live_name_until_retry() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let before = owner.documents().files().saved.borrow().clone();
    renamed(&mut owner, 0, "Unsaved body");
    owner.documents().files().fail_stage.set(true);
    assert!(
        Client::new(owner.document_host())
            .documents()
            .save(&SaveDocumentArgs { document })
            .is_err()
    );
    assert_eq!(*owner.documents().files().saved.borrow(), before);
    assert_eq!(listed(&mut owner)[0].name, "Unsaved body");
    assert!(owner.files.workspace().needs_save(document).unwrap());
    owner.documents().files().fail_stage.set(false);
    save_reopen_body(&mut owner, document);
    assert_eq!(listed(&mut owner)[0].name, "Unsaved body");
}

#[test]
fn body_rename_requires_an_active_part_and_injected_topology_owner() {
    let (mut owner, _) = owner();
    let request = BodyRenameArgs {
        body_index: 0,
        name: "New name".into(),
    };
    assert!(
        Client::new(owner.document_host())
            .body()
            .rename(&request)
            .is_err()
    );
    let document = open_prism(&mut owner, false);
    let mut guids = OwnerThreadGuids::default();
    let host = oblikovati_automation::FileHost::new(&mut owner.files, &mut guids, "test");
    assert!(
        Client::new(host)
            .body()
            .rename(&request)
            .unwrap_err()
            .to_string()
            .contains("injected exact topology owner")
    );
    assert!(!owner.files.workspace().needs_save(document).unwrap());
}

#[test]
fn unchanged_body_names_follow_the_shared_snapshot_dirty_policy() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    renamed(&mut owner, 0, "");
    assert!(!owner.files.workspace().needs_save(document).unwrap());
    renamed(&mut owner, 0, "Saved");
    Client::new(owner.document_host())
        .documents()
        .save(&SaveDocumentArgs { document })
        .unwrap();
    renamed(&mut owner, 0, "Saved");
    assert!(!owner.files.workspace().needs_save(document).unwrap());
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(listed(&mut owner)[0].name, "Solid1");
    renamed(&mut owner, 0, "");
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(listed(&mut owner)[0].name, "Saved");
}
