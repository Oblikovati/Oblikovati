// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::MODEL_PARAM;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    history::SketchReservationMarks,
    parameter::ParameterCollection,
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchPlane},
};
use oblikovati_persistence::package::DocumentPackage;
use std::collections::BTreeMap;
fn geometry() -> DocumentSketches {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
    geometry
        .add_circle(SketchEntityId(2), SketchEntityId(1), 3.0)
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Radius {
                curve: SketchEntityId(2),
                parameter: 1,
            },
        )
        .unwrap();
    placed_geometry(geometry)
}
fn placed_geometry(geometry: Sketch2D) -> DocumentSketches {
    let mut sketches = DocumentSketches::default();
    sketches
        .add(DocumentSketchId(1), "Radius", SketchPlane::xy(), geometry)
        .unwrap();
    sketches
}
fn package() -> DocumentPackage {
    let package =
        DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\nmodel:\n  parameters: []\n")
            .unwrap();
    let mut parameters = ParameterCollection::new(Default::default());
    parameters
        .insert_expression(1, "d7", MODEL_PARAM, "3 cm")
        .unwrap();
    package
        .with_parameters(&parameters, parameters.units().clone())
        .unwrap()
}
fn marks(names: &[(u64, u64)]) -> SketchReservationMarks {
    SketchReservationMarks {
        sketch: 2,
        entity: 2,
        constraints: BTreeMap::from([(1, 1)]),
        dimension_names: names.iter().copied().collect(),
    }
}
fn saved() -> DocumentPackage {
    package()
        .with_sketch_reservations(&geometry(), marks(&[(2, 20), (1, 10)]))
        .unwrap()
}
fn replace_names(package: &DocumentPackage, mapping: Option<&str>) -> DocumentPackage {
    let mut root: serde_yaml_ng::Value =
        serde_yaml_ng::from_slice(&package.encode().unwrap()).unwrap();
    let section = root["model"]["planarSketches"].as_mapping_mut().unwrap();
    let key = serde_yaml_ng::Value::String("dimensionNameOrdinals".into());
    section.remove(&key);
    if let Some(mapping) = mapping {
        section.insert(key, serde_yaml_ng::from_str(mapping).unwrap());
    }
    DocumentPackage::decode(serde_yaml_ng::to_string(&root).unwrap().as_bytes()).unwrap()
}
#[test]
fn explicit_name_marks_round_trip_and_allocate_in_retired_namespaces() {
    let saved = saved();
    let encoded = saved.encode().unwrap();
    let reopened = DocumentPackage::decode(&encoded).unwrap();
    let mut restored = reopened.restore_sketches().unwrap();
    assert_eq!(
        restored.dimension_names.marks(),
        BTreeMap::from([(1, 10), (2, 20)])
    );
    let parameters = reopened.restore_document_parameters().unwrap().parameters;
    assert_next_names(&mut restored.dimension_names, &parameters);
    assert_eq!(encoded, reopened.encode().unwrap());
}
#[test]
fn old_documents_bootstrap_live_names_and_new_documents_require_complete_marks() {
    let old = replace_names(&saved(), None);
    assert_eq!(
        old.restore_sketches().unwrap().dimension_names.marks(),
        BTreeMap::from([(1, 8)])
    );
    let captured = old.with_sketches(&geometry(), 2, 2).unwrap();
    assert_eq!(
        captured.restore_sketches().unwrap().dimension_names.marks(),
        BTreeMap::from([(1, 8)])
    );
    for mapping in ["{}", "{1: 7}", "{2: 20}"] {
        assert!(
            replace_names(&saved(), Some(mapping))
                .restore_sketches()
                .is_err()
        );
    }
}
#[test]
fn malformed_saved_name_maps_are_refused_with_the_offending_field() {
    for mapping in [
        "{0: 1}",
        "{3: 1}",
        "{1: -1}",
        "{1: broken}",
        "null",
        "[]",
        "false",
    ] {
        let cause = replace_names(&saved(), Some(mapping))
            .restore_sketches()
            .unwrap_err()
            .to_string();
        assert!(cause.contains("model.planarSketches"), "{mapping}: {cause}");
    }
}
#[test]
fn deleting_all_sketches_and_standalone_capture_retain_permanent_names() {
    let deleted = saved()
        .with_sketches(&DocumentSketches::default(), 2, 2)
        .unwrap();
    assert_eq!(
        deleted.restore_sketches().unwrap().dimension_names.marks(),
        BTreeMap::from([(1, 10), (2, 20)])
    );
    assert_no_name_rewind(&deleted);
}

#[test]
fn empty_and_exhausted_name_maps_have_stable_explicit_storage() {
    let empty = package()
        .with_sketch_reservations(&DocumentSketches::default(), marks(&[]))
        .unwrap();
    let yaml = String::from_utf8(empty.encode().unwrap()).unwrap();
    assert!(yaml.contains("dimensionNameOrdinals: {}"), "{yaml}");
}
#[test]
fn exhausted_saved_name_marks_cannot_allocate_or_rewind() {
    let empty = package()
        .with_sketch_reservations(&DocumentSketches::default(), marks(&[]))
        .unwrap();
    let exhausted = empty
        .with_sketch_reservations(&DocumentSketches::default(), marks(&[(2, u64::MAX)]))
        .unwrap();
    let mut restored = exhausted.restore_sketches().unwrap();
    let parameters = package().restore_document_parameters().unwrap().parameters;
    assert!(
        restored
            .dimension_names
            .allocate(DocumentSketchId(2), 2, &parameters)
            .is_err()
    );
    assert_eq!(
        restored.dimension_names.marks(),
        BTreeMap::from([(2, u64::MAX)])
    );
}

fn assert_next_names(
    names: &mut oblikovati_model::history::SketchDimensionNameReservations,
    parameters: &ParameterCollection<u64>,
) {
    for (owner, expected) in [(1, "d10"), (2, "d20")] {
        assert_eq!(
            names
                .allocate(DocumentSketchId(owner), 2, parameters)
                .unwrap(),
            expected
        );
    }
}

fn assert_no_name_rewind(deleted: &DocumentPackage) {
    for names in [
        vec![],
        vec![(1, 10)],
        vec![(1, 9), (2, 20)],
        vec![(1, 10), (2, 19)],
    ] {
        let cause = deleted
            .with_sketch_reservations(&DocumentSketches::default(), marks(&names))
            .unwrap_err()
            .to_string();
        assert!(cause.contains("dimension name reservation mark"), "{cause}");
    }
}
