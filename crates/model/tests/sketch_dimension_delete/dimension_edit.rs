// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{history::SketchDimensionEdit, quantity::Dimension};
fn apply(snapshot: &mut DocumentSnapshot, edit: SketchDimensionEdit) -> Result<(), SketchError> {
    snapshot.edit_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1), &edit)
}
fn bounds(lower: f64, upper: f64) -> SketchDimensionLimits {
    SketchDimensionLimits { lower, upper }
}
fn target(snapshot: &DocumentSnapshot) -> f64 {
    snapshot.parameters.model_value(1).unwrap().value
}
fn drive(snapshot: &mut DocumentSnapshot, expression: &str) -> Result<(), SketchError> {
    let edit = SketchDimensionEdit {
        target_expression: Some(expression.into()),
        ..Default::default()
    };
    apply(snapshot, edit)
}
fn set_limits(snapshot: &mut DocumentSnapshot, lower: f64, upper: f64) {
    let edit = SketchDimensionEdit {
        drive_limits: Some(bounds(lower, upper)),
        ..Default::default()
    };
    apply(snapshot, edit).unwrap();
}
fn set_reference(snapshot: &mut DocumentSnapshot, reference: bool) {
    let edit = SketchDimensionEdit {
        reference: Some(reference),
        ..Default::default()
    };
    apply(snapshot, edit).unwrap();
}
#[test]
fn limits_do_not_clamp_the_old_target_but_drive_clamps_and_freezes_expression() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    snapshot
        .parameters
        .insert_expression(2, "width", USER_PARAM, "4 cm")
        .unwrap();
    set_limits(&mut snapshot, 2.0, 3.0);
    assert_eq!(target(&snapshot), 1.0);
    drive(&mut snapshot, "width * 2").unwrap();
    assert_eq!(target(&snapshot), 3.0);
    snapshot.parameters.set_expression(2, "5 cm").unwrap();
    assert_eq!(target(&snapshot), 3.0);
    drive(&mut snapshot, "1 mm").unwrap();
    assert_eq!(target(&snapshot), 2.0);
}
#[test]
fn drive_uses_document_display_units_and_recomputes_parameter_readers() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    snapshot
        .parameters
        .insert_expression(2, "reader", USER_PARAM, "radius * 2")
        .unwrap();
    drive(&mut snapshot, "25").unwrap();
    assert_eq!(target(&snapshot), 2.5);
    assert_eq!(snapshot.parameters.model_value(2).unwrap().value, 5.0);
    drive(&mut snapshot, "30 deg").unwrap_err();
    assert_eq!(target(&snapshot), 2.5);
}
#[test]
fn dimension_edit_keeps_expression_and_tracks_its_driver() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    snapshot
        .parameters
        .insert_expression(2, "width", USER_PARAM, "2 cm")
        .unwrap();
    drive(&mut snapshot, "width * 2").unwrap();
    assert_eq!(snapshot.parameters.get(1).unwrap().source, "width * 2");
    assert_eq!(target(&snapshot), 4.0);
    snapshot.parameters.set_expression(2, "3 cm").unwrap();
    assert_eq!(target(&snapshot), 6.0);
}
#[test]
fn self_reference_is_rejected_without_changing_the_dimension() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    let before = snapshot.clone();
    assert!(drive(&mut snapshot, "radius").is_err());
    assert_unchanged(&before, &snapshot);
    set_limits(&mut snapshot, 2.0, 3.0);
    let before = snapshot.clone();
    assert!(drive(&mut snapshot, "radius * 4").is_err());
    assert_unchanged(&before, &snapshot);
}
#[test]
fn failed_owned_targets_can_be_driven_but_user_and_missing_targets_cannot() {
    let edit = SketchDimensionEdit {
        target_expression: Some("2 cm".into()),
        ..Default::default()
    };
    let mut failed = snapshot(MODEL_PARAM, "missing");
    apply(&mut failed, edit.clone()).unwrap();
    assert_eq!(target(&failed), 2.0);
    for absent in [false, true] {
        let mut protected = snapshot(USER_PARAM, "1 cm");
        if absent {
            protected.parameters.remove_for_owner(1).unwrap();
        }
        let before = protected.clone();
        let message = apply(&mut protected, edit.clone()).unwrap_err().to_string();
        assert!(message.contains("parameter 1") && message.contains("owned Model scalar"));
        assert_unchanged(&before, &protected);
    }
}
#[test]
fn invalid_expressions_and_bounds_cannot_publish_partial_settings() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    for source in ["missing", "true", "1 cm^2", "1 cm / 0", "1e309 cm"] {
        let before = snapshot.clone();
        let edit = SketchDimensionEdit {
            target_expression: Some(source.into()),
            reference: Some(true),
            drive_limits: Some(bounds(2.0, 3.0)),
        };
        let message = apply(&mut snapshot, edit).unwrap_err().to_string();
        assert!(message.contains(source));
        assert_unchanged(&before, &snapshot);
    }
}
#[test]
fn invalid_bounds_cannot_change_reference_mode() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    let before = snapshot.clone();
    apply(
        &mut snapshot,
        SketchDimensionEdit {
            reference: Some(true),
            drive_limits: Some(bounds(3.0, 2.0)),
            ..Default::default()
        },
    )
    .unwrap_err();
    assert_unchanged(&before, &snapshot);
}
fn assert_unchanged(before: &DocumentSnapshot, after: &DocumentSnapshot) {
    assert!(before.sketches.same_evaluated_state(&after.sketches));
    assert_eq!(before.parameters.get(1), after.parameters.get(1));
}
#[test]
fn reference_mode_does_not_overwrite_a_failed_backing_parameter() {
    let mut snapshot = snapshot(MODEL_PARAM, "missing");
    let before = snapshot.parameters.get(1).unwrap().clone();
    set_reference(&mut snapshot, true);
    assert_eq!(snapshot.parameters.get(1).unwrap(), &before);
    let geometry = source_geometry(&snapshot);
    assert!(
        geometry
            .dimension_is_reference(SketchConstraintId(1))
            .unwrap()
    );
    assert!(
        geometry
            .solve(&snapshot.parameters, Default::default(), || false)
            .is_ok()
    );
}
#[test]
fn invalid_owners_are_refused_before_any_settings_change() {
    let mut snapshot = snapshot(MODEL_PARAM, "1 cm");
    let before = snapshot.clone();
    let edit = SketchDimensionEdit {
        reference: Some(true),
        ..Default::default()
    };
    for sketch in [DocumentSketchId(1), DocumentSketchId(99)] {
        assert!(
            snapshot
                .edit_sketch_dimension(sketch, SketchConstraintId(99), &edit)
                .is_err()
        );
    }
    assert_unchanged(&before, &snapshot);
    assert_eq!(
        snapshot.parameters.model_value(1).unwrap().dimension,
        Dimension::Length
    );
}

/// A quarter arc of radius 1 about the origin, with its radius dimension.
fn arc_snapshot() -> DocumentSnapshot {
    let mut geometry = Sketch2D::default();
    for (id, position) in [(1, [0., 0.]), (2, [1., 0.]), (3, [0., 1.])] {
        geometry.add_point(SketchEntityId(id), position).unwrap();
    }
    geometry
        .add_arc(
            SketchEntityId(4),
            SketchEntityId(1),
            [SketchEntityId(2), SketchEntityId(3)],
            true,
        )
        .unwrap();
    let relation = PlanarConstraint::Radius {
        curve: SketchEntityId(4),
        parameter: 1,
    };
    geometry
        .add_constraint(SketchConstraintId(1), relation)
        .unwrap();
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "radius", MODEL_PARAM, "1 cm")
        .unwrap();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Arc", SketchPlane::xy(), geometry)
        .unwrap();
    snapshot
}
#[test]
fn an_arc_keeps_its_center_and_its_turn_when_its_radius_is_driven() {
    let mut snapshot = arc_snapshot();
    drive(&mut snapshot, "3 cm").unwrap();
    let geometry = source_geometry(&snapshot);
    let at = |id| geometry.point(SketchEntityId(id)).unwrap().position;
    let close = |left: [f64; 2], right: [f64; 2]| {
        (left[0] - right[0]).abs() < 1e-9 && (left[1] - right[1]).abs() < 1e-9
    };
    assert!(close(at(1), [0., 0.]), "center {:?}", at(1));
    assert!(close(at(2), [3., 0.]), "start {:?}", at(2));
    assert!(close(at(3), [0., 3.]), "end {:?}", at(3));
}
fn source_geometry(snapshot: &DocumentSnapshot) -> &Sketch2D {
    let placed = snapshot.sketches.get(DocumentSketchId(1)).unwrap();
    placed.geometry()
}

#[path = "angular_drive.rs"]
mod angles;
