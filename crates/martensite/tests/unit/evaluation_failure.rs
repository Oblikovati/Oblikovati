// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkbrep::{
    brep::BRepBuilder,
    topods::{Face, Shape},
};
#[test]
fn native_validator_failure_keeps_cause_class_stage_and_context() {
    let cause =
        crate::checked_shape::require_geometry(&Shape::default(), "prism result").unwrap_err();
    assert_eq!(cause.phase(), EvaluationPhase::Validation);
    assert_eq!(cause.kind(), EvaluationFailureKind::Kernel);
    assert_eq!(cause.entity(), &EvaluationEntity::Body);
    let native = cause.source().unwrap().downcast_ref::<Failure>().unwrap();
    assert_eq!(native.exception_type(), "Standard_NullObject");
    assert_eq!(
        cause.to_string(),
        format!("prism result: expected geometric validity: {native}")
    );
    assert_eq!(cause.feature(), None);
    assert_eq!(cause.sketch(), None);
}
#[test]
fn invalid_geometry_is_distinct_from_a_failed_validator() {
    let mut face = Face::default();
    BRepBuilder::default().make_face(&mut face);
    let cause = crate::checked_shape::require_geometry(&face, "prism profile").unwrap_err();
    assert_eq!(cause.kind(), EvaluationFailureKind::InvalidGeometry);
    assert!(cause.source().is_none());
    assert_eq!(
        cause.to_string(),
        "prism profile: expected a geometrically valid B-rep"
    );
}
#[test]
fn cloned_failure_retains_native_type_bytes_stack_and_replay_context() {
    let native = Failure::with_type_bytes(
        "Standard_DomainError",
        b"bad edge 7\xff",
        b"native trace\xfe",
    );
    let failure = EvaluationPhase::Profile
        .native(EvaluationEntity::Edge(7), String::new(), native.clone())
        .for_sketch(
            DocumentSketchId(2),
            "document sketch DocumentSketchId(2)".into(),
        )
        .for_feature(FeatureId(3));
    let cloned = failure.clone();
    assert_eq!(cloned, failure);
    assert_eq!(cloned.feature(), Some(FeatureId(3)));
    assert_eq!(cloned.sketch(), Some(DocumentSketchId(2)));
    assert_eq!(
        cloned.source().unwrap().downcast_ref::<Failure>(),
        Some(&native)
    );
    assert_eq!(format!("{cloned:?}"), format!("{:?}", cloned.to_string()));
}
#[test]
fn model_cause_and_naming_context_are_not_kernel_exceptions() {
    let sketch = SketchError::Definition("bad point 9: expected finite coordinates".into());
    let failure =
        EvaluationPhase::Placement.sketch(EvaluationEntity::SketchEntity(9), sketch.clone());
    assert_eq!(
        failure.source().unwrap().downcast_ref::<SketchError>(),
        Some(&sketch)
    );
    assert_eq!(failure.kind(), EvaluationFailureKind::InvalidInput);
    assert_eq!(failure.to_string(), sketch.to_string());
    let naming = EvaluationPhase::Naming
        .refused(
            EvaluationFailureKind::InvalidReference,
            EvaluationEntity::Body,
            "bad name: expected one binding".into(),
        )
        .with_sketch_identity(DocumentSketchId(2));
    assert_eq!(naming.sketch(), Some(DocumentSketchId(2)));
    assert!(naming.source().is_none());
    assert_eq!(naming.to_string(), "bad name: expected one binding");
}
#[test]
fn copy_failure_retains_the_native_exception() {
    let failure = crate::checked_shape::copy_shape(&Shape::default(), "prism copy")
        .err()
        .unwrap();
    assert_eq!(failure.phase(), EvaluationPhase::Copy);
    assert_eq!(failure.kind(), EvaluationFailureKind::Kernel);
    assert!(
        failure
            .source()
            .unwrap()
            .downcast_ref::<Failure>()
            .is_some()
    );
}
#[test]
fn collinear_prism_retains_validation_decline_and_legacy_text() {
    let definition = crate::PrismProbeDefinition {
        outer: crate::PrismProbeLoop {
            edge_keys: vec![1, 2, 3],
            vertices: vec![[0., 0.], [1., 0.], [2., 0.]],
        },
        holes: Vec::new(),
        displacement: [0., 0., 1.],
    };
    let failure =
        crate::prism::build_prism(&definition, oblikovati_model::sketch::SketchPlane::xy())
            .err()
            .unwrap();
    assert_eq!(failure.phase(), EvaluationPhase::Validation);
    assert_eq!(failure.kind(), EvaluationFailureKind::InvalidGeometry);
    assert!(failure.source().is_none());
    assert_eq!(
        crate::retain_planar_prism(&definition).err().unwrap(),
        failure.to_string()
    );
}
