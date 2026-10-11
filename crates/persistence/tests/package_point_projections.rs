// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::DOCUMENT_PART;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches, SketchPointProjection},
    feature::{FeatureId, work_geometry::WorkPointReference},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use oblikovati_persistence::package::DocumentPackage;
fn sketches() -> DocumentSketches {
    let mut geometry = Sketch2D::default();
    geometry
        .add_point(SketchEntityId(7), [-0.0, f64::MAX])
        .unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add(
            DocumentSketchId(3),
            "Projection",
            SketchPlane::xy(),
            geometry,
        )
        .unwrap();
    sketches
        .set_point_projections(DocumentSketchId(3), vec![binding()])
        .unwrap();
    sketches
}
fn binding() -> SketchPointProjection {
    SketchPointProjection {
        anchor: SketchEntityId(7),
        reference: WorkPointReference::Feature(FeatureId(5)),
    }
}
fn package() -> DocumentPackage {
    DocumentPackage::empty(DOCUMENT_PART, "Projected")
        .with_sketches(&sketches(), 9, 15)
        .unwrap()
}
fn yaml_document() -> serde_yaml_ng::Value {
    serde_yaml_ng::from_slice(&package().encode().unwrap()).unwrap()
}
fn decode_document(document: &serde_yaml_ng::Value) -> DocumentPackage {
    DocumentPackage::decode(serde_yaml_ng::to_string(document).unwrap().as_bytes()).unwrap()
}

#[test]
fn yaml_round_trip_preserves_binding_identity_seed_bits_and_reservation_marks() {
    let package = package();
    let decoded = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    let mut restored = decoded.restore_sketches().unwrap();
    assert!(restored.sketches.same_recipe(&sketches()));
    assert_eq!(restored.sketches.recipe().version, 4);
    let placed = restored.sketches.get(DocumentSketchId(3)).unwrap();
    let position = placed.geometry().points()[0].position;
    assert_eq!(
        position.map(f64::to_bits),
        [-0.0, f64::MAX].map(f64::to_bits)
    );
    assert_eq!(restored.sketch_ids.allocate().unwrap(), 10);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 16);
    assert_eq!(package.encode().unwrap(), decoded.encode().unwrap());
}
#[test]
fn older_recipe_versions_cannot_silently_accept_projection_sources() {
    for version in [1u64, 2, 3, 5] {
        let mut document = yaml_document();
        document["model"]["planarSketches"]["recipe"]["version"] = version.into();
        assert!(decode_document(&document).restore_sketches().is_err());
    }
}
#[test]
fn zero_and_missing_anchor_ids_are_refused_during_package_restore() {
    for anchor in [0u64, 8] {
        let mut document = yaml_document();
        document["model"]["planarSketches"]["recipe"]["sketches"][0]["point_projections"][0]["anchor"] =
            anchor.into();
        assert!(decode_document(&document).restore_sketches().is_err());
    }
}
