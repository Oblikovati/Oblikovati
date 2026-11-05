// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DocumentType};
fn hidden_package_geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry
        .add_defining_point(SketchEntityId(100), [0.0, -0.0])
        .unwrap();
    geometry
        .add_defining_point(SketchEntityId(200), [2.0, 0.0])
        .unwrap();
    geometry
        .add_line(
            SketchEntityId(10),
            [SketchEntityId(100), SketchEntityId(200)],
        )
        .unwrap();
    geometry
}
fn defining_point_owner() -> DocumentSketches {
    let mut owner = DocumentSketches::default();
    owner
        .add(
            DocumentSketchId(7),
            "Hidden endpoints",
            SketchPlane::xy(),
            hidden_package_geometry(),
        )
        .unwrap();
    owner
}
#[test]
fn both_native_package_owners_preserve_hidden_points_and_reserve_their_highest_id() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_defining_point_package(kind);
    }
}
fn assert_defining_point_package(kind: DocumentType) {
    let owner = defining_point_owner();
    let package = DocumentPackage::empty(kind, "Point ownership")
        .with_sketches(&owner, 7, 200)
        .unwrap();
    let bytes = package.encode().unwrap();
    let decoded = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = decoded.restore_sketches().unwrap();
    assert!(owner.same_recipe(&restored.sketches));
    assert_eq!(restored.entity_ids.allocate().unwrap(), 201);
    assert_eq!(decoded.encode().unwrap(), bytes);
    assert_hidden_package_geometry(&restored.sketches);
}
fn assert_hidden_package_geometry(sketches: &DocumentSketches) {
    let geometry = sketches.get(DocumentSketchId(7)).unwrap().geometry();
    assert_eq!(geometry.entity_ids(), &[SketchEntityId(10)]);
    assert_eq!(geometry.recipe().version, 12);
    assert_eq!(
        geometry.point(SketchEntityId(100)).unwrap().position[1].to_bits(),
        (-0.0_f64).to_bits()
    );
}
#[test]
fn visible_entity_order_does_not_authorize_reuse_of_hidden_solver_point_ids() {
    let owner = defining_point_owner();
    for mark in [10, 100, 199] {
        assert!(
            DocumentPackage::empty(DOCUMENT_PART, "Invalid mark")
                .with_sketches(&owner, 7, mark)
                .is_err()
        );
    }
}
