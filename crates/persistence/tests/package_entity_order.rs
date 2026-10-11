// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::DOCUMENT_PART;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use oblikovati_persistence::package::DocumentPackage;
fn geometry() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(5), [0.0; 2]).unwrap();
    sketch
        .add_circle(SketchEntityId(1), SketchEntityId(5), 2.0)
        .unwrap();
    sketch.add_point(SketchEntityId(9), [2.0, 0.0]).unwrap();
    sketch
        .add_line(SketchEntityId(3), [SketchEntityId(5), SketchEntityId(9)])
        .unwrap();
    sketch
}
fn sketches() -> DocumentSketches {
    let mut sketches = DocumentSketches::default();
    sketches
        .add(
            DocumentSketchId(7),
            "Ordered",
            SketchPlane::xy(),
            geometry(),
        )
        .unwrap();
    sketches
}
fn package() -> DocumentPackage {
    DocumentPackage::empty(DOCUMENT_PART, "Ordered")
        .with_sketches(&sketches(), 7, 9)
        .unwrap()
}
#[test]
fn native_yaml_preserves_mixed_geometry_order_and_monotonic_reservations() {
    let original = package();
    let bytes = original.encode().unwrap();
    let decoded = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = decoded.restore_sketches().unwrap();
    assert!(restored.sketches.same_recipe(&sketches()));
    let sketch = restored
        .sketches
        .get(DocumentSketchId(7))
        .unwrap()
        .geometry();
    assert_eq!(sketch.entity_ids(), geometry().entity_ids());
    assert_eq!(sketch.recipe().version, 2);
    assert_eq!(restored.sketch_ids.allocate().unwrap(), 8);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 10);
    assert_eq!(decoded.encode().unwrap(), bytes);
}
fn legacy_package() -> DocumentPackage {
    let mut legacy = geometry().recipe();
    legacy.version = 1;
    legacy.entity_order = None;
    let geometry = Sketch2D::from_recipe(&legacy).unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add(DocumentSketchId(7), "Legacy", SketchPlane::xy(), geometry)
        .unwrap();
    DocumentPackage::empty(DOCUMENT_PART, "Legacy")
        .with_sketches(&sketches, 7, 9)
        .unwrap()
}
#[test]
fn legacy_geometry_recipes_stay_version_one_inside_current_native_packages() {
    let package = legacy_package();
    let restored = DocumentPackage::decode(&package.encode().unwrap())
        .unwrap()
        .restore_sketches()
        .unwrap();
    let geometry = restored
        .sketches
        .get(DocumentSketchId(7))
        .unwrap()
        .geometry();
    assert_eq!(geometry.recipe().version, 1);
    assert!(geometry.recipe().entity_order.is_none());
    let order: Vec<u64> = geometry.entity_ids().iter().map(|id| id.0).collect();
    assert_eq!(order, [5, 9, 3, 1]);
}
