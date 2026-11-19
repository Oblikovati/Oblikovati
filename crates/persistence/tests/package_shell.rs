// SPDX-License-Identifier: GPL-2.0-only
//! Native YAML round-trip of authored Shell inputs (#4106).
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        part::PartFeatureDefinition,
        shell::{ShellDefinition, ShellDirection, ShellFaceThickness},
        work_geometry::DatumLength,
    },
    identity::{DocumentGuid, SemanticReferenceName},
    sketch::SketchPlane,
};
use oblikovati_persistence::package::DocumentPackage;

fn reference(kind: u32, role: &str) -> Vec<u8> {
    let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    SemanticReferenceName::new(document, 1, kind, "shell-test", role, &[1])
        .unwrap()
        .encode()
        .unwrap()
}
#[test]
fn shell_program_survives_native_yaml_round_trip() {
    let mut program = FeatureSequence::default();
    program
        .append(AuthoredFeature::datum(
            FeatureId(1),
            "Source",
            PartFeatureDefinition::Fixed(SketchPlane::xy()),
        ))
        .unwrap();
    let shell = ShellDefinition::new(
        reference(4, "body"),
        vec![reference(1, "opening")],
        DatumLength::Parameter(3),
        ShellDirection::Both,
        vec![ShellFaceThickness {
            face: reference(1, "retained"),
            thickness: DatumLength::Parameter(4),
        }],
    )
    .unwrap();
    let reads = shell.reads();
    program
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Shell",
            PartFeatureDefinition::Shell(Box::new(shell)),
            reads,
        ))
        .unwrap();
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    let encoded = package
        .with_work_geometry(&program, 2)
        .unwrap()
        .encode()
        .unwrap();
    let reopened = DocumentPackage::decode(&encoded).unwrap();
    let restored = reopened.restore_work_geometry().unwrap();
    assert!(program.same_work_geometry_recipe(&restored.program));
    assert_eq!(reopened.encode().unwrap(), encoded);
}
