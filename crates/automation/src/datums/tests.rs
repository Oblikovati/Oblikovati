// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    feature::{
        FeatureHealthRecord, FeatureSuppressionComparison, FeatureSuppressionCondition,
        work_geometry::{DatumLength, WorkAxisPoints, WorkPointReference},
    },
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
    sketch::SketchPlane,
    units::UnitsOfMeasure,
};

pub(super) fn snapshot() -> DocumentSnapshot {
    DocumentSnapshot {
        work_geometry: Default::default(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        sketch_settings: None,
        parameters: ParameterCollection::new(UnitsOfMeasure::default()),
        properties: Default::default(),
    }
}
fn add_point(snapshot: &mut DocumentSnapshot, id: u64) {
    let definition = WorkGeometryDefinition::Point([DatumLength::Constant(0.0); 3]);
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(id), "Point", definition))
        .unwrap();
}
#[test]
fn references_are_canonical_live_and_verified_against_the_intended_feature() {
    let mut snapshot = snapshot();
    add_point(&mut snapshot, 1);
    add_point(&mut snapshot, 2);
    let key = reference::encode_reference(&snapshot, FeatureId(1)).unwrap();
    assert_eq!(
        reference::decode_reference(&snapshot, &key).unwrap(),
        FeatureId(1)
    );
    assert!(
        reference::verify_reference(&snapshot, FeatureId(2), &key)
            .unwrap_err()
            .to_string()
            .contains("expected FeatureId(2)")
    );
    assert_malformed_references(&snapshot);
}
fn assert_malformed_references(snapshot: &DocumentSnapshot) {
    for key in [
        "",
        "point/1",
        "datum/v2/1",
        "datum/v1/0",
        "datum/v1/01",
        "datum/v1/+1",
        "datum/v1/18446744073709551616",
        "datum/v1/99",
    ] {
        assert!(reference::decode_reference(snapshot, key).is_err());
    }
}
#[test]
fn indices_refuse_overflow_and_other_dispatchers_are_not_datum_commands() {
    assert_eq!(rows::wire_index(2, 3).unwrap(), 5);
    assert!(rows::wire_index(usize::MAX, 1).is_err());
    assert!(rows::wire_index(usize::MAX, 0).is_err());
    assert!(
        input::definition(METHOD_WORK_AXES_LIST, b"{}", &snapshot())
            .unwrap_err()
            .to_string()
            .contains("expected")
    );
}
fn declined_program() -> DocumentSnapshot {
    let mut snapshot = snapshot();
    add_point(&mut snapshot, 1);
    let failed = WorkGeometryDefinition::Point([DatumLength::Parameter(99); 3]);
    snapshot
        .work_geometry
        .replace(FeatureId(1), failed, failed.reads())
        .unwrap();
    let axis = WorkGeometryDefinition::AxisThroughPoints(WorkAxisPoints {
        first: WorkPointReference::Origin,
        second: WorkPointReference::Feature(FeatureId(1)),
    });
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(FeatureId(2), "Axis", axis))
        .unwrap();
    add_declined_points(&mut snapshot);
    snapshot
}
fn add_declined_points(snapshot: &mut DocumentSnapshot) {
    for id in [3, 4, 5, 6] {
        add_point(snapshot, id);
    }
    snapshot
        .work_geometry
        .set_suppressed(FeatureId(3), true)
        .unwrap();
    let condition = FeatureSuppressionCondition::new(
        99,
        FeatureSuppressionComparison::GreaterThan,
        Quantity::new(1.0, Dimension::Length),
    )
    .unwrap();
    snapshot
        .work_geometry
        .set_suppression_condition(FeatureId(4), Some(condition))
        .unwrap();
    snapshot.work_geometry.set_end_of_part(Some(5)).unwrap();
}
#[test]
fn listing_health_preserves_local_causes_suppression_blocking_and_cutoff() {
    let snapshot = declined_program();
    let evaluation = snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .unwrap();
    for id in [1, 3, 4, 6] {
        let cause = evaluation.point(FeatureId(id)).unwrap_err();
        assert!(
            !health::reason(&evaluation, FeatureId(id), Some(&cause))
                .unwrap()
                .is_empty()
        );
    }
    assert_blocked_axis(&evaluation);
    assert_eq!(health::reason(&evaluation, FeatureId(5), None).unwrap(), "");
    assert_missing_healthy_result(&evaluation);
}
fn assert_blocked_axis(evaluation: &WorkGeometryEvaluation) {
    let cause = evaluation.axis(FeatureId(2)).unwrap_err();
    assert!(
        health::reason(evaluation, FeatureId(2), Some(&cause))
            .unwrap()
            .contains("Blocked")
    );
}
fn assert_missing_healthy_result(evaluation: &WorkGeometryEvaluation) {
    let cause = WorkGeometryError::Unavailable {
        feature: FeatureId(5),
        expected: "point",
    };
    assert!(health::reason(evaluation, FeatureId(5), Some(&cause)).is_err());
}
#[test]
fn duplicate_receipts_and_wrong_constructor_rows_are_named_integrity_errors() {
    let mut evaluation = WorkGeometryEvaluation::default();
    let record = FeatureHealthRecord {
        feature: FeatureId(1),
        status: FeatureEvaluationStatus::Suppressed,
    };
    evaluation.health = vec![record.clone(), record];
    let cause = WorkGeometryError::Unavailable {
        feature: FeatureId(1),
        expected: "point",
    };
    assert!(
        health::reason(&evaluation, FeatureId(1), Some(&cause))
            .unwrap_err()
            .to_string()
            .contains("at most one")
    );
    assert!(rows::constructor_kind(WorkGeometryDefinition::Fixed(SketchPlane::xy())).is_err());
}

#[test]
fn list_requests_accept_sdk_defaults_and_null_but_refuse_malformed_shapes() {
    for bytes in [b"".as_slice(), b"null", b"{}"] {
        let args: ListWorkPointsArgs = query_args(METHOD_WORK_POINTS_LIST, bytes).unwrap();
        assert!(!args.include_construction);
        assert!(args.occurrence.is_none());
    }
    for bytes in [
        b"[]".as_slice(),
        b"false",
        b"{",
        b"{\"includeConstruction\":1}",
    ] {
        assert!(query_args::<ListWorkPointsArgs>(METHOD_WORK_POINTS_LIST, bytes).is_err());
    }
}
