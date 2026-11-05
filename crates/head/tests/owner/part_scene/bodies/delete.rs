// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    FeatureEvaluationStatus,
    part::{DeleteBodyDefinition, PartFeatureDefinition, PartFeatureFailure},
};
fn deleted(owner: &mut TestOwner, index: i64) -> Vec<BodyInfo> {
    Client::new(owner.document_host())
        .body()
        .delete(&BodyIndexArgs { body_index: index })
        .unwrap()
        .bodies
        .unwrap_or_default()
}
fn delete_id(owner: &TestOwner, document: u64) -> FeatureId {
    owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()
        .last()
        .unwrap()
        .id()
}
#[test]
fn body_delete_changes_native_results_and_mass_with_one_undo_step() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let original = listed(&mut owner);
    let properties = Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: 1,
            density_g_cm3: 1.,
            accuracy: String::new(),
        })
        .unwrap();
    let exact = retained_report(&owner, document, FeatureId(2));
    let mut expected = original[1].clone();
    expected.index = 0;
    expected.name = "Solid1".into();
    assert_eq!(deleted(&mut owner, 0), [expected]);
    assert_eq!(retained_report(&owner, document, FeatureId(2)), exact);
    let remaining = Client::new(owner.document_host())
        .analysis()
        .mass_properties(&MassPropertiesArgs {
            density_g_cm3: 1.,
            ..Default::default()
        })
        .unwrap();
    assert_eq!(remaining, properties);
    let id = delete_id(&owner, document);
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(session.undo_labels(), ["Delete Body"]);
    assert!(
        owner
            .part_scenes
            .current(session)
            .unwrap()
            .deleted_body(id)
            .is_some()
    );
    assert!(owner.files.workspace().needs_save(document).unwrap());
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(listed(&mut owner), original);
    // Undo and abort cannot reuse an allocated feature identity.
    assert!(owner.files.workspace().needs_save(document).unwrap());
    assert!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .history()
            .feature_high_water()
            >= 3
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(listed(&mut owner).len(), 1);
    assert!(deleted(&mut owner, 0).is_empty());
    let mass = Client::new(owner.document_host())
        .analysis()
        .mass_properties(&MassPropertiesArgs::default())
        .unwrap();
    assert_eq!(mass.volume_mm3, 0.);
    assert!(
        owner
            .take_scene()
            .unwrap()
            .unwrap()
            .scene
            .triangles
            .is_empty()
    );
}
#[test]
fn body_delete_survives_recompute_save_and_reopen_and_can_be_suppressed() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let key = listed(&mut owner)[0].key.clone();
    deleted(&mut owner, 1);
    set_parameter(&mut owner, "width", "6 cm");
    assert_eq!(listed(&mut owner)[0].key, key);
    let reopened = rename::save_reopen_body(&mut owner, document);
    assert_eq!(listed(&mut owner).len(), 1);
    assert_eq!(listed(&mut owner)[0].key, key);
    let id = delete_id(&owner, reopened);
    owner
        .files
        .commands(reopened)
        .unwrap()
        .run("Suppress deletion", |snapshot, _| {
            snapshot.work_geometry.set_suppressed(id, true)
        })
        .unwrap();
    assert_eq!(listed(&mut owner).len(), 2);
    owner.files.commands(reopened).unwrap().undo().unwrap();
    assert_eq!(listed(&mut owner).len(), 1);
}
#[test]
fn body_delete_uses_current_indices_but_keeps_a_stable_replay_target() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let original = listed(&mut owner);
    set_parameter(&mut owner, "depth", "0 cm");
    assert_eq!(listed(&mut owner)[0].key, original[1].key);
    assert!(deleted(&mut owner, 0).is_empty());
    set_parameter(&mut owner, "depth", "2 cm");
    assert_eq!(listed(&mut owner)[0].key, original[0].key);
    assert_eq!(listed(&mut owner).len(), 1);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert!(listed(&mut owner).is_empty());
}
#[test]
fn body_delete_rejections_and_transaction_abort_leave_original_bodies() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let original = listed(&mut owner);
    for index in [-1, 2, i64::MAX] {
        assert!(
            Client::new(owner.document_host())
                .body()
                .delete(&BodyIndexArgs { body_index: index })
                .is_err()
        );
    }
    assert!(!owner.files.workspace().needs_save(document).unwrap());
    assert!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .undo_labels()
            .is_empty()
    );
    owner
        .files
        .commands(document)
        .unwrap()
        .begin("Batch")
        .unwrap();
    deleted(&mut owner, 1);
    owner.files.commands(document).unwrap().abort().unwrap();
    assert_eq!(listed(&mut owner), original);
    // Undo and abort cannot reuse an allocated feature identity.
    assert!(owner.files.workspace().needs_save(document).unwrap());
    assert!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .history()
            .feature_high_water()
            >= 3
    );
}
#[test]
fn body_delete_foreign_document_key_fails_without_consuming_geometry() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    deleted(&mut owner, 0);
    let id = delete_id(&owner, document);
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Foreign reference", |snapshot, _| {
            let PartFeatureDefinition::DeleteBody(target) =
                snapshot.work_geometry.get(id).unwrap().definition()
            else {
                panic!("expected deletion")
            };
            let mut key = target.key().to_vec();
            key[21] ^= 1; // The semantic payload begins with the document GUID.
            let definition =
                PartFeatureDefinition::DeleteBody(DeleteBodyDefinition::new(key).unwrap());
            snapshot
                .work_geometry
                .replace(id, definition.clone(), definition.reads())
        })
        .unwrap();
    assert_eq!(listed(&mut owner).len(), 1);
    let update = owner.take_scene().unwrap().unwrap();
    assert!(matches!(
        update.status.health[1].status,
        FeatureEvaluationStatus::Failed(PartFeatureFailure::BodyReference(_))
    ));
    assert_eq!(update.status.visible_bodies, 1);
}

#[test]
fn body_delete_requires_an_active_part_and_native_owner() {
    let (mut owner, _) = owner();
    let request = BodyIndexArgs { body_index: 0 };
    assert!(
        Client::new(owner.document_host())
            .body()
            .delete(&request)
            .is_err()
    );
    let document = open_prism(&mut owner, false);
    let mut guids = OwnerThreadGuids::default();
    let host = oblikovati_automation::FileHost::new(&mut owner.files, &mut guids, "test");
    assert!(
        Client::new(host)
            .body()
            .delete(&request)
            .unwrap_err()
            .to_string()
            .contains("injected exact topology owner")
    );
    assert!(!owner.files.workspace().needs_save(document).unwrap());
}

#[test]
fn body_delete_appends_after_the_existing_end_of_part_without_moving_it() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Cutoff", |snapshot, _| {
            snapshot.work_geometry.set_end_of_part(Some(1))
        })
        .unwrap();
    let original = listed(&mut owner);
    assert_eq!(original.len(), 1);
    assert_eq!(deleted(&mut owner, 0), original);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .work_geometry
            .end_of_part(),
        Some(1)
    );
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Restore tail", |snapshot, _| {
            snapshot.work_geometry.set_end_of_part(None)
        })
        .unwrap();
    let remaining = listed(&mut owner);
    assert_eq!(remaining.len(), 1);
    assert_ne!(remaining[0].key, original[0].key);
}

fn retained_report(
    owner: &TestOwner,
    document: u64,
    feature: FeatureId,
) -> (oblikovati_martensite::PrismProbeReport, Vec<Vec<u8>>) {
    let session = owner.files.workspace().session(document).unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let body = owner
        .part_scenes
        .current(session)
        .unwrap()
        .body(feature)
        .unwrap();
    let key = body.body_reference_key(session.history(), guid).unwrap();
    let report = body
        .body_report(session.history(), guid, key)
        .unwrap()
        .clone();
    (
        report,
        body.reference_keys(session.history(), guid).unwrap(),
    )
}
