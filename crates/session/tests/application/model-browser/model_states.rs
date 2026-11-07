// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::model_state::ModelStateMembers;
fn members(host: &Host, id: u64) -> &ModelStateMembers {
    host.workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .model_states
        .as_ref()
        .unwrap()
}
fn author_member(host: &mut Host, id: u64, name: &str) -> u64 {
    host.commands(id)
        .unwrap()
        .run("Create model-state member", |snapshot, ids| {
            let reserved = ids
                .allocate_model_state()
                .map_err(|cause| cause.to_string())?;
            let members = snapshot.model_states.as_mut().unwrap();
            members.insert(reserved, name)?;
            members.activate(reserved)?;
            Ok::<_, String>(reserved)
        })
        .unwrap()
}
#[test]
fn part_members_survive_undo_redo_save_reopen_with_permanent_reservations() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = host
        .create_unbound(
            DocumentPackage::empty(DOCUMENT_PART, "Part1"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
        )
        .unwrap()
        .session_id;
    assert_eq!(members(&host, id).active().unwrap().name, "Primary");
    let reserved = author_member(&mut host, id, "Large");
    let expected = members(&host, id).clone();
    host.commands(id).unwrap().undo().unwrap();
    assert_eq!(members(&host, id).list().len(), 1);
    host.commands(id).unwrap().redo().unwrap();
    assert_eq!(members(&host, id), &expected);
    verify_reopened_members(&mut host, id, &expected, reserved, &mut source);
}
fn verify_reopened_members(
    host: &mut Host,
    id: u64,
    expected: &ModelStateMembers,
    reserved: u64,
    source: &mut SequenceGuids,
) {
    host.save_as(id, &path("states.opd"), "alpha", source)
        .unwrap();
    host.close_discarding_changes(id).unwrap();
    let reopened = host
        .open(
            &path("states.opd"),
            RecordingObserver::default(),
            "alpha",
            source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id();
    assert_eq!(members(host, reopened), expected);
    assert!(!host.workspace().needs_save(reopened).unwrap());
    let next = author_member(host, reopened, "Small");
    assert!(next > reserved);
}
