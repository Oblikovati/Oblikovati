// SPDX-License-Identifier: GPL-2.0-only
//! Changed sketch replay adoption checks.
use super::*;

#[test]
fn changed_sketch_replay_requires_a_new_part_generation() {
    use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
    let mut session = face_session();
    let mut slot = PartSlot::default();
    publish_face(&mut session, &mut slot, 2.5);
    let datums = slot.current(&session).unwrap().datum_evaluation();
    let request = DocumentSketchCreation {
        name: "Face sketch".into(),
        placement: SketchPlacement::WorkPlane(FeatureId(2)),
        orientation: None,
        project_origin: true,
    };
    session
        .run("Create face sketch", |snapshot, ids| {
            snapshot.create_document_sketch_with_datums(&request, ids, &datums)
        })
        .unwrap();
    let before = session.generation();
    let replay = session
        .prepare_part_evaluation()
        .recompute_sketches(&mut NamedFaceEvaluator, Default::default())
        .unwrap();
    assert!(!session.adopt_sketch_replay(replay, &mut slot).unwrap());
    assert!(session.generation() > before);
    assert!(slot.current(&session).is_err());
}

#[test]
fn changed_sketch_replay_rejects_a_foreign_slot_before_refresh() {
    use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
    let mut foreign = session(8);
    let mut slot = PartSlot::default();
    publish(&mut foreign, &mut slot);
    let mut owner = face_session();
    let mut owner_slot = PartSlot::default();
    publish_face(&mut owner, &mut owner_slot, 2.5);
    let datums = owner_slot.current(&owner).unwrap().datum_evaluation();
    let request = DocumentSketchCreation {
        name: "Face sketch".into(),
        placement: SketchPlacement::WorkPlane(FeatureId(2)),
        orientation: None,
        project_origin: true,
    };
    owner
        .run("Create face sketch", |snapshot, ids| {
            snapshot.create_document_sketch_with_datums(&request, ids, &datums)
        })
        .unwrap();
    let before = owner.generation();
    let replay = owner
        .prepare_part_evaluation()
        .recompute_sketches(&mut NamedFaceEvaluator, Default::default())
        .unwrap();
    assert!(matches!(
        owner.adopt_sketch_replay(replay, &mut slot),
        Err(PartSketchAdoptionError::Evaluation(
            PartEvaluationError::ForeignDocument { .. }
        ))
    ));
    assert_eq!(owner.generation(), before);
    assert_eq!(current_depth(&slot, &foreign), 2.);
}
