// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::DocumentSketchId;
#[derive(Debug, PartialEq)]
struct PreviewReservations {
    parameter: u64,
    sketch: u64,
    entity: u64,
    constraint: u64,
    name: String,
    feature: u64,
    derived: i64,
}
fn reserve_every_namespace(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<PreviewReservations, String> {
    let sketch = DocumentSketchId(5);
    Ok(PreviewReservations {
        parameter: ids.allocate().unwrap(),
        sketch: ids.allocate_sketch().unwrap().0,
        entity: ids.allocate_sketch_entity().unwrap().0,
        constraint: ids.allocate_sketch_constraint(sketch).unwrap().0,
        name: ids
            .allocate_dimension_name(sketch, &snapshot.parameters)
            .unwrap(),
        feature: ids.allocate_feature().unwrap().0,
        derived: ids.allocate_derived_table().unwrap(),
    })
}
#[test]
fn preview_preserves_deleted_reservations_and_does_not_consume_any_namespace() {
    let original = history();
    let mut history = DocumentEditHistory::new(
        original.snapshot().clone(),
        RecipeIdAllocator::restore(Some(70), &[1]).unwrap(),
    )
    .unwrap()
    .with_sketch_reservations(SketchReservationMarks {
        sketch: 10,
        entity: 20,
        constraints: [(5, 30)].into(),
        dimension_names: [(5, 40)].into(),
    })
    .unwrap()
    .with_feature_reservations(50)
    .unwrap()
    .with_derived_reservations(60)
    .unwrap();
    let marks = history.sketch_reservations();
    let expected = PreviewReservations {
        parameter: 71,
        sketch: 11,
        entity: 21,
        constraint: 31,
        name: "d40".into(),
        feature: 51,
        derived: 61,
    };
    assert_eq!(history.preview(reserve_every_namespace).unwrap(), expected);
    assert_eq!(history.preview(reserve_every_namespace).unwrap(), expected);
    assert_eq!(history.sketch_reservations(), marks);
    assert_eq!(
        (
            history.high_water(),
            history.feature_high_water(),
            history.derived_high_water()
        ),
        (70, 50, 60)
    );
    let token = history.begin("Real edit").unwrap();
    assert_eq!(
        history.edit(&token, reserve_every_namespace).unwrap(),
        expected
    );
    history.abort(&token).unwrap();
}
#[test]
fn preview_reads_open_edits_without_changing_generation_navigation_or_the_active_token() {
    let mut history = history();
    let token = history.begin("Edit width").unwrap();
    set_width(&mut history, &token, "20 mm");
    let generation = history.generation();
    let preview = history
        .preview(|snapshot, ids| {
            let before = snapshot
                .parameters
                .get(1)
                .unwrap()
                .current_value()
                .unwrap()
                .value;
            snapshot.parameters.set_expression(1, "30 mm").unwrap();
            Ok::<_, String>((before, ids.allocate().unwrap()))
        })
        .unwrap();
    assert_eq!(preview, (2.0, 2));
    assert_eq!(width(&history), 2.0);
    assert_eq!(history.generation(), generation);
    assert_eq!(history.high_water(), 1);
    assert!(history.undo_labels().is_empty());
    history.commit(&token, false).unwrap();
    history.undo().unwrap();
    let labels = history.redo_labels();
    history.preview(|_, _| Ok::<_, String>(())).unwrap();
    assert_eq!(history.redo_labels(), labels);
}
#[test]
fn failed_and_unreserved_preview_edits_leave_original_state_and_reservations_untouched() {
    let history = history();
    let failed = history
        .preview(|snapshot, ids| {
            snapshot.parameters.set_expression(1, "30 mm").unwrap();
            ids.allocate().unwrap();
            Err::<(), _>("declined preview")
        })
        .unwrap_err();
    assert!(matches!(failed, EditError::Mutation("declined preview")));
    let invalid = history.preview(|snapshot, _| {
        snapshot
            .parameters
            .insert_expression(99, "unreserved", USER_PARAM, "1 cm")
    });
    assert!(matches!(
        invalid,
        Err(EditError::History(HistoryError::UnreservedIdentity(99)))
    ));
    assert_eq!(history.high_water(), 1);
    assert_eq!(width(&history), 1.0);
}
