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
            expression: "10 cm".into(),
            text_point: None,
        }),
        None,
        Some(SketchPlacementFieldLock {
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
