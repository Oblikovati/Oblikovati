// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureSequence,
        work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneOrigin, WorkPlaneReference},
    },
    sketch::SketchPlane,
};
use oblikovati_persistence::package::DocumentPackage;
fn package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {future: [keep]}\ndisplaySettings: {grid: true}\n").unwrap()
}
fn program() -> FeatureSequence<WorkPlaneDefinition> {
    let mut program = FeatureSequence::default();
    let offset = WorkPlaneDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance: WorkPlaneDistance::Constant(-0.0),
    };
    append(&mut program, 3, "Base: 日本語", offset);
    append(
        &mut program,
        7,
        "Side",
        WorkPlaneDefinition::Fixed(SketchPlane::yz()),
    );
    program.set_suppressed(FeatureId(7), true).unwrap();
    program.set_end_of_part(Some(1)).unwrap();
    program
}
fn append(
    program: &mut FeatureSequence<WorkPlaneDefinition>,
    id: u64,
    name: &str,
    definition: WorkPlaneDefinition,
) {
    program
        .append(AuthoredFeature::work_plane(FeatureId(id), name, definition))
        .unwrap();
}
#[test]
fn authored_program_round_trips_exactly_and_preserves_unrelated_sections() {
    let original = package();
    let program = program();
    let captured = original.with_work_planes(&program, 12).unwrap();
    let encoded = captured.encode().unwrap();
    let decoded = DocumentPackage::decode(&encoded).unwrap();
    let mut restored = decoded.restore_work_planes().unwrap();
    assert!(program.same_work_plane_recipe(&restored.program));
    assert_eq!(restored.ids.allocate().unwrap(), 13);
    assert_eq!(decoded.manifest().schema_version, 3);
    assert_eq!(decoded.encode().unwrap(), encoded);
    assert_retained_sections(&decoded, &original);
}
fn assert_retained_sections(decoded: &DocumentPackage, original: &DocumentPackage) {
    assert_eq!(
        decoded.section_yaml("displaySettings").unwrap(),
        original.section_yaml("displaySettings").unwrap()
    );
    assert!(
        String::from_utf8(decoded.section_yaml("model").unwrap().unwrap())
            .unwrap()
            .contains("future:")
    );
    assert_ne!(
        decoded.recipe_digest().unwrap(),
        original.recipe_digest().unwrap()
    );
    assert_eq!(original.manifest().schema_version, 2);
}
#[test]
fn deleting_all_planes_keeps_reservations_and_rejects_a_lower_mark() {
    let captured = package().with_work_planes(&program(), 12).unwrap();
    let deleted = captured
        .with_work_planes(&FeatureSequence::default(), 12)
        .unwrap();
    let decoded = DocumentPackage::decode(&deleted.encode().unwrap()).unwrap();
    let mut restored = decoded.restore_work_planes().unwrap();
    assert!(restored.program.features().is_empty());
    assert_eq!(restored.ids.allocate().unwrap(), 13);
    assert_retained_mark(&deleted);
}
fn assert_retained_mark(deleted: &DocumentPackage) {
    assert!(
        deleted
            .with_work_planes(&FeatureSequence::default(), 11)
            .unwrap_err()
            .to_string()
            .contains("retained mark 12")
    );
    assert!(
        package()
            .with_work_planes(&program(), 6)
            .unwrap_err()
            .to_string()
            .contains("high-water mark 6")
    );
}
#[test]
fn absent_and_empty_legacy_sections_preserve_an_unchanged_package() {
    for field in ["", "features: []", "features: null"] {
        let input = format!("schemaVersion: 2\ndocumentType: 1\nmodel: {{{field}}}\n");
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        assert!(
            package
                .restore_work_planes()
                .unwrap()
                .program
                .features()
                .is_empty()
        );
        assert_eq!(
            package
                .with_work_planes(&FeatureSequence::default(), 0)
                .unwrap(),
            package
        );
    }
}
#[test]
fn unsupported_legacy_features_cannot_be_silently_discarded() {
    for legacy in ["[{kind: offset-plane}]", "{invalid: shape}", "false"] {
        let input = format!("schemaVersion: 2\ndocumentType: 1\nmodel: {{features: {legacy}}}\n");
        let package = DocumentPackage::decode(input.as_bytes()).unwrap();
        let original = package.encode().unwrap();
        assert!(
            package
                .restore_work_planes()
                .unwrap_err()
                .to_string()
                .contains("explicit legacy feature import")
        );
        assert!(package.with_work_planes(&program(), 12).is_err());
        assert_eq!(package.encode().unwrap(), original);
    }
}
fn section(package: &DocumentPackage) -> serde_yaml_ng::Value {
    serde_yaml_ng::from_slice(&package.section_yaml("model").unwrap().unwrap()).unwrap()
}
fn from_section(model: &serde_yaml_ng::Value, schema: u32) -> DocumentPackage {
    let input = format!(
        "schemaVersion: {schema}\ndocumentType: 1\nmodel:\n{}",
        serde_yaml_ng::to_string(model)
            .unwrap()
            .lines()
            .map(|line| format!("  {line}\n"))
            .collect::<String>()
    );
    DocumentPackage::decode(input.as_bytes()).unwrap()
}
#[test]
fn invalid_versions_marks_and_unknown_fields_fail_before_restore() {
    let captured = package().with_work_planes(&program(), 12).unwrap();
    assert!(
        from_section(&section(&captured), 2)
            .restore_work_planes()
            .unwrap_err()
            .to_string()
            .contains("expected schema 3")
    );
    for (key, value) in [("featureHighWater", 6u64), ("unknown", 1)] {
        let mut model = section(&captured);
        model["workPlanes"][key] = value.into();
        assert!(from_section(&model, 3).restore_work_planes().is_err());
    }
    assert_invalid_recipe_version(&captured);
}
fn assert_invalid_recipe_version(captured: &DocumentPackage) {
    let mut model = section(captured);
    model["workPlanes"]["recipe"]["version"] = 10u64.into();
    assert!(
        from_section(&model, 3)
            .restore_work_planes()
            .unwrap_err()
            .to_string()
            .contains("version 10")
    );
}

#[path = "package_work_planes/mixed.rs"]
mod mixed;

#[test]
fn construction_in_a_legacy_program_is_refused_without_repairing_package_bytes() {
    let (captured, program) = constructed_program_package();
    let mut model = section(&captured);
    model["workPlanes"]["recipe"]["version"] = 1u64.into();
    let invalid = from_section(&model, 3);
    let before = invalid.encode().unwrap();
    assert!(
        invalid
            .restore_work_planes()
            .unwrap_err()
            .to_string()
            .contains("work plane 3")
    );
    assert!(invalid.with_work_planes(&program, 12).is_err());
    assert_eq!(invalid.encode().unwrap(), before);
}
#[test]
fn empty_program_retains_an_authored_cutoff_and_zero_ids_are_refused() {
    let mut planes = FeatureSequence::default();
    planes.set_end_of_part(Some(0)).unwrap();
    let captured = package().with_work_planes(&planes, 0).unwrap();
    assert_eq!(
        captured
            .restore_work_planes()
            .unwrap()
            .program
            .end_of_part(),
        Some(0)
    );
    assert_zero_id_refused();
}
fn assert_zero_id_refused() {
    let mut planes = FeatureSequence::default();
    append(
        &mut planes,
        0,
        "Invalid",
        WorkPlaneDefinition::Fixed(SketchPlane::xy()),
    );
    assert!(
        package()
            .with_work_planes(&planes, 0)
            .unwrap_err()
            .to_string()
            .contains("nonzero ID")
    );
}
#[test]
fn clearing_an_empty_program_cutoff_replaces_the_saved_authored_state() {
    let mut planes = FeatureSequence::default();
    planes.set_end_of_part(Some(0)).unwrap();
    let rolled_back = package().with_work_planes(&planes, 0).unwrap();
    planes.set_end_of_part(None).unwrap();
    let restored_all = rolled_back.with_work_planes(&planes, 0).unwrap();
    let decoded = DocumentPackage::decode(&restored_all.encode().unwrap()).unwrap();
    let restored = decoded.restore_work_planes().unwrap();
    assert_eq!(restored.program.end_of_part(), None);
    assert_ne!(
        restored_all.recipe_digest().unwrap(),
        rolled_back.recipe_digest().unwrap()
    );
    assert_eq!(
        restored_all.with_work_planes(&planes, 0).unwrap(),
        restored_all
    );
}

fn constructed_program_package() -> (DocumentPackage, FeatureSequence<WorkPlaneDefinition>) {
    let mut program = program();
    program
        .set_work_plane_construction(FeatureId(3), true)
        .unwrap();
    let captured = package().with_work_planes(&program, 12).unwrap();
    let restored = captured.restore_work_planes().unwrap();
    assert!(
        restored
            .program
            .get(FeatureId(3))
            .unwrap()
            .is_construction_datum()
    );
    (captured, program)
}
