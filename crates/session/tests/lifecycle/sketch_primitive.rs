// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    history::{DocumentSketchCreation, SketchPrimitiveCreationError, SketchReservationMarks},
    sketch::{Sketch2D, SketchEntityId, SketchPlane, SketchPointInput, SketchPrimitiveCreation},
};
use std::error::Error;
fn empty_sketch(
    session: &mut DocumentSession<RecordingObserver>,
    origin: bool,
) -> DocumentSketchId {
    session
        .create_sketch(&DocumentSketchCreation {
            name: "Authored".into(),
            placement: SketchPlacement::Fixed(SketchPlane::xy()),
            orientation: None,
            project_origin: origin,
        })
        .unwrap()
}
fn point(position: [f64; 2]) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Point { position }
}
fn line(start: SketchPointInput) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Line {
        endpoints: [start, SketchPointInput::New([3.0, 4.0])],
        construction: false,
        centerline: false,
    }
}
fn circle(radius: f64, construction: bool) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Circle {
        center: SketchPointInput::New([2.0, 3.0]),
        radius,
        construction,
    }
}
fn arc(construction: bool, counter_clockwise: bool) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Arc {
        center: SketchPointInput::New([0.0; 2]),
        endpoints: [
            SketchPointInput::New([2.0, 0.0]),
            SketchPointInput::New([0.0, 2.0]),
        ],
        counter_clockwise,
        construction,
    }
}
fn geometry(session: &DocumentSession<RecordingObserver>, sketch: DocumentSketchId) -> &Sketch2D {
    session.snapshot().sketches.get(sketch).unwrap().geometry()
}
#[test]
fn all_primitives_reserve_ordered_owned_entities_and_current_solver_evidence() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    let inputs = primitive_fixture_inputs();
    let expected = [vec![1], vec![2, 3, 4], vec![5, 6], vec![7, 8, 9, 10]];
    for (input, ids) in inputs.iter().zip(expected) {
        assert_created_primitive(&mut session, sketch, input, ids);
    }
    assert_eq!(
        geometry(&session, sketch).entity_ids(),
        (1..=10).map(SketchEntityId).collect::<Vec<_>>()
    );
    assert_eq!(
        session.undo_labels().last().copied().unwrap(),
        "Create sketch primitive"
    );
}
#[test]
fn shared_points_keep_identity_and_the_associative_anchor_remains_grounded() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let report = session
        .create_sketch_primitive(sketch, &line(SketchPointInput::Owned(SketchEntityId(1))))
        .unwrap();
    assert_eq!(report.points, [SketchEntityId(1), SketchEntityId(2)]);
    assert_eq!(report.created, [SketchEntityId(2), SketchEntityId(3)]);
    assert_eq!(geometry(&session, sketch).points().len(), 2);
    assert_eq!(
        geometry(&session, sketch)
            .line(report.entity)
            .unwrap()
            .endpoints,
        [SketchEntityId(1), SketchEntityId(2)]
    );
    let placed = session.snapshot().sketches.get(sketch).unwrap();
    assert_eq!(placed.point_projections().len(), 1);
    assert_numeric_verification(&session, sketch, 2);
}
#[test]
fn fully_shared_curves_allocate_only_their_own_identity() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    session
        .create_sketch_primitive(sketch, &arc(false, false))
        .unwrap();
    let request = SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::Owned(SketchEntityId(2)),
            SketchPointInput::Owned(SketchEntityId(3)),
        ],
        construction: true,
        centerline: false,
    };
    let report = session.create_sketch_primitive(sketch, &request).unwrap();
    assert_shared_report(&report);
    assert_flagged_entity(&session, sketch, report.entity);
    assert!(!geometry(&session, sketch).arcs()[0].counter_clockwise);
}
#[test]
fn curve_roles_survive_creation_solver_candidates_undo_and_redo() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    let report = session
        .create_sketch_primitive(sketch, &circle(2.0, true))
        .unwrap();
    let saved = geometry(&session, sketch).recipe();
    assert!(geometry(&session, sketch).profile_curve_ids().is_empty());
    assert_flagged_entity(&session, sketch, report.entity);
    session.undo().unwrap();
    assert!(geometry(&session, sketch).entity_ids().is_empty());
    session.redo().unwrap();
    assert_eq!(geometry(&session, sketch).recipe(), saved);
    assert_numeric_verification(&session, sketch, 3);
}
#[test]
fn existing_geometry_constraints_and_flags_are_retained_when_appending() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let created = session
        .create_sketch_primitive(sketch, &circle(2.0, true))
        .unwrap();
    bind_radius(&mut session, sketch, created.entity);
    let before = geometry(&session, sketch).recipe();
    session
        .create_sketch_primitive(sketch, &arc(true, true))
        .unwrap();
    let after = geometry(&session, sketch).recipe();
    assert_eq!(after.circles, before.circles);
    assert_eq!(&after.points[..before.points.len()], before.points);
    assert_eq!(after.constraints, before.constraints);
    assert_construction_prefix(&after, &before);
}
fn assert_refused(
    session: &mut DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    input: &SketchPrimitiveCreation,
) {
    let original = session.snapshot().sketches.recipe();
    let generation = session.generation();
    let labels: Vec<String> = session
        .undo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect();
    let cause = session
        .create_sketch_primitive(sketch, input)
        .unwrap_err()
        .to_string();
    assert!(cause.contains("expected"), "{cause}");
    assert_eq!(session.snapshot().sketches.recipe(), original);
    assert_eq!(session.generation(), generation);
    assert_eq!(session.undo_labels(), labels);
}
#[test]
fn invalid_coordinates_radius_owner_and_point_references_decline_before_reserving() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    refuse_invalid_numeric_inputs(&mut session, sketch);
    assert_refused(
        &mut session,
        sketch,
        &line(SketchPointInput::Owned(SketchEntityId(99))),
    );
    assert_refused(&mut session, DocumentSketchId(99), &point([0.0; 2]));
    assert_next_point(&mut session, sketch, 1);
}
#[test]
fn foreign_points_and_owned_curve_ids_cannot_be_used_as_defining_points() {
    let mut session = session(None);
    let first = empty_sketch(&mut session, false);
    let second = empty_sketch(&mut session, false);
    let report = session
        .create_sketch_primitive(first, &circle(2.0, false))
        .unwrap();
    assert_refused(
        &mut session,
        second,
        &line(SketchPointInput::Owned(report.points[0])),
    );
    assert_refused(
        &mut session,
        first,
        &line(SketchPointInput::Owned(report.entity)),
    );
}
#[test]
fn response_failure_discards_geometry_but_does_not_reuse_reserved_ids() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    let before = session.generation();
    let cause = session
        .create_sketch_primitive_projected(sketch, &circle(2.0, false), |result| {
            assert_eq!(result.unwrap().entity, SketchEntityId(2));
            Err::<(), _>("reply refused")
        })
        .unwrap_err();
    assert!(cause.to_string().contains("reply refused"));
    assert!(geometry(&session, sketch).entity_ids().is_empty());
    assert_eq!(session.generation(), before);
    assert_next_point(&mut session, sketch, 3);
}
#[test]
fn vetoed_creation_keeps_geometry_and_permanent_reservations() {
    let mut session = prepared_session(Some(EVENT_TRANSACTION_COMMITTED), 0);
    let sketch = DocumentSketchId(1);
    assert!(
        session
            .create_sketch_primitive(sketch, &arc(true, true))
            .is_err()
    );
    assert!(geometry(&session, sketch).entity_ids().is_empty());
    assert!(
        session
            .create_sketch_primitive_projected(sketch, &point([0.0; 2]), |result| {
                assert_eq!(result.as_ref().unwrap().entity, SketchEntityId(5));
                result
            })
            .is_err()
    );
    assert!(session.undo_labels().is_empty());
}
#[test]
fn abort_and_undo_never_reuse_reserved_geometry_identities() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    session.begin("Geometry batch").unwrap();
    session
        .create_sketch_primitive(sketch, &circle(2.0, true))
        .unwrap();
    session.abort().unwrap();
    assert!(geometry(&session, sketch).entity_ids().is_empty());
    let report = session
        .create_sketch_primitive(sketch, &point([0.0; 2]))
        .unwrap();
    assert_eq!(report.entity, SketchEntityId(3));
    session.undo().unwrap();
    assert_next_point(&mut session, sketch, 4);
}
fn exhausted_session(mark: u64) -> DocumentSession<RecordingObserver> {
    prepared_session(None, mark)
}
#[test]
fn mid_primitive_identity_exhaustion_never_publishes_partial_points() {
    let mut session = exhausted_session(u64::MAX - 1);
    let sketch = DocumentSketchId(1);
    let error = session
        .create_sketch_primitive(sketch, &circle(2.0, true))
        .unwrap_err();
    assert!(error.to_string().contains("identity"));
    assert!(geometry(&session, sketch).points().is_empty());
    assert!(
        session
            .create_sketch_primitive(sketch, &point([0.0; 2]))
            .is_err()
    );
    assert!(session.undo_labels().is_empty());
}
#[test]
fn typed_creation_declines_keep_the_original_cause_for_reply_preparation() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    session
        .create_sketch_primitive_projected(sketch, &circle(-1.0, true), |result| {
            let error = result.unwrap_err();
            assert_sketch_error_source(&error);
            Err::<(), _>(error)
        })
        .unwrap_err();
    let error = identity_failure();
    assert!(
        error
            .source()
            .unwrap()
            .downcast_ref::<oblikovati_model::identity::RecipeIdError>()
            .is_some()
    );
}

fn assert_created_primitive(
    session: &mut DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    input: &SketchPrimitiveCreation,
    ids: Vec<u64>,
) {
    let report = session.create_sketch_primitive(sketch, input).unwrap();
    assert_eq!(
        report.created,
        ids.into_iter().map(SketchEntityId).collect::<Vec<_>>()
    );
    assert_eq!(report.entity, *report.created.last().unwrap());
    assert_eq!(report.sketch, sketch);
    let freedom = session.analyze_sketch(sketch).unwrap().degrees_of_freedom;
    assert_numeric_verification(session, sketch, freedom);
}
fn assert_shared_report(report: &oblikovati_model::history::SketchPrimitiveCreationReport) {
    assert_eq!(report.created, [SketchEntityId(5)]);
    assert_eq!(report.points, [SketchEntityId(2), SketchEntityId(3)]);
}
fn assert_numeric_verification(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    freedom: usize,
) {
    use oblikovati_model::{
        document_sketch::SketchReadinessCause, history::DocumentSketchPlacementError,
    };
    let error = session
        .snapshot()
        .require_resolved_sketch(sketch)
        .unwrap_err();
    match error {
        DocumentSketchPlacementError::Verification(cause) => {
            assert_eq!(cause.cause, SketchReadinessCause::FreeVariables(freedom))
        }
        unexpected => {
            panic!("verification {unexpected:?}: expected current converged free geometry")
        }
    }
}

fn refuse_invalid_numeric_inputs(
    session: &mut DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
) {
    for bad in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        assert_refused(session, sketch, &point([bad, 1.0]));
        assert_refused(session, sketch, &line(SketchPointInput::New([1.0, bad])));
        assert_refused(session, sketch, &circle(bad, false));
    }
    for bad in [0.0, -2.0] {
        assert_refused(session, sketch, &circle(bad, false));
    }
}
fn assert_next_point(
    session: &mut DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    expected: u64,
) {
    assert_eq!(
        session
            .create_sketch_primitive(sketch, &point([0.0; 2]))
            .unwrap()
            .entity,
        SketchEntityId(expected)
    );
}

fn prepared_session(veto: Option<&'static str>, mark: u64) -> DocumentSession<RecordingObserver> {
    let history = DocumentEditHistory::new(
        prepared_snapshot(),
        RecipeIdAllocator::restore(Some(1), &[1]).unwrap(),
    )
    .unwrap()
    .with_sketch_reservations(SketchReservationMarks {
        sketch: 1,
        entity: mark,
        constraints: Default::default(),
    })
    .unwrap();
    DocumentSession::new(
        17,
        history,
        RecordingObserver {
            veto,
            ..Default::default()
        },
    )
}

fn assert_sketch_error_source(error: &SketchPrimitiveCreationError) {
    assert!(matches!(error, SketchPrimitiveCreationError::Sketch(_)));
    assert!(
        error
            .source()
            .unwrap()
            .downcast_ref::<oblikovati_model::sketch::SketchError>()
            .is_some()
    );
}

fn prepared_snapshot() -> DocumentSnapshot {
    let mut snapshot = recipe();
    snapshot
        .sketches
        .add(
            DocumentSketchId(1),
            "Authored",
            SketchPlane::xy(),
            Sketch2D::default(),
        )
        .unwrap();
    snapshot
}
fn primitive_fixture_inputs() -> [SketchPrimitiveCreation; 4] {
    [
        point([1.0, 2.0]),
        line(SketchPointInput::New([0.0; 2])),
        circle(2.0, false),
        arc(false, true),
    ]
}
fn assert_flagged_entity(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    entity: SketchEntityId,
) {
    assert!(geometry(session, sketch).is_construction(entity).unwrap());
}
fn assert_construction_prefix(
    after: &oblikovati_model::sketch::SketchRecipe,
    before: &oblikovati_model::sketch::SketchRecipe,
) {
    assert!(
        after
            .construction_curves
            .as_ref()
            .unwrap()
            .starts_with(before.construction_curves.as_ref().unwrap())
    );
}
fn identity_failure() -> SketchPrimitiveCreationError {
    SketchPrimitiveCreationError::Identity(oblikovati_model::identity::RecipeIdError::Exhausted)
}

fn bind_radius(
    session: &mut DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    curve: SketchEntityId,
) {
    use oblikovati_model::sketch::PlanarConstraint;
    session
        .run("Bind radius", |snapshot, ids| {
            let mut geometry = snapshot.sketches.get(sketch)?.geometry().clone();
            geometry.add_constraint(
                ids.allocate_sketch_constraint(sketch).unwrap(),
                PlanarConstraint::Radius {
                    curve,
                    parameter: 1,
                },
            )?;
            snapshot.sketches.replace_geometry(sketch, geometry)
        })
        .unwrap();
}

#[path = "sketch_primitive/failures.rs"]
mod failures;

#[path = "sketch_primitive/inferred_line.rs"]
mod inferred_line;
