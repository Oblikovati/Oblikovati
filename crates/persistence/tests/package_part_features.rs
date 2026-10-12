// SPDX-License-Identifier: GPL-2.0-only
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
use fixture::*;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, sketch_prism::SketchPrismDefinition, work_geometry::DatumLength,
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    units::UnitsOfMeasure,
};
use oblikovati_persistence::package::DocumentPackage;
fn saved_part() -> DocumentPackage {
    let mut snapshot = selected_sketch_document();
    snapshot
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    let definition = saved_definition();
    let feature =
        AuthoredFeature::sketch_prism(FeatureId(7), "Extrude", definition, &snapshot).unwrap();
    snapshot.work_geometry.append(feature).unwrap();
    capture_part(&snapshot)
}
fn saved_definition() -> SketchPrismDefinition {
    SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
}
fn capture_part(snapshot: &DocumentSnapshot) -> DocumentPackage {
    let package =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {future: keep}\n")
            .unwrap();
    let parameters = package
        .with_parameters(&snapshot.parameters, UnitsOfMeasure::default())
        .unwrap();
    let sketches = parameters
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap();
    sketches
        .with_work_geometry(&snapshot.work_geometry, 9)
        .unwrap()
}
#[test]
fn stored_part_inputs_restore_with_one_program_and_permanent_feature_mark() {
    let original = saved_part();
    let encoded = original.encode().unwrap();
    let decoded = DocumentPackage::decode(&encoded).unwrap();
    let restored = decoded.restore_work_geometry().unwrap();
    assert_eq!(restored.program.part_recipe().unwrap().version, 16);
    assert_eq!(restored.ids.high_water(), 9);
    assert_eq!(decoded.encode().unwrap(), encoded);
    assert!(
        String::from_utf8(decoded.section_yaml("model").unwrap().unwrap())
            .unwrap()
            .contains("future: keep")
    );
    let mut owner = restored_part_owner(&decoded);
    assert!(owner.snapshot().require_resolved_sketch(SKETCH).is_err());
    refresh_part(&mut owner);
    assert!(owner.snapshot().require_resolved_sketch(SKETCH).is_ok());
    assert_eq!(owner.feature_high_water(), 9);
}
fn refresh_part(owner: &mut DocumentEditHistory) {
    owner
        .refresh(|snapshot| {
            snapshot.sketches = snapshot
                .solve_pending_document_sketches(Default::default(), || false)
                .candidate;
            Ok::<_, String>(())
        })
        .unwrap();
}
fn restored_part_owner(package: &DocumentPackage) -> DocumentEditHistory {
    let restored = package.restore_document_parameters().unwrap();
    let sketches = package.restore_sketches().unwrap();
    let features = package.restore_work_geometry().unwrap();
    let snapshot = DocumentSnapshot {
        parameters: restored.parameters,
        sketches: sketches.sketches,
        work_geometry: features.program,
        sketch_settings: None,
        properties: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
    };
    DocumentEditHistory::new(snapshot, restored.ids)
        .unwrap()
        .with_feature_reservations(9)
        .unwrap()
}
