// SPDX-License-Identifier: GPL-2.0-only
//! Native document serialization of authored Hole inputs (#4093).
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        hole::{HoleDefinition, HoleExtent, HolePosition, HoleRecipe},
        part::PartFeatureDefinition,
        work_geometry::DatumLength,
    },
    identity::{DocumentGuid, SemanticReferenceName},
    sketch::SketchPlane,
};
use oblikovati_persistence::package::DocumentPackage;

fn reference(kind: u32) -> Vec<u8> {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(document, 1, kind, "hole-test", "source", &[1])
        .unwrap()
        .encode()
        .unwrap()
}
fn program() -> FeatureSequence<PartFeatureDefinition> {
    let mut program = FeatureSequence::default();
    program
        .append(AuthoredFeature::datum(
            FeatureId(1),
            "Source",
            PartFeatureDefinition::Fixed(SketchPlane::xy()),
        ))
        .unwrap();
    let hole = HoleDefinition::from_recipe(HoleRecipe {
        version: 1,
        body: reference(4),
        face: reference(1),
        position: HolePosition::ModelPoint([
            DatumLength::Constant(-0.),
            DatumLength::Parameter(5),
            DatumLength::Constant(0.),
        ]),
        diameter: DatumLength::Parameter(3),
        extent: HoleExtent::Distance(DatumLength::Parameter(4)),
        options: Default::default(),
    })
    .unwrap();
    let reads = hole.reads();
    program
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Hole",
            PartFeatureDefinition::Hole(Box::new(hole)),
            reads,
        ))
        .unwrap();
    program
}

#[test]
fn hole_program_survives_native_yaml_round_trip() {
    let source = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    let original = program();
    let captured = source.with_work_geometry(&original, 2).unwrap();
    let encoded = captured.encode().unwrap();
    let reopened = DocumentPackage::decode(&encoded).unwrap();
    let restored = reopened.restore_work_geometry().unwrap();
    assert!(original.same_work_geometry_recipe(&restored.program));
    assert_eq!(reopened.encode().unwrap(), encoded);
}
