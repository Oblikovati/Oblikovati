// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{SketchLineChainCreation, SketchLineCreationError},
    sketch::{PlanarConstraint, SketchConstraintId},
};
fn request() -> SketchLineChainCreation {
    SketchLineChainCreation {
        points: vec![[0.0001, 0.0001], [4.0, 0.01], [4.01, 3.0]],
        closed: false,
        construction: false,
        centerline: false,
    }
}
#[test]
fn preview_and_commit_have_identical_solved_geometry_members_and_inference() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let before = session.snapshot().sketches.clone();
    let marks = session.history().sketch_reservations();
    let generation = session.generation();
    let events = session.observer().observations.len();
    let preview = session
        .preview_inferred_sketch_line_chain(sketch, &request())
        .unwrap();
    assert_eq!(session.snapshot().sketches, before);
    assert_eq!(session.history().sketch_reservations(), marks);
    assert_eq!(session.generation(), generation);
    assert_eq!(session.observer().observations.len(), events);
    assert_eq!(
        preview,
        session
            .preview_inferred_sketch_line_chain(sketch, &request())
            .unwrap()
    );
    let report = session
        .create_inferred_sketch_line_chain(sketch, &request())
        .unwrap();
    assert_eq!(preview.report, report);
    assert_eq!(&preview.geometry, geometry(&session, sketch));
    assert_eq!(
        preview.members,
        [
            report.points[0],
            report.points[1],
            report.lines[0].primitive.entity,
            report.points[2],
            report.lines[1].primitive.entity
        ]
    );
    assert!(!preview.members.contains(&SketchEntityId(1)));
    assert_numeric_verification(&session, sketch, 2);
}
#[test]
fn preview_preserves_an_open_group_and_redo_while_commit_joins_the_group() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    session
        .create_sketch_primitive(sketch, &point([9.0, 9.0]))
        .unwrap();
    session.undo().unwrap();
    let redo = session
        .redo_labels()
        .iter()
        .map(|label| label.to_string())
        .collect::<Vec<_>>();
    session.begin("Continuous Line").unwrap();
    let preview = session
        .preview_inferred_sketch_line_chain(sketch, &request())
        .unwrap();
    assert!(session.in_transaction());
    assert_eq!(session.redo_labels(), redo);
    let report = session
        .create_inferred_sketch_line_chain(sketch, &request())
        .unwrap();
    assert_eq!(preview.report, report);
    session.end().unwrap();
    assert_eq!(session.undo_label(), Some("Continuous Line"));
    session.undo().unwrap();
    assert!(geometry(&session, sketch).lines().is_empty());
    session.redo().unwrap();
    assert_eq!(geometry(&session, sketch), &preview.geometry);
}
fn fixed_endpoints_session() -> DocumentSession<RecordingObserver> {
    let mut snapshot = prepared_snapshot();
    let mut geometry = Sketch2D::default();
    for (index, position) in [[0.0, 0.0], [4.0, 0.01]].into_iter().enumerate() {
        let id = (index + 1) as u64;
        geometry.add_point(SketchEntityId(id), position).unwrap();
        geometry
            .add_constraint(
                SketchConstraintId(id),
                PlanarConstraint::Fix {
                    point: SketchEntityId(id),
                    position,
                },
            )
            .unwrap();
    }
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    DocumentSession::new(17, history, RecordingObserver::default())
}
#[test]
fn conflicting_inference_is_refused_after_solve_in_preview_and_commit() {
    let mut session = fixed_endpoints_session();
    let sketch = DocumentSketchId(1);
    let before = session.snapshot().sketches.clone();
    let marks = session.history().sketch_reservations();
    let mut input = request();
    input.points = vec![[0.0, 0.0], [4.0, 0.01]];
    let preview = session
        .preview_inferred_sketch_line_chain(sketch, &input)
        .unwrap_err();
    assert_unconverged(&preview);
    assert_eq!(session.history().sketch_reservations(), marks);
    let commit = session
        .create_inferred_sketch_line_chain(sketch, &input)
        .unwrap_err();
    assert_unconverged(&commit);
    assert_eq!(session.snapshot().sketches, before);
    assert!(session.history().sketch_reservations().entity > marks.entity);
    assert!(session.undo_labels().is_empty());
}
fn assert_unconverged(error: &CommandError<SketchLineCreationError>) {
    assert!(error.to_string().contains("expected"));
    let CommandError::Edit(EditError::Mutation(MutationError::Command(
        SketchLineCreationError::Readiness(error),
    ))) = error
    else {
        panic!("expected typed post-solve decline: {error}");
    };
    assert!(matches!(
        error.cause,
        oblikovati_model::document_sketch::SketchReadinessCause::Unconverged(_)
    ));
    assert!(error.to_string().contains("expected"));
}
#[test]
fn invalid_preview_and_missing_sketch_do_not_change_events_or_reservations() {
    let session = prepared_session(None, 0);
    let mut invalid = request();
    invalid.points[1][0] = f64::NAN;
    assert!(
        session
            .preview_inferred_sketch_line_chain(DocumentSketchId(1), &invalid)
            .is_err()
    );
    assert!(
        session
            .preview_inferred_sketch_line_chain(DocumentSketchId(99), &request())
            .is_err()
    );
    assert_eq!(session.history().sketch_reservations().entity, 0);
    assert!(session.observer().observations.is_empty());
}

#[test]
fn preview_reads_current_parameters_and_inference_preferences_in_an_open_edit() {
    let mut session = driven_reference_session();
    let sketch = DocumentSketchId(1);
    session.begin("Edit source and extend").unwrap();
    session
        .edit(|snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")?;
            let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
            settings.auto_apply_constraints = false;
            snapshot.sketch_settings = Some(settings);
            Ok::<_, oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    let mut input = request();
    input.points[0] = [2.0001, 0.0001];
    let preview = session
        .preview_inferred_sketch_line_chain(sketch, &input)
        .unwrap();
    assert_eq!(
        preview.report.lines[0].inference.points[0].target,
        SketchEntityId(2)
    );
    assert!(
        preview
            .report
            .lines
            .iter()
            .all(|line| line.direction.is_none() && line.point_constraints.is_empty())
    );
    assert_eq!(
        preview
            .geometry
            .point(preview.report.points[0])
            .unwrap()
            .position,
        input.points[0]
    );
    let report = session
        .create_inferred_sketch_line_chain(sketch, &input)
        .unwrap();
    assert_eq!(report, preview.report);
    assert_eq!(geometry(&session, sketch), &preview.geometry);
    session.end().unwrap();
}
fn driven_reference_session() -> DocumentSession<RecordingObserver> {
    let mut snapshot = prepared_snapshot();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry.add_point(SketchEntityId(2), [1.0, 0.0]).unwrap();
    geometry
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    for (index, relation) in [
        PlanarConstraint::Fix {
            point: SketchEntityId(1),
            position: [0.0, 0.0],
        },
        PlanarConstraint::Distance {
            points: [SketchEntityId(1), SketchEntityId(2)],
            parameter: 1,
            measure: oblikovati_model::sketch::DistanceMeasure::Aligned,
        },
        PlanarConstraint::LineHorizontal {
            line: SketchEntityId(3),
        },
    ]
    .into_iter()
    .enumerate()
    {
        geometry
            .add_constraint(SketchConstraintId(index as u64 + 1), relation)
            .unwrap();
    }
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    DocumentSession::new(17, history, RecordingObserver::default())
}
