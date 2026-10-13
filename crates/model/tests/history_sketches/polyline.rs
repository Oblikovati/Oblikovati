// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::{SketchPolylineCreation, SketchPrimitiveCreationError};
use oblikovati_model::identity::RecipeIdError;
use std::error::Error;
fn polyline_history() -> DocumentEditHistory {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Polyline",
            SketchPlane::xy(),
            Sketch2D::default(),
        )
        .unwrap();
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
}
fn input(closed: bool) -> SketchPolylineCreation {
    SketchPolylineCreation {
        points: vec![[0.0, 0.0], [2.0, 0.0], [2.0, 2.0]],
        closed,
        construction: true,
    }
}
#[test]
fn polyline_owns_unique_hidden_points_and_shares_them_in_authored_order() {
    for closed in [false, true] {
        let mut history = polyline_history();
        let token = history.begin("Polyline").unwrap();
        let report = history
            .edit(&token, |snapshot, ids| {
                snapshot.create_sketch_polyline(DocumentSketchId(1), &input(closed), ids)
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        assert_eq!(report.points, [1, 2, 3].map(SketchEntityId));
        let expected = if closed { vec![4, 5, 6] } else { vec![4, 5] };
        assert_eq!(
            report.lines,
            expected.into_iter().map(SketchEntityId).collect::<Vec<_>>()
        );
        verify_chain(history.snapshot(), closed);
    }
}
fn verify_chain(snapshot: &DocumentSnapshot, closed: bool) {
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let mut expected = vec![[1, 2], [2, 3]];
    if closed {
        expected.push([3, 1]);
    }
    assert_eq!(geometry.lines().len(), expected.len());
    for (line, pair) in geometry.lines().iter().zip(expected) {
        assert_eq!(line.endpoints, pair.map(SketchEntityId));
        assert!(geometry.is_construction(line.id).unwrap());
        assert!(!geometry.is_centerline(line.id).unwrap());
    }
    assert_eq!(geometry.entity_ids().len(), geometry.lines().len());
    assert!(geometry.constraints().is_empty());
    assert!(geometry.same_recipe(&Sketch2D::from_recipe(&geometry.recipe()).unwrap()));
}
#[test]
fn bad_polyline_definitions_and_missing_owners_reserve_nothing() {
    let mut bad = input(true);
    bad.points[2][0] = f64::NAN;
    for (owner, request) in [
        (1, bad),
        (99, input(false)),
        (
            1,
            SketchPolylineCreation {
                points: vec![[0.0; 2]; 2],
                ..input(true)
            },
        ),
        (
            1,
            SketchPolylineCreation {
                points: vec![[0.0; 2]],
                ..input(false)
            },
        ),
    ] {
        let mut history = polyline_history();
        let token = history.begin("Invalid").unwrap();
        assert!(
            history
                .edit(&token, |snapshot, ids| snapshot.create_sketch_polyline(
                    DocumentSketchId(owner),
                    &request,
                    ids
                ))
                .is_err()
        );
        assert_eq!(history.sketch_reservations().entity, 0);
        assert!(
            history
                .snapshot()
                .sketches
                .get(DocumentSketchId(1))
                .unwrap()
                .geometry()
                .points()
                .is_empty()
        );
    }
}
#[test]
fn exhaustion_during_points_or_lines_keeps_the_complete_original_snapshot() {
    for remaining in 0..6 {
        let mut history = polyline_history()
            .with_sketch_reservations(SketchReservationMarks {
                sketch: 1,
                entity: u64::MAX - remaining,
                ..Default::default()
            })
            .unwrap();
        let before = history.snapshot().sketches.clone();
        let token = history.begin("Exhaust").unwrap();
        let error = history
            .edit(&token, |snapshot, ids| {
                snapshot.create_sketch_polyline(DocumentSketchId(1), &input(true), ids)
            })
            .unwrap_err();
        let EditError::Mutation(cause) = error else {
            panic!("expected allocator mutation failure")
        };
        assert_eq!(
            cause,
            SketchPrimitiveCreationError::Identity(RecipeIdError::Exhausted)
        );
        assert_eq!(
            cause.source().unwrap().downcast_ref::<RecipeIdError>(),
            Some(&RecipeIdError::Exhausted)
        );
        assert!(before.same_recipe(&history.snapshot().sketches));
        assert_eq!(history.sketch_reservations().entity, u64::MAX);
    }
}
#[test]
fn coincident_positions_do_not_merge_distinct_authored_vertex_identities() {
    let mut history = polyline_history();
    let token = history.begin("Repeated vertex").unwrap();
    let request = SketchPolylineCreation {
        points: vec![[0.0; 2]; 3],
        ..input(true)
    };
    history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_polyline(DocumentSketchId(1), &request, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    verify_chain(history.snapshot(), true);
}
