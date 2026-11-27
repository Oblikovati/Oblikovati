// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::SketchPlacementFieldLock,
    sketch::{Sketch2D, SketchPlacementRecipe},
};
fn placement() -> SketchPlacementRecipe {
    SketchPlacementRecipe::straight_slot([[0.0; 2], [10.0, 0.0]], 2.0).unwrap()
}
fn locks() -> Vec<Option<SketchPlacementFieldLock>> {
    vec![
        Some(SketchPlacementFieldLock {
            reference: false,
            expression: "10 cm".into(),
            text_point: None,
        }),
        None,
        Some(SketchPlacementFieldLock {
            reference: false,
            expression: "2 cm".into(),
            text_point: None,
        }),
    ]
}
#[test]
fn recipe_commit_uses_one_history_command_and_replays_all_owned_fields() {
    let mut session =
        super::sketch_fixture::owned_sketch_session(None, "Placement", Sketch2D::default());
    let before = session.snapshot().clone();
    let report = session
        .create_sketch_placement(DocumentSketchId(1), &placement(), &locks())
        .unwrap();
    assert_eq!(
        session
            .analyze_sketch(report.sketch)
            .unwrap()
            .degrees_of_freedom,
        3
    );
    assert_eq!(session.undo_labels().len(), 1);
    let after = session.snapshot().clone();
    session.undo().unwrap();
    assert!(before.sketches.same_recipe(&session.snapshot().sketches));
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
    session.redo().unwrap();
    assert!(
        after
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
    assert!(
        after
            .parameters
            .same_evaluated_state(&session.snapshot().parameters)
    );
}
#[test]
fn recipe_commit_veto_and_group_abort_keep_the_previous_snapshot_and_burn_ids() {
    for veto in [None, Some(EVENT_TRANSACTION_COMMITTED)] {
        let mut session =
            super::sketch_fixture::owned_sketch_session(veto, "Placement", Sketch2D::default());
        let before = session.snapshot().clone();
        if veto.is_none() {
            session.begin("Placement group").unwrap();
        }
        let result = session.create_sketch_placement(DocumentSketchId(1), &placement(), &locks());
        assert_eq!(result.is_err(), veto.is_some());
        if veto.is_none() {
            session.abort().unwrap();
        }
        assert!(before.sketches.same_recipe(&session.snapshot().sketches));
        assert!(
            before
                .parameters
                .same_recipe(&session.snapshot().parameters)
        );
        assert_eq!(session.history().sketch_reservations().entity, 11);
        assert!(session.undo_labels().is_empty());
    }
}
#[test]
fn empty_locked_values_leave_only_the_geometric_recipe() {
    let mut session =
        super::sketch_fixture::owned_sketch_session(None, "Placement", Sketch2D::default());
    let report = session
        .create_sketch_placement(
            DocumentSketchId(1),
            &placement(),
            &[Some(SketchPlacementFieldLock {
                reference: false,
                expression: String::new(),
                text_point: None,
            })],
        )
        .unwrap();
    assert!(report.dimensions.is_empty());
    assert_eq!(
        session
            .analyze_sketch(report.sketch)
            .unwrap()
            .degrees_of_freedom,
        5
    );
}

fn arc_inference_session(pin_center: bool, persist: bool) -> DocumentSession<RecordingObserver> {
    use oblikovati_model::sketch::SketchEntityId;
    let mut source = Sketch2D::default();
    let reference = SketchEntityId(900);
    source.add_point(reference, [2.0005, 0.0]).unwrap();
    if pin_center {
        source.add_point(SketchEntityId(901), [0.0, 0.0]).unwrap();
    }
    let mut session = super::sketch_fixture::owned_sketch_session(None, "Arc", source);
    session.begin("Inference settings").unwrap();
    session
        .edit(|snapshot, _| {
            let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
            settings.persist_inferred_constraints = persist;
            snapshot.sketch_settings = Some(settings);
            Ok::<_, oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    session.end().unwrap();
    let sketch = DocumentSketchId(1);
    session
        .set_sketch_inference_scope(sketch, INFERENCE_SCOPE_ALL_GEOMETRY, &[])
        .unwrap();
    session
}
fn nonpersistent_arc() -> SketchPlacementRecipe {
    SketchPlacementRecipe::arc([0.0, 0.0], [[2.0001, 0.0], [0.0, 2.0]], true).unwrap()
}
#[test]
fn nonpersistent_arc_inference_keeps_source_and_snapped_endpoint_without_relation() {
    use oblikovati_model::sketch::SketchEntityId;
    let mut session = arc_inference_session(false, false);
    let sketch = DocumentSketchId(1);
    let reference = SketchEntityId(900);
    let arc = nonpersistent_arc();
    let before = session.snapshot().clone();
    let verifier = super::placement_verification::RecordingCircularVerifier::default();
    let preview = session
        .preview_sketch_placement(sketch, &arc, &[], &verifier)
        .unwrap();
    let report = session.create_sketch_placement(sketch, &arc, &[]).unwrap();
    let geometry = session.snapshot().sketches.get(sketch).unwrap().geometry();
    assert!(
        report.relations.is_empty(),
        "temporary inference must not be saved"
    );
    assert_eq!(geometry.point(reference).unwrap().position, [2.0005, 0.0]);
    assert_eq!(
        geometry.point(report.points[1]).unwrap().position,
        [2.0005, 0.0]
    );
    assert!(geometry.constraints().is_empty());
    assert!(geometry.same_recipe(&preview.geometry));
    let mut disturbed = geometry.clone();
    disturbed.set_position(reference, [3.0, 0.0]).unwrap();
    assert!(report.verify_inference_preserved(&disturbed).is_err());
    let mut disturbed = geometry.clone();
    disturbed
        .set_position(report.points[1], [3.0, 0.0])
        .unwrap();
    assert!(report.verify_inference_preserved(&disturbed).is_err());
    let after = session.snapshot().clone();
    session.undo().unwrap();
    assert!(before.sketches.same_recipe(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(
        after
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
}

#[test]
fn nonpersistent_arc_inference_survives_a_driving_radius() {
    use oblikovati_model::sketch::SketchEntityId;
    let mut session = arc_inference_session(false, false);
    let sketch = DocumentSketchId(1);
    let lock = SketchPlacementFieldLock {
        reference: false,
        expression: "1.5 cm".into(),
        text_point: None,
    };
    let verifier = super::placement_verification::RecordingCircularVerifier::default();
    let preview = session
        .preview_sketch_placement(
            sketch,
            &nonpersistent_arc(),
            &[Some(lock.clone())],
            &verifier,
        )
        .unwrap();
    let before = session.snapshot().clone();
    let report = session
        .create_sketch_placement(sketch, &nonpersistent_arc(), &[Some(lock)])
        .unwrap();
    let geometry = session.snapshot().sketches.get(sketch).unwrap().geometry();
    assert!(report.relations.is_empty());
    assert_eq!(
        geometry.point(SketchEntityId(900)).unwrap().position,
        [2.0005, 0.0]
    );
    assert_eq!(
        geometry.point(report.points[1]).unwrap().position,
        [2.0005, 0.0]
    );
    assert_eq!(report.dimensions.len(), 1);
    assert!(geometry.same_recipe(&preview.geometry));
    let after = session.snapshot().clone();
    session.undo().unwrap();
    assert!(before.sketches.same_recipe(&session.snapshot().sketches));
    session.redo().unwrap();
    assert!(
        after
            .sketches
            .same_evaluated_state(&session.snapshot().sketches)
    );
}

#[test]
fn incompatible_nonpersistent_inference_and_radius_refuse_atomically() {
    let mut session = arc_inference_session(true, false);
    let sketch = DocumentSketchId(1);
    let before = session.snapshot().clone();
    let lock = SketchPlacementFieldLock {
        reference: false,
        expression: "1.5 cm".into(),
        text_point: None,
    };
    assert!(
        session
            .create_sketch_placement(sketch, &nonpersistent_arc(), &[Some(lock)])
            .is_err()
    );
    assert!(before.sketches.same_recipe(&session.snapshot().sketches));
    assert!(
        before
            .parameters
            .same_recipe(&session.snapshot().parameters)
    );
}

#[test]
fn persisted_arc_inference_keeps_its_existing_target_fixed() {
    use oblikovati_model::sketch::SketchEntityId;
    let mut session = arc_inference_session(false, true);
    let sketch = DocumentSketchId(1);
    let report = session
        .create_sketch_placement(sketch, &nonpersistent_arc(), &[])
        .unwrap();
    let geometry = session.snapshot().sketches.get(sketch).unwrap().geometry();
    assert_eq!(report.relations.len(), 1);
    assert_eq!(
        geometry.point(SketchEntityId(900)).unwrap().position,
        [2.0005, 0.0]
    );
    let actual = geometry.point(report.points[1]).unwrap().position;
    assert!((actual[0] - 2.0005).abs() < 1e-8 && actual[1].abs() < 1e-8);
}
