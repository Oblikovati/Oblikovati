// SPDX-License-Identifier: GPL-2.0-only
//! Motion request conversion and validation (#4097).
use super::*;
use oblikovati_api::wire::featureargs::MoveBodyOp;
use oblikovati_model::feature::{move_body::MoveBodyOperation, work_geometry::DatumLength};

#[test]
fn ordered_operations_keep_parameter_drivers_and_precede_translation() {
    let snapshot = crate::prism_test_fixture::selected_prism_document();
    let input = MoveBody {
        translation: Some(vec![9., 9., 9.]),
        operations: Some(vec![
            MoveBodyOp {
                type_: "freeDrag".into(),
                x: "width".into(),
                y: "-2 mm".into(),
                ..Default::default()
            },
            MoveBodyOp {
                type_: "alongRay".into(),
                dir: Some(vec![0., 0., 1.]),
                dist: "3 mm".into(),
                ..Default::default()
            },
            MoveBodyOp {
                type_: "rotateAboutLine".into(),
                dir: Some(vec![0., 0., 1.]),
                point: Some(vec![0., 0., 0.]),
                angle: "90 deg".into(),
                ..Default::default()
            },
        ]),
        ..Default::default()
    };
    let operations = input::operations(&snapshot, &input).unwrap();
    assert_eq!(operations.len(), 3);
    assert_eq!(
        operations[0],
        MoveBodyOperation::Translation([
            DatumLength::Parameter(1),
            DatumLength::Constant(-0.2),
            DatumLength::Constant(0.)
        ])
    );
    assert!(
        matches!(operations[1], MoveBodyOperation::AlongRay { distance: DatumLength::Constant(value), .. } if (value - 0.3).abs() < 1e-12)
    );
    assert!(
        matches!(operations[2], MoveBodyOperation::Rotation { angle, .. } if (angle.resolve(&snapshot.parameters).unwrap() - std::f64::consts::FRAC_PI_2).abs() < 1e-12)
    );
}
#[test]
fn motion_rejects_invalid_vectors_types_and_dimensions() {
    let snapshot = crate::prism_test_fixture::selected_prism_document();
    for op in [
        MoveBodyOp {
            type_: "bad".into(),
            ..Default::default()
        },
        MoveBodyOp {
            type_: "alongRay".into(),
            dir: Some(vec![1., 2.]),
            dist: "1 mm".into(),
            ..Default::default()
        },
        MoveBodyOp {
            type_: "rotateAboutLine".into(),
            dir: Some(vec![0., 0., 1.]),
            point: Some(vec![0., 0., 0.]),
            angle: "1 mm".into(),
            ..Default::default()
        },
    ] {
        let input = MoveBody {
            operations: Some(vec![op]),
            ..Default::default()
        };
        assert!(input::operations(&snapshot, &input).is_err());
    }
    assert!(input::operations(&snapshot, &MoveBody::default()).is_err());
}
