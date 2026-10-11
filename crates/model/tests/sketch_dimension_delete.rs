// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_api::types::{MODEL_PARAM, USER_PARAM};
use oblikovati_model::{document_sketch::DocumentSketchId, history::DocumentSnapshot, sketch::*};
fn snapshot(kind: oblikovati_api::types::ParameterKind, source: &str) -> DocumentSnapshot {
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "radius", kind, source)
        .unwrap();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Circle", SketchPlane::xy(), geometry())
        .unwrap();
    snapshot
}
fn geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry
        .add_circle(SketchEntityId(2), SketchEntityId(1), 1.0)
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Radius {
                curve: SketchEntityId(2),
                parameter: 1,
            },
        )
        .unwrap();
    geometry
}
fn delete(snapshot: &mut DocumentSnapshot) -> Result<usize, SketchError> {
    snapshot.delete_sketch_dimension(
        DocumentSketchId(1),
        SketchConstraintId(1),
        Default::default(),
    )
}
#[test]
fn owner_cascade_removes_source_and_invalidates_its_expression_readers() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    snapshot
        .parameters
        .insert_expression(2, "reader", USER_PARAM, "radius * 2")
        .unwrap();
    assert_eq!(delete(&mut snapshot).unwrap(), 3);
    assert!(snapshot.parameters.get(1).is_none());
    let reader = snapshot.parameters.get(2).unwrap();
    assert!(reader.current_value().is_none());
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert!(geometry.constraints().is_empty());
}
#[test]
fn failed_parameter_can_be_removed_to_recover_geometry() {
    let mut snapshot = snapshot(MODEL_PARAM, "missing");
    assert_eq!(delete(&mut snapshot).unwrap(), 3);
    assert!(snapshot.parameters.get(1).is_none());
}
#[test]
fn user_and_missing_parameters_cannot_be_claimed_by_dimension_deletion() {
    for absent in [false, true] {
        let mut snapshot = snapshot(USER_PARAM, "1 cm");
        if absent {
            snapshot.parameters.remove_for_owner(1).unwrap();
        }
        let before = snapshot.clone();
        let cause = delete(&mut snapshot).unwrap_err().to_string();
        assert!(cause.contains("parameter 1") && cause.contains("owned Model scalar"));
        assert!(before.sketches.same_evaluated_state(&snapshot.sketches));
        assert_eq!(before.parameters.get(1), snapshot.parameters.get(1));
    }
}
#[test]
fn invalid_constraint_owner_is_refused_before_the_cascade() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    let before = snapshot.clone();
    let cause = snapshot
        .delete_sketch_dimension(
            DocumentSketchId(1),
            SketchConstraintId(99),
            Default::default(),
        )
        .unwrap_err();
    assert!(cause.to_string().contains("99"));
    assert!(before.sketches.same_evaluated_state(&snapshot.sketches));
    assert_eq!(before.parameters.get(1), snapshot.parameters.get(1));
}
#[test]
fn another_relation_with_a_deleted_driver_cannot_publish_a_partial_candidate() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    add_shared_dimension(&mut snapshot);
    let before = snapshot.clone();
    assert!(delete(&mut snapshot).is_err());
    assert!(before.sketches.same_evaluated_state(&snapshot.sketches));
    assert_eq!(before.parameters.get(1), snapshot.parameters.get(1));
}

fn add_shared_dimension(snapshot: &mut DocumentSnapshot) {
    let mut geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    let relation = PlanarConstraint::Radius {
        curve: SketchEntityId(2),
        parameter: 1,
    };
    geometry
        .add_constraint(SketchConstraintId(2), relation)
        .unwrap();
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .unwrap();
}
