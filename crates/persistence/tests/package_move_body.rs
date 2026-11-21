// SPDX-License-Identifier: GPL-2.0-only
//! Native YAML preserves ordered body-motion drivers (#4097).
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        move_body::{MoveBodyDefinition, MoveBodyOperation},
        part::PartFeatureDefinition,
        work_geometry::{DatumAngle, DatumLength},
    },
    identity::{DocumentGuid, SemanticReferenceName},
    sketch::SketchPlane,
};
use oblikovati_persistence::package::DocumentPackage;
fn program() -> FeatureSequence<PartFeatureDefinition> {
    let mut program = FeatureSequence::default();
    program
        .append(AuthoredFeature::datum(
            FeatureId(1),
            "Source",
            PartFeatureDefinition::Fixed(SketchPlane::xy()),
        ))
        .unwrap();
    let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let body = SemanticReferenceName::new(doc, 1, 4, "fixture", "source", &[1])
        .unwrap()
        .encode()
        .unwrap();
    let motion = MoveBodyDefinition::new(
        body,
        vec![
            MoveBodyOperation::Translation([
                DatumLength::Constant(-0.),
                DatumLength::Parameter(3),
                DatumLength::Constant(2.),
            ]),
            MoveBodyOperation::AlongRay {
                direction: [0., 3., 4.],
                distance: DatumLength::Parameter(3),
            },
            MoveBodyOperation::Rotation {
                origin: [DatumLength::Constant(0.); 3],
                direction: [0., 0., 1.],
                angle: DatumAngle::parameter(4),
            },
        ],
    )
    .unwrap();
    let reads = motion.reads();
    program
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Move",
            PartFeatureDefinition::MoveBody(Box::new(motion)),
            reads,
        ))
        .unwrap();
    program
}
#[test]
fn motion_program_preserves_order_and_scalar_sources_through_native_yaml() {
    let source = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    let original = program();
    let encoded = source
        .with_work_geometry(&original, 2)
        .unwrap()
        .encode()
        .unwrap();
    let reopened = DocumentPackage::decode(&encoded).unwrap();
    let restored = reopened.restore_work_geometry().unwrap();
    assert!(original.same_work_geometry_recipe(&restored.program));
    assert_eq!(reopened.encode().unwrap(), encoded);
}
