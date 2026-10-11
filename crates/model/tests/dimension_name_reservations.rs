// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::MODEL_PARAM;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{
        DocumentEditHistory, DocumentSnapshot, SketchDimensionNameReservations,
        SketchReservationMarks,
    },
    identity::{RecipeIdAllocator, RecipeIdError},
    parameter::ParameterCollection,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchPlane,
    },
};
use std::collections::BTreeMap;
fn snapshot(name: &str) -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, name, MODEL_PARAM, "3 cm")
        .unwrap();
    let geometry = dimension_geometry(1);
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Dimensions",
            SketchPlane::xy(),
            geometry,
        )
        .unwrap();
    snapshot
}
fn dimension_geometry(parameter: u64) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry.add_point(SketchEntityId(2), [3.0, 0.0]).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Distance {
                points: [SketchEntityId(1), SketchEntityId(2)],
                measure: DistanceMeasure::Aligned,
                parameter,
            },
        )
        .unwrap();
    geometry
}
fn reservations(
    snapshot: &DocumentSnapshot,
    marks: Option<&BTreeMap<u64, u64>>,
) -> Result<SketchDimensionNameReservations, RecipeIdError> {
    SketchDimensionNameReservations::restore(&snapshot.sketches, &snapshot.parameters, 2, marks)
}
#[test]
fn legacy_bootstrap_reads_owned_canonical_names_and_leaves_renamed_sources_alone() {
    for (name, expected) in [
        ("d7", 8),
        ("d0", 1),
        ("d01", 0),
        ("d", 0),
        ("width", 0),
        ("d18446744073709551616", 0),
    ] {
        assert_bootstrapped_name(name, expected);
    }
    assert!(matches!(
        reservations(&snapshot(&format!("d{}", u64::MAX)), None),
        Err(RecipeIdError::Exhausted)
    ));
}
#[test]
fn explicit_marks_cover_live_names_and_reject_foreign_namespaces() {
    let snapshot = snapshot("d7");
    for marks in [
        BTreeMap::new(),
        BTreeMap::from([(1, 7)]),
        BTreeMap::from([(0, 9)]),
        BTreeMap::from([(3, 9)]),
    ] {
        assert!(reservations(&snapshot, Some(&marks)).is_err());
    }
    let names = reservations(&snapshot, Some(&BTreeMap::from([(1, 8), (2, 19)]))).unwrap();
    assert_eq!(names.marks(), BTreeMap::from([(1, 8), (2, 19)]));
    assert_missing_name_mark_diagnostic(&snapshot);
}

#[test]
fn colliding_names_are_consumed_across_independent_sketch_counters() {
    let mut parameters = ParameterCollection::new(Default::default());
    for (id, name) in [(1, "d0"), (2, "d2")] {
        parameters
            .insert_expression(id, name, MODEL_PARAM, "1 cm")
            .unwrap();
    }
    let mut names = SketchDimensionNameReservations::default();
    for (owner, expected) in [(1, "d1"), (1, "d3"), (2, "d1")] {
        assert_eq!(
            names
                .allocate(DocumentSketchId(owner), 2, &parameters)
                .unwrap(),
            expected
        );
    }
    assert_eq!(names.marks(), BTreeMap::from([(1, 4), (2, 2)]));
}
#[test]
fn retired_namespaces_and_exhaustion_keep_their_permanent_marks() {
    let snapshot = document_snapshot::empty_snapshot();
    let saved = BTreeMap::from([(1, 0), (2, u64::MAX)]);
    let mut names = reservations(&snapshot, Some(&saved)).unwrap();
    let before = names.marks();
    for owner in [0, 2, 3] {
        assert!(
            names
                .allocate(DocumentSketchId(owner), 2, &snapshot.parameters)
                .is_err()
        );
        assert_eq!(names.marks(), before);
    }
    assert_eq!(
        names
            .allocate(DocumentSketchId(1), 2, &snapshot.parameters)
            .unwrap(),
        "d0"
    );
}
fn history() -> DocumentEditHistory {
    DocumentEditHistory::new(
        snapshot("d0"),
        RecipeIdAllocator::restore(Some(1), &[1]).unwrap(),
    )
    .unwrap()
}
fn reserve(
    history: &mut DocumentEditHistory,
    fail: bool,
) -> Result<String, oblikovati_model::history::EditError<String>> {
    let token = history.begin("Reserve dimension").unwrap();
    let result = history.edit(&token, |snapshot, ids| {
        let name = ids
            .allocate_dimension_name(DocumentSketchId(1), &snapshot.parameters)
            .unwrap();
        if fail {
            return Err(format!("refused {name}"));
        }
        Ok(name)
    });
    history.abort(&token).unwrap();
    result
}
#[test]
fn failed_edits_and_abort_do_not_reuse_names() {
    let mut history = history();
    assert!(
        reserve(&mut history, true)
            .unwrap_err()
            .to_string()
            .contains("d1")
    );
    assert_eq!(reserve(&mut history, false).unwrap(), "d2");
    assert_eq!(
        history.sketch_reservations().dimension_names,
        BTreeMap::from([(1, 3)])
    );
    assert_eq!(history.snapshot().parameters.get(1).unwrap().name, "d0");
}
#[test]
fn undo_redo_and_deleted_sketches_do_not_rewind_names() {
    let mut history = history();
    let token = history.begin("Delete sketch").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            ids.allocate_dimension_name(DocumentSketchId(1), &snapshot.parameters)
                .unwrap();
            snapshot.sketches.remove(DocumentSketchId(1))
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    history.undo().unwrap();
    assert_eq!(reserve(&mut history, false).unwrap(), "d2");
    history.redo().unwrap();
    assert_eq!(history.sketch_reservations().dimension_names[&1], 3);
    assert_eq!(reserve(&mut history, false).unwrap(), "d3");
}
#[test]
fn restored_history_refuses_to_drop_or_rewind_retired_name_marks() {
    for supplied in [
        BTreeMap::new(),
        BTreeMap::from([(1, 1)]),
        BTreeMap::from([(1, 2), (2, 3)]),
    ] {
        let mut marks = history().sketch_reservations();
        marks.sketch = 2;
        marks.dimension_names = BTreeMap::from([(1, 2), (2, 4)]);
        let restored = history().with_sketch_reservations(marks.clone()).unwrap();
        marks.dimension_names = supplied;
        assert!(restored.with_sketch_reservations(marks).is_err());
    }
    assert_retired_history_restore();
}
fn assert_retired_history_restore() {
    let mut marks: SketchReservationMarks = history().sketch_reservations();
    marks.sketch = 2;
    marks.dimension_names = BTreeMap::from([(1, 5), (2, 7)]);
    let mut restored = history().with_sketch_reservations(marks).unwrap();
    assert_eq!(reserve(&mut restored, false).unwrap(), "d5");
}

fn assert_bootstrapped_name(name: &str, expected: u64) {
    let snapshot = snapshot(name);
    let mut names = reservations(&snapshot, None).unwrap();
    assert_eq!(
        names
            .allocate(DocumentSketchId(1), 2, &snapshot.parameters)
            .unwrap(),
        format!("d{expected}")
    );
    assert_eq!(names.marks(), BTreeMap::from([(1, expected + 1)]));
}

fn assert_missing_name_mark_diagnostic(snapshot: &DocumentSnapshot) {
    let error = reservations(snapshot, Some(&BTreeMap::new())).unwrap_err();
    assert_eq!(
        error,
        RecipeIdError::DimensionNameMark {
            sketch: 1,
            mark: 0,
            minimum: 8
        }
    );
    assert!(error.to_string().contains("ordinal 0: expected at least 8"));
}
#[test]
fn renamed_numeric_dimension_names_do_not_raise_the_saved_counter() {
    let mut snapshot = snapshot("d0");
    snapshot.parameters.rename(1, "d99").unwrap();
    let saved = BTreeMap::from([(1, 1)]);
    let mut names = reservations(&snapshot, Some(&saved)).unwrap();
    assert_eq!(
        names
            .allocate(DocumentSketchId(1), 2, &snapshot.parameters)
            .unwrap(),
        "d1"
    );
    let mut legacy = reservations(&snapshot, None).unwrap();
    assert_eq!(
        legacy
            .allocate(DocumentSketchId(1), 2, &snapshot.parameters)
            .unwrap(),
        "d0"
    );
}
