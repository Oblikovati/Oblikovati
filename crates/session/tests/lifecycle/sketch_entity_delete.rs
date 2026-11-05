// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_dimension_delete::{
    assert_owned_dimension, dimension_session, dimension_session_from_snapshot,
};
use super::*;
use oblikovati_model::feature::{
    AuthoredFeature, FeatureId,
    work_geometry::{DatumLength, WorkGeometryDefinition},
    work_plane::{WorkPlaneOrigin, WorkPlaneReference},
};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPointProjection},
    sketch::*,
};

#[test]
fn entity_dimension_cascade_restores_on_undo_redo_and_abort() {
    let mut session = dimension_session(None);
    let before = session.snapshot().sketches.recipe();
    let report = session
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(2)])
        .unwrap();
    assert_eq!(report.entities, vec![SketchEntityId(2)]);
    assert_eq!(report.constraints.len(), 1);
    assert_owned_dimension(&session, false);
    session.undo().unwrap();
    assert_eq!(session.snapshot().sketches.recipe(), before);
    assert_owned_dimension(&session, true);
    session.redo().unwrap();
    assert_owned_dimension(&session, false);
    session.undo().unwrap();
    assert_entity_group_abort(&mut session, &before);
}
fn assert_entity_group_abort(
    session: &mut DocumentSession<RecordingObserver>,
    before: &oblikovati_model::document_sketch::DocumentSketchRecipe,
) {
    session.begin("Delete group").unwrap();
    session
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(2)])
        .unwrap();
    session.abort().unwrap();
    assert_eq!(&session.snapshot().sketches.recipe(), before);
    assert_owned_dimension(session, true);
}

#[test]
fn rejected_reply_discards_geometry_and_parameter_cleanup() {
    let mut session = dimension_session(None);
    let generation = session.generation();
    let result: Result<(), CommandError<String>> = session.delete_sketch_entities_projected(
        DocumentSketchId(1),
        &[SketchEntityId(2)],
        |reply| {
            assert_eq!(reply.unwrap().constraints.len(), 1);
            Err("reply rejected".into())
        },
    );
    assert!(result.unwrap_err().to_string().contains("reply rejected"));
    assert_owned_dimension(&session, true);
    assert_eq!(session.generation(), generation);
}

#[test]
fn veto_and_unknown_selection_leave_no_history_entry() {
    for veto in [None, Some(EVENT_TRANSACTION_COMMITTED)] {
        let mut session = dimension_session(veto);
        let selection = if veto.is_some() {
            vec![SketchEntityId(2)]
        } else {
            vec![SketchEntityId(2), SketchEntityId(999)]
        };
        assert!(
            session
                .delete_sketch_entities(DocumentSketchId(1), &selection)
                .is_err()
        );
        assert_owned_dimension(&session, true);
        assert!(session.undo_labels().is_empty());
    }
}

#[test]
fn user_source_survives_its_last_dimension() {
    let mut snapshot = dimension_session(None).snapshot().clone();
    bind_user_radius(&mut snapshot);
    snapshot
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(2)])
        .unwrap();
    assert_eq!(snapshot.parameters.get(1).unwrap().kind, USER_PARAM);
    assert!(snapshot.parameters.get(2).is_some());
}

#[test]
fn shared_reference_dimension_preserves_model_source_until_last_owner() {
    let mut snapshot = dimension_session(None).snapshot().clone();
    add_shared_reference(&mut snapshot);
    snapshot
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(2)])
        .unwrap();
    assert!(snapshot.parameters.get(2).is_some());
    snapshot
        .delete_sketch_entities(DocumentSketchId(2), &[SketchEntityId(4)])
        .unwrap();
    assert!(snapshot.parameters.get(2).is_none());
}

#[test]
fn deleting_projection_anchor_detaches_association_and_dependent_curve() {
    let snapshot = projected_snapshot();
    let mut session = dimension_session_from_snapshot(snapshot, None);
    let before = session.snapshot().sketches.recipe();
    let report = session
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(1)])
        .unwrap();
    assert_eq!(report.entities, vec![SketchEntityId(1), SketchEntityId(2)]);
    let placed = session
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap();
    assert!(placed.point_projections().is_empty());
    assert!(placed.geometry().owned_entity_ids().is_empty());
    assert_owned_dimension(&session, false);
    session.undo().unwrap();
    assert_eq!(session.snapshot().sketches.recipe(), before);
}

#[test]
fn retiring_dimension_source_invalidates_authored_formula_reader() {
    let mut snapshot = dimension_session(None).snapshot().clone();
    snapshot
        .parameters
        .insert_expression(3, "dependent", USER_PARAM, "radius * 2")
        .unwrap();
    snapshot
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(2)])
        .unwrap();
    assert!(snapshot.parameters.get(2).is_none());
    assert!(
        snapshot
            .parameters
            .get(3)
            .unwrap()
            .current_value()
            .is_none()
    );
}

#[test]
fn feature_owned_scalar_remains_after_shared_sketch_dimension_is_removed() {
    let mut snapshot = feature_owned_snapshot();
    snapshot
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(2)])
        .unwrap();
    snapshot.validate_scalar_ownership().unwrap();
    assert!(snapshot.parameters.get(2).is_some());
}

#[test]
fn shared_source_survives_same_sketch_then_retires_once() {
    let mut snapshot = dimension_session(None).snapshot().clone();
    add_second_shared_circle(&mut snapshot);
    snapshot
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(2)])
        .unwrap();
    assert!(snapshot.parameters.get(2).is_some());
    snapshot
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(3)])
        .unwrap();
    assert!(snapshot.parameters.get(2).is_none());
}

#[test]
fn failed_dimension_source_can_be_retired_without_healthy_solve() {
    let mut snapshot = dimension_session(None).snapshot().clone();
    snapshot
        .parameters
        .set_expression(2, "missing_length")
        .unwrap();
    let radius = snapshot.parameters.get(2).unwrap();
    assert!(radius.current_value().is_none());
    let mut session = dimension_session_from_snapshot(snapshot, None);
    session
        .delete_sketch_entities(DocumentSketchId(1), &[SketchEntityId(2)])
        .unwrap();
    assert_owned_dimension(&session, false);
    session.undo().unwrap();
    assert_owned_dimension(&session, true);
    assert_failed_radius(&session);
}

#[test]
fn empty_selection_keeps_sources_and_unknown_sketch_refuses_atomically() {
    let mut snapshot = dimension_session(None).snapshot().clone();
    let before = snapshot.sketches.recipe();
    assert_eq!(
        snapshot
            .delete_sketch_entities(DocumentSketchId(1), &[])
            .unwrap(),
        Default::default()
    );
    assert!(
        snapshot
            .delete_sketch_entities(DocumentSketchId(999), &[SketchEntityId(2)])
            .is_err()
    );
    assert_eq!(snapshot.sketches.recipe(), before);
    assert!(snapshot.parameters.get(2).is_some());
}

fn bind_user_radius(snapshot: &mut DocumentSnapshot) {
    let mut geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    geometry
        .remove_dimensional_constraint(SketchConstraintId(1))
        .unwrap();
    bind_radius(&mut geometry, SketchEntityId(2), 1);
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
}

fn add_shared_reference(snapshot: &mut DocumentSnapshot) {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(3), [3.0, 4.0]).unwrap();
    geometry
        .add_circle(SketchEntityId(4), SketchEntityId(3), 2.0)
        .unwrap();
    bind_radius(&mut geometry, SketchEntityId(4), 2);
    geometry
        .set_dimension_reference(SketchConstraintId(2), true)
        .unwrap();
    snapshot
        .sketches
        .add(DocumentSketchId(2), "Shared", SketchPlane::xy(), geometry)
        .unwrap();
}

fn bind_radius(geometry: &mut Sketch2D, curve: SketchEntityId, parameter: u64) {
    geometry
        .add_constraint(
            SketchConstraintId(2),
            PlanarConstraint::Radius { curve, parameter },
        )
        .unwrap();
}

fn projected_snapshot() -> DocumentSnapshot {
    use oblikovati_model::feature::work_geometry::WorkPointReference;
    let mut snapshot = dimension_session(None).snapshot().clone();
    snapshot
        .sketches
        .set_point_projections(
            DocumentSketchId(1),
            vec![SketchPointProjection {
                anchor: SketchEntityId(1),
                reference: WorkPointReference::Origin,
            }],
        )
        .unwrap();
    snapshot
}

fn feature_owned_snapshot() -> DocumentSnapshot {
    let mut snapshot = dimension_session(None).snapshot().clone();
    let definition = WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: DatumLength::Parameter(2),
    };
    snapshot
        .work_geometry
        .append(
            AuthoredFeature::datum(FeatureId(1), "Offset", definition)
                .with_owned_scalar("offset", 2)
                .unwrap(),
        )
        .unwrap();
    snapshot
}

fn add_second_shared_circle(snapshot: &mut DocumentSnapshot) {
    let mut geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    geometry
        .add_circle(SketchEntityId(3), SketchEntityId(1), 1.0)
        .unwrap();
    bind_radius(&mut geometry, SketchEntityId(3), 2);
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
}

fn assert_failed_radius(session: &DocumentSession<RecordingObserver>) {
    assert!(
        session
            .snapshot()
            .parameters
            .get(2)
            .unwrap()
            .current_value()
            .is_none()
    );
}
