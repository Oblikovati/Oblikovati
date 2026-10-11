// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, PlacedSketchHealth, SketchEvaluation},
    history::{SketchDimensionCreation, SketchDimensionCreationReport},
    sketch::{
        MeasuredDimensionOperands, Sketch2D, SketchDimensionalConstraintCreation, SketchEntityId,
        SketchPlane,
    },
};
fn creation_session(veto: Option<&'static str>) -> DocumentSession<RecordingObserver> {
    let history = DocumentEditHistory::new(
        circle_snapshot(),
        RecipeIdAllocator::restore(Some(1), &[1]).unwrap(),
    )
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
fn circle_snapshot() -> DocumentSnapshot {
    let mut snapshot = recipe();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry
        .add_circle(SketchEntityId(2), SketchEntityId(1), 2.0)
        .unwrap();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Circle", SketchPlane::xy(), geometry)
        .unwrap();
    snapshot
}
fn input(source: &str, reference: bool) -> SketchDimensionCreation {
    SketchDimensionCreation {
        operands: SketchDimensionalConstraintCreation::Measured(
            MeasuredDimensionOperands::Radius {
                curve: SketchEntityId(2),
            },
        ),
        expression: source.into(),
        reference,
        text_point: Some([3.0, 4.0]),
    }
}
fn create(
    session: &mut DocumentSession<RecordingObserver>,
    source: &str,
) -> SketchDimensionCreationReport {
    session
        .create_sketch_dimension(DocumentSketchId(1), &input(source, false))
        .unwrap()
}
fn assert_absent(session: &DocumentSession<RecordingObserver>) {
    assert_eq!(session.snapshot().parameters.ids(), [1]);
    let geometry = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(geometry.dimensional_constraints().count(), 0);
    assert_eq!(geometry.circle(SketchEntityId(2)).unwrap().radius, 2.0);
}
#[test]
fn owned_source_and_dimension_are_one_undo_step_and_restore_together() {
    let mut session = creation_session(None);
    let report = create(&mut session, "3 cm");
    assert!((report.measured.value - 3.0).abs() < 1e-9);
    assert_eq!(report.degrees_of_freedom, 2);
    assert_eq!(session.undo_labels(), ["Create sketch dimension"]);
    session.undo().unwrap();
    assert_absent(&session);
    session.redo().unwrap();
    assert_restored_creation(&session, &report);
}
fn assert_restored_creation(
    session: &DocumentSession<RecordingObserver>,
    report: &SketchDimensionCreationReport,
) {
    assert_eq!(
        session
            .snapshot()
            .parameters
            .get(report.parameter)
            .unwrap()
            .source,
        "3 cm"
    );
    let geometry = creation_geometry(session);
    assert_eq!(
        geometry.dimension_text_point(report.constraint).unwrap(),
        Some([3.0, 4.0])
    );
}
#[test]
fn rejected_creation_replies_publish_nothing_and_permanently_consume_attempted_names() {
    let mut session = creation_session(None);
    let generation = session.generation();
    reject_creation_reply(&mut session);
    assert_eq!(session.generation(), generation);
    assert_absent(&session);
    let next = create(&mut session, "3 cm");
    assert_eq!(
        (
            next.parameter,
            next.constraint.0,
            next.parameter_name.as_str()
        ),
        (3, 2, "d1")
    );
}
fn reject_creation_reply(session: &mut DocumentSession<RecordingObserver>) {
    let result = session.create_sketch_dimension_projected(
        DocumentSketchId(1),
        &input("3 cm", false),
        |result| -> Result<(), String> {
            assert_eq!(result.unwrap().parameter_name, "d0");
            Err("reply refused".into())
        },
    );
    assert!(result.is_err());
}
#[test]
fn creation_errors_reach_the_projected_reply_before_any_candidate_is_adopted() {
    let mut session = creation_session(None);
    let reply = session
        .create_sketch_dimension_projected(
            DocumentSketchId(1),
            &input("2 *", false),
            |result| -> Result<String, String> { Ok(result.unwrap_err().to_string()) },
        )
        .unwrap();
    assert!(reply.contains("dimension creation"));
    assert_absent(&session);
}
#[test]
fn group_abort_and_deletion_do_not_reuse_parameter_constraint_or_name_reservations() {
    let mut session = creation_session(None);
    session.begin("Group").unwrap();
    create(&mut session, "3 cm");
    session.abort().unwrap();
    assert_absent(&session);
    let created = create(&mut session, "4 cm");
    assert_eq!(created.parameter_name, "d1");
    session
        .delete_sketch_dimension(DocumentSketchId(1), created.constraint)
        .unwrap();
    assert_recreated_identity(&create(&mut session, "5 cm"));
}
fn assert_recreated_identity(next: &SketchDimensionCreationReport) {
    assert_eq!(
        (
            next.parameter,
            next.constraint.0,
            next.parameter_name.as_str()
        ),
        (4, 3, "d2")
    );
}
#[test]
fn commit_veto_restores_geometry_and_source_but_consumes_creation_reservations() {
    let mut session = creation_session(Some(EVENT_TRANSACTION_COMMITTED));
    for name in ["d0", "d1"] {
        let result = session.create_sketch_dimension_projected(
            DocumentSketchId(1),
            &input("3 cm", false),
            |result| -> Result<(), String> {
                assert_eq!(result.unwrap().parameter_name, name);
                Ok(())
            },
        );
        assert!(result.is_err());
        assert_absent(&session);
    }
    assert!(session.undo_labels().is_empty());
}
#[test]
fn reference_creation_ignores_failed_unused_source_and_preserves_current_geometry() {
    let mut session = creation_session(None);
    let report = session
        .create_sketch_dimension(DocumentSketchId(1), &input("futureRadius", true))
        .unwrap();
    assert_eq!(report.measured.value, 2.0);
    assert_eq!(report.degrees_of_freedom, 3);
    assert!(report.unavailable_targets.is_empty());
    assert_current_health(&session, false);
}
#[test]
fn failed_creation_source_is_explicit_and_auto_recompute_repairs_it_after_binding() {
    let mut session = creation_session(None);
    let report = create(&mut session, "futureRadius");
    assert_eq!(report.measured.value, 2.0);
    assert_eq!(report.degrees_of_freedom, 3);
    assert_eq!(report.unavailable_targets.len(), 1);
    assert_current_health(&session, true);
    bind_radius(&mut session, report.parameter);
    assert_current_health(&session, false);
}
fn bind_radius(session: &mut DocumentSession<RecordingObserver>, parameter: u64) {
    session
        .run("Bind future radius", |snapshot, ids| {
            snapshot.parameters.insert_expression(
                ids.allocate().unwrap(),
                "futureRadius",
                USER_PARAM,
                "4 cm",
            )
        })
        .unwrap();
    let target = session
        .snapshot()
        .parameters
        .model_value(parameter)
        .unwrap();
    assert_eq!(target.value, 4.0);
}

fn assert_current_health(session: &DocumentSession<RecordingObserver>, failed: bool) {
    let placed = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap();
    match placed.health(&session.snapshot().parameters) {
        PlacedSketchHealth::Current(SketchEvaluation::Failed(_)) => assert!(failed),
        PlacedSketchHealth::Current(SketchEvaluation::Numerical(diagnostics)) => {
            assert!(!failed);
            assert_eq!(
                diagnostics.termination,
                oblikovati_solve::nonlinear::SolveTermination::Converged
            );
        }
        other => panic!("{other:?}: expected current source-aware creation health"),
    }
}

fn creation_geometry(session: &DocumentSession<RecordingObserver>) -> &Sketch2D {
    session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
}
