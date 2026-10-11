// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{
        DocumentSketchId, SketchAxisReference, SketchOrientationDefinition, SketchPlacement,
    },
    feature::{
        AuthoredFeature, FeatureId,
        work_geometry::{WorkGeometryDefinition, WorkPointReference},
    },
    history::{DocumentSketchCreation, DocumentSketchCreationError, SketchReservationMarks},
    sketch::{SketchFrameAxis, SketchPlane},
};
use oblikovati_model::{identity::RecipeIdError, sketch::SketchError};
use std::error::Error;

fn request(project_origin: bool) -> DocumentSketchCreation {
    DocumentSketchCreation {
        name: "Created".into(),
        placement: SketchPlacement::Fixed(SketchPlane::xy()),
        orientation: None,
        project_origin,
    }
}
fn frame_origin() -> SketchOrientationDefinition {
    SketchOrientationDefinition {
        reference: SketchAxisReference::OriginX,
        sketch_axis: SketchFrameAxis::Y,
        reverse: true,
        origin: Some([7.0, 11.0, 13.0]),
    }
}
fn assert_anchor(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    expected: [f64; 2],
) {
    let view = session.snapshot().require_resolved_sketch(sketch).unwrap();
    let binding = view.sketch.point_projections()[0];
    assert_eq!(binding.reference, WorkPointReference::Origin);
    assert_eq!(
        view.sketch
            .geometry()
            .point(binding.anchor)
            .unwrap()
            .position,
        expected
    );
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
}
#[test]
fn creation_is_one_history_step_with_an_exact_associative_origin() {
    let mut session = session(None);
    let sketch = session.create_sketch(&request(true)).unwrap();
    assert_anchor(&session, sketch, [0.0; 2]);
    assert_eq!(session.undo_labels(), ["Create sketch"]);
    session.undo().unwrap();
    assert!(session.snapshot().sketches.get(sketch).is_err());
    session.redo().unwrap();
    assert_anchor(&session, sketch, [0.0; 2]);
}
#[test]
fn disabling_origin_projection_creates_an_empty_checked_sketch() {
    let mut session = session(None);
    let sketch = session.create_sketch(&request(false)).unwrap();
    let placed = session.snapshot().sketches.get(sketch).unwrap();
    assert!(placed.point_projections().is_empty());
    assert!(placed.geometry().points().is_empty());
    assert_eq!(
        session.snapshot().resolve_sketch_plane(sketch).unwrap(),
        SketchPlane::xy()
    );
}
#[test]
fn orientation_is_resolved_before_the_origin_anchor_is_created() {
    let mut session = session(None);
    let mut request = request(true);
    request.orientation = Some(frame_origin());
    let sketch = session.create_sketch(&request).unwrap();
    assert_anchor(&session, sketch, [-11.0, 7.0]);
    assert_eq!(
        session
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .orientation(),
        request.orientation
    );
}
fn host_session() -> (DocumentSession<RecordingObserver>, FeatureId) {
    let mut session = session(None);
    let host = session
        .run("Host plane", |snapshot, ids| {
            let host = ids.allocate_feature().unwrap();
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(
                    host,
                    "Host",
                    WorkGeometryDefinition::Fixed(SketchPlane::xz()),
                ))
                .unwrap();
            Ok::<_, String>(host)
        })
        .unwrap();
    (session, host)
}
#[test]
fn creation_keeps_the_owned_host_and_reprojects_after_its_frame_changes() {
    let (mut session, host) = host_session();
    let mut request = request(true);
    request.placement = SketchPlacement::WorkPlane(host);
    let sketch = session.create_sketch(&request).unwrap();
    assert_anchor(&session, sketch, [0.0; 2]);
    change_host(&mut session, host);
    assert_anchor(&session, sketch, [-3.0, -5.0]);
    assert_eq!(
        session.snapshot().sketches.get(sketch).unwrap().placement(),
        request.placement
    );
}
fn change_host(session: &mut DocumentSession<RecordingObserver>, host: FeatureId) {
    session
        .run("Move host", |snapshot, _| {
            let frame = SketchPlane::new(
                [3.0, 7.0, 5.0],
                [1.0, 0.0, 0.0],
                [0.0, 0.0, 1.0],
                Default::default(),
            )
            .unwrap();
            let definition = WorkGeometryDefinition::Fixed(frame);
            snapshot
                .work_geometry
                .replace(host, definition, definition.reads())
        })
        .unwrap();
}
fn assert_current_failure(
    session: &mut DocumentSession<RecordingObserver>,
    request: &DocumentSketchCreation,
) {
    let before = session.snapshot().sketches.clone();
    let labels = session
        .undo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect::<Vec<_>>();
    let error = session.create_sketch(request).unwrap_err();
    assert!(error.to_string().contains("sketch creation"));
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    assert_eq!(session.undo_labels(), labels);
}
#[test]
fn missing_host_is_refused_even_when_origin_projection_is_disabled() {
    let mut session = session(None);
    let mut input = request(false);
    input.placement = SketchPlacement::WorkPlane(FeatureId(42));
    assert_current_failure(&mut session, &input);
    let sketch = session.create_sketch(&request(true)).unwrap();
    assert_eq!(sketch, DocumentSketchId(2));
}
#[test]
fn degenerate_orientation_is_refused_without_a_base_frame_fallback() {
    let mut session = session(None);
    let mut input = request(true);
    input.orientation = Some(SketchOrientationDefinition {
        reference: SketchAxisReference::OriginZ,
        ..frame_origin()
    });
    assert_current_failure(&mut session, &input);
    assert_eq!(
        session.create_sketch(&request(true)).unwrap(),
        DocumentSketchId(2)
    );
}
#[test]
fn invalid_orientation_shape_and_zero_host_are_atomic_sketch_declines() {
    let mut session = session(None);
    let mut input = request(true);
    input.orientation = Some(SketchOrientationDefinition {
        origin: Some([f64::NAN; 3]),
        ..frame_origin()
    });
    assert_current_failure(&mut session, &input);
    input.orientation = None;
    input.placement = SketchPlacement::WorkPlane(FeatureId(0));
    assert_current_failure(&mut session, &input);
    assert_eq!(
        session.create_sketch(&request(true)).unwrap(),
        DocumentSketchId(3)
    );
}
#[test]
fn rejected_creation_reply_preserves_state_but_does_not_reuse_reserved_ids() {
    let mut session = session(None);
    let before = session.snapshot().sketches.clone();
    let result = session.create_sketch_projected(&request(true), |created| {
        assert_eq!(created.unwrap(), DocumentSketchId(1));
        Err::<(), _>("reply refused")
    });
    assert!(result.is_err());
    assert!(before.same_evaluated_state(&session.snapshot().sketches));
    let sketch = session.create_sketch(&request(true)).unwrap();
    let anchor = session
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .point_projections()[0]
        .anchor;
    assert_eq!(sketch.0, 2);
    assert_eq!(anchor.0, 2);
}
#[test]
fn aborted_creation_reserves_both_ids_and_drops_the_whole_candidate() {
    let mut session = session(None);
    session.begin("Batch").unwrap();
    session.create_sketch(&request(true)).unwrap();
    session.abort().unwrap();
    assert!(session.snapshot().sketches.sketch_ids().is_empty());
    assert!(session.undo_labels().is_empty());
    let sketch = session.create_sketch(&request(true)).unwrap();
    assert_eq!(sketch.0, 2);
    assert_created_anchor_id(&session, sketch, 2);
}
fn exhausted_session(marks: SketchReservationMarks) -> DocumentSession<RecordingObserver> {
    let ids = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let history = DocumentEditHistory::new(recipe(), ids)
        .unwrap()
        .with_sketch_reservations(marks)
        .unwrap();
    DocumentSession::new(17, history, RecordingObserver::default())
}
#[test]
fn exhausted_sketch_or_anchor_namespace_never_publishes_a_partial_sketch() {
    for marks in [
        SketchReservationMarks {
            sketch: u64::MAX,
            entity: 0,
            constraints: Default::default(),
        },
        SketchReservationMarks {
            sketch: 0,
            entity: u64::MAX,
            constraints: Default::default(),
        },
    ] {
        let mut session = exhausted_session(marks);
        let error = session.create_sketch(&request(true)).unwrap_err();
        assert!(error.to_string().contains("sketch creation identity"));
        assert!(session.snapshot().sketches.sketch_ids().is_empty());
        assert!(session.undo_labels().is_empty());
    }
}
#[test]
fn typed_creation_error_keeps_its_cause_through_the_session_boundary() {
    let mut session = session(None);
    let mut input = request(true);
    input.placement = SketchPlacement::WorkPlane(FeatureId(42));
    session
        .create_sketch_projected(&input, |result| {
            let error = result.unwrap_err();
            assert!(matches!(error, DocumentSketchCreationError::Placement(_)));
            assert!(error.source().is_some());
            assert!(error.to_string().contains("FeatureId(42)"));
            Ok::<_, String>(())
        })
        .unwrap();
    assert!(session.snapshot().sketches.sketch_ids().is_empty());
}

fn assert_created_anchor_id(
    session: &DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
    expected: u64,
) {
    assert_eq!(
        session
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .points()[0]
            .id
            .0,
        expected
    );
}
#[test]
fn sketch_and_identity_creation_errors_keep_typed_sources_and_messages() {
    let errors = [
        DocumentSketchCreationError::Identity(RecipeIdError::Exhausted),
        DocumentSketchCreationError::Sketch(SketchError::Definition(
            "invalid origin NaN: expected finite coordinates".into(),
        )),
    ];
    let identity_cause = errors[0].source().unwrap();
    let sketch_cause = errors[1].source().unwrap();
    assert!(identity_cause.downcast_ref::<RecipeIdError>().is_some());
    assert!(sketch_cause.downcast_ref::<SketchError>().is_some());
    for error in errors {
        assert!(error.to_string().contains("sketch creation"));
    }
}
#[test]
fn repeated_creations_reserve_distinct_sketches_and_origin_anchor_identities() {
    let mut session = session(None);
    let first = session.create_sketch(&request(true)).unwrap();
    let second = session.create_sketch(&request(true)).unwrap();
    assert_ne!(first, second);
    assert_created_anchor_id(&session, first, 1);
    assert_created_anchor_id(&session, second, 2);
    assert_anchor(&session, first, [0.0; 2]);
    assert_anchor(&session, second, [0.0; 2]);
    assert_eq!(session.undo_labels(), ["Create sketch", "Create sketch"]);
}
