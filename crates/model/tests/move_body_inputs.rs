// SPDX-License-Identifier: GPL-2.0-only
//! Ordered named body-motion input validation and dependencies (#4097).
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    feature::{
        FeatureId, FeatureReadKey,
        move_body::{MoveBodyDefinition, MoveBodyOperation, ResolvedBodyMotion},
        work_geometry::{DatumAngle, DatumLength},
    },
    identity::{DocumentGuid, SemanticReferenceName},
    parameter::ParameterCollection,
};
fn key(kind: u32) -> Vec<u8> {
    let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(doc, 2, kind, "fixture", "source", &[1])
        .unwrap()
        .encode()
        .unwrap()
}
fn inputs() -> ParameterCollection<u64> {
    let mut params = ParameterCollection::new(Default::default());
    params
        .insert_expression(1, "offset", USER_PARAM, "10 mm")
        .unwrap();
    params
        .insert_expression(2, "angle", USER_PARAM, "90 deg")
        .unwrap();
    params
}
fn point(values: [f64; 3]) -> [DatumLength; 3] {
    values.map(DatumLength::Constant)
}
#[test]
fn ordered_parameter_driven_operations_resolve_and_track_all_reads() {
    let mut params = inputs();
    let definition = MoveBodyDefinition::new(
        key(4),
        vec![
            MoveBodyOperation::Translation([
                DatumLength::Parameter(1),
                DatumLength::Constant(-1.),
                DatumLength::Constant(0.),
            ]),
            MoveBodyOperation::AlongRay {
                direction: [0., 3., 4.],
                distance: DatumLength::Parameter(1),
            },
            MoveBodyOperation::Rotation {
                origin: point([0., 0., 0.]),
                direction: [0., 0., 1.],
                angle: DatumAngle::parameter(2),
            },
        ],
    )
    .unwrap();
    assert_eq!(definition.body_source(), FeatureId(2));
    for read in [
        FeatureReadKey::Feature(FeatureId(2)),
        FeatureReadKey::Parameter(1),
        FeatureReadKey::Parameter(2),
    ] {
        assert!(definition.reads().keys().contains(&read));
    }
    let first = definition.resolve(&params).unwrap();
    assert_eq!(first[0], ResolvedBodyMotion::Translation([1., -1., 0.]));
    let ResolvedBodyMotion::Translation(vector) = first[1] else {
        panic!("expected ray translation")
    };
    assert!((vector[1] - 0.6).abs() < 1e-12 && (vector[2] - 0.8).abs() < 1e-12);
    let ResolvedBodyMotion::Rotation { radians, .. } = first[2] else {
        panic!("expected rotation")
    };
    assert!((radians - std::f64::consts::FRAC_PI_2).abs() < 1e-12);
    params.set_expression(1, "20 mm").unwrap();
    assert_eq!(
        definition.resolve(&params).unwrap()[0],
        ResolvedBodyMotion::Translation([2., -1., 0.])
    );
}
#[test]
fn malformed_sources_axes_and_dimensions_are_refused() {
    assert!(MoveBodyDefinition::new(key(4), vec![]).is_err());
    assert!(
        MoveBodyDefinition::new(key(1), vec![MoveBodyOperation::Translation(point([0.; 3]))])
            .is_err()
    );
    for direction in [[0.; 3], [f64::NAN, 0., 1.], [f64::INFINITY, 1., 0.]] {
        assert!(
            MoveBodyDefinition::new(
                key(4),
                vec![MoveBodyOperation::AlongRay {
                    direction,
                    distance: DatumLength::Constant(1.)
                }]
            )
            .is_err()
        );
    }
    for bad in [DatumLength::Parameter(0), DatumLength::Constant(f64::NAN)] {
        assert!(
            MoveBodyDefinition::new(key(4), vec![MoveBodyOperation::Translation([bad; 3])])
                .is_err()
        );
    }
    let wrong_angle = MoveBodyDefinition::new(
        key(4),
        vec![MoveBodyOperation::Rotation {
            origin: point([0.; 3]),
            direction: [1., 0., 0.],
            angle: DatumAngle::parameter(1),
        }],
    )
    .unwrap();
    assert!(wrong_angle.resolve(&inputs()).is_err());
}
#[test]
fn extreme_directions_and_exact_recipe_comparison_are_stable() {
    for scale in [f64::MAX, f64::MIN_POSITIVE] {
        let a = MoveBodyDefinition::new(
            key(4),
            vec![MoveBodyOperation::AlongRay {
                direction: [scale, scale, 0.],
                distance: DatumLength::Constant(2.),
            }],
        )
        .unwrap();
        let ResolvedBodyMotion::Translation(vector) = a.resolve(&inputs()).unwrap()[0] else {
            panic!("expected translation")
        };
        assert!((vector[0] - std::f64::consts::SQRT_2).abs() < 1e-12);
        assert!(a.same_recipe(&a.clone()));
    }
    let a = MoveBodyDefinition::new(
        key(4),
        vec![MoveBodyOperation::Translation(point([0., 0., 0.]))],
    )
    .unwrap();
    let b = MoveBodyDefinition::new(
        key(4),
        vec![MoveBodyOperation::Translation(point([-0., 0., 0.]))],
    )
    .unwrap();
    assert!(!a.same_recipe(&b));
}
