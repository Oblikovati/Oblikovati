// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    history::{
        DocumentEditHistory, DocumentSnapshot, SketchConstraintReservations, SketchReservationMarks,
    },
    identity::{RecipeIdAllocator, RecipeIdError},
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchPlane},
};
use std::collections::BTreeMap;
fn fixed_geometry(point: u64, constraint: u64) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry
        .add_point(SketchEntityId(point), [1.0, 2.0])
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(constraint),
            PlanarConstraint::Fix {
                point: SketchEntityId(point),
                position: [1.0, 2.0],
            },
        )
        .unwrap();
    geometry
}
fn local_sketches() -> DocumentSketches {
    let mut sketches = DocumentSketches::default();
    for (sketch, point) in [(1, 1), (2, 2)] {
        sketches
            .add(
                DocumentSketchId(sketch),
                "Local",
                SketchPlane::xy(),
                fixed_geometry(point, 7),
            )
            .unwrap();
    }
    sketches
}
fn history() -> DocumentEditHistory {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot.sketches = local_sketches();
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
}
fn marks(constraints: &[(u64, u64)]) -> SketchReservationMarks {
    SketchReservationMarks {
        sketch: 3,
        entity: 2,
        constraints: constraints.iter().copied().collect(),
    }
}
#[test]
fn missing_legacy_marks_bootstrap_independent_local_namespaces() {
    let mut reservations =
        SketchConstraintReservations::restore(&local_sketches(), 2, None).unwrap();
    assert_eq!(reservations.marks(), BTreeMap::from([(1, 7), (2, 7)]));
    assert_eq!(reservations.allocate(DocumentSketchId(1), 2).unwrap().0, 8);
    assert_eq!(reservations.allocate(DocumentSketchId(1), 2).unwrap().0, 9);
    assert_eq!(reservations.allocate(DocumentSketchId(2), 2).unwrap().0, 8);
}
#[test]
fn explicit_marks_must_cover_every_live_constraint_and_namespace() {
    for supplied in [
        BTreeMap::new(),
        BTreeMap::from([(1, 7)]),
        BTreeMap::from([(1, 6), (2, 7)]),
    ] {
        assert!(
            SketchConstraintReservations::restore(&local_sketches(), 2, Some(&supplied)).is_err()
        );
    }
    assert!(SketchConstraintReservations::restore(&local_sketches(), 1, None).is_err());
    for supplied in [BTreeMap::from([(0, 1)]), BTreeMap::from([(3, 1)])] {
        assert!(
            SketchConstraintReservations::restore(&DocumentSketches::default(), 2, Some(&supplied))
                .is_err()
        );
    }
}
#[test]
fn retired_namespaces_and_exhaustion_never_lose_their_marks() {
    let supplied = BTreeMap::from([(1, 19), (2, 0), (3, u64::MAX)]);
    let mut reservations =
        SketchConstraintReservations::restore(&DocumentSketches::default(), 3, Some(&supplied))
            .unwrap();
    assert_eq!(
        reservations.marks(),
        BTreeMap::from([(1, 19), (3, u64::MAX)])
    );
    assert_eq!(reservations.allocate(DocumentSketchId(1), 3).unwrap().0, 20);
    assert_eq!(reservations.allocate(DocumentSketchId(2), 3).unwrap().0, 1);
    assert_allocation_failures(&mut reservations);
}
fn assert_allocation_failures(reservations: &mut SketchConstraintReservations) {
    let before = reservations.marks();
    for (sketch, cause) in [(3, RecipeIdError::Exhausted), (0, RecipeIdError::ZeroId)] {
        assert_allocation_diagnostic(reservations, sketch, cause);
    }
    assert_allocation_diagnostic(
        reservations,
        4,
        RecipeIdError::BeyondHighWater {
            id: 4,
            high_water: 3,
        },
    );
    assert_eq!(reservations.marks(), before);
}
#[test]
fn history_restore_retains_retired_marks_and_refuses_rewind_or_drop() {
    let retained = [(1, 7), (2, 7), (3, 30)];
    for supplied in [&[(1, 7), (2, 7)][..], &[(1, 7), (2, 7), (3, 29)][..]] {
        assert!(
            history()
                .with_sketch_reservations(marks(&retained))
                .unwrap()
                .with_sketch_reservations(marks(supplied))
                .is_err()
        );
    }
    assert_retired_allocation(&retained);
}
fn assert_retired_allocation(retained: &[(u64, u64)]) {
    let mut history = history().with_sketch_reservations(marks(retained)).unwrap();
    let token = history.begin("Retired owner").unwrap();
    history
        .edit(&token, |_, ids| -> Result<(), String> {
            let allocated = ids.allocate_sketch_constraint(DocumentSketchId(3)).unwrap();
            assert_eq!(allocated.0, 31);
            Ok(())
        })
        .unwrap();
    history.abort(&token).unwrap();
    assert_eq!(history.sketch_reservations().constraints[&3], 31);
}
#[path = "sketch_constraint_reservations/transitions.rs"]
mod transitions;
fn assert_allocation_diagnostic(
    reservations: &mut SketchConstraintReservations,
    sketch: u64,
    cause: RecipeIdError,
) {
    use std::error::Error;
    let failure = reservations
        .allocate(DocumentSketchId(sketch), 3)
        .unwrap_err();
    assert_eq!(failure.sketch.0, sketch);
    assert_eq!(failure.cause, cause);
    assert!(
        failure
            .to_string()
            .contains(&format!("sketch {sketch} constraint"))
    );
    assert!(failure.source().is_some());
}
