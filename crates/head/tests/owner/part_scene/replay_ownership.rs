// SPDX-License-Identifier: GPL-2.0-only
//! Part replay slots remain scoped to their own document generations.
use super::*;
use oblikovati_automation::{PartReplayService, PartUpdateAccess};

fn open_named_prism(owner: &mut TestOwner, name: &str) -> u64 {
    let path = std::env::temp_dir().join(name);
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), stored_bytes(false));
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id
}

#[test]
fn headless_replay_tracks_each_document_and_its_own_generation() {
    let (mut owner, _) = owner();
    let first = open_named_prism(&mut owner, "replay-owner-first.opd");
    let second = open_named_prism(&mut owner, "replay-owner-second.opd");
    let mut replay = PartReplayService::default();
    assert!(replay.part_requires_update(&owner.files, first).unwrap());
    assert!(replay.part_requires_update(&owner.files, second).unwrap());
    assert!(
        replay
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    assert!(
        replay
            .update_part(&mut owner.files, second)
            .unwrap()
            .is_empty()
    );
    assert!(!replay.part_requires_update(&owner.files, first).unwrap());
    assert!(!replay.part_requires_update(&owner.files, second).unwrap());
    Client::new(owner.document_host())
        .documents()
        .activate(&ActivateDocumentArgs { id: first })
        .unwrap();
    set_parameter(&mut owner, "width", "6 cm");
    assert!(replay.part_requires_update(&owner.files, first).unwrap());
    assert!(!replay.part_requires_update(&owner.files, second).unwrap());
    assert!(
        replay
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    assert!(!replay.part_requires_update(&owner.files, first).unwrap());
    owner.files.commands(first).unwrap().undo().unwrap();
    assert!(replay.part_requires_update(&owner.files, first).unwrap());
    assert!(!replay.part_requires_update(&owner.files, second).unwrap());
    assert!(
        replay
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    owner.files.commands(first).unwrap().redo().unwrap();
    assert!(replay.part_requires_update(&owner.files, first).unwrap());
    assert!(!replay.part_requires_update(&owner.files, second).unwrap());
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: second,
            force: false,
        })
        .unwrap();
    assert!(
        replay
            .update_part(&mut owner.files, first)
            .unwrap()
            .is_empty()
    );
    let reopened = open_named_prism(&mut owner, "replay-owner-second.opd");
    assert_ne!(reopened, second);
    assert!(replay.part_requires_update(&owner.files, reopened).unwrap());
    assert!(!replay.part_requires_update(&owner.files, first).unwrap());
    assert!(
        replay
            .update_part(&mut owner.files, reopened)
            .unwrap()
            .is_empty()
    );
}
