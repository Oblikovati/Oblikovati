// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchEllipse2D;
fn ellipse_sketches() -> DocumentSketches {
    let mut owner = sketches();
    let mut geometry = geometry();
    let ellipse = SketchEllipse2D::new(
        SketchEntityId(10),
        SketchEntityId(5),
        [3.0, 4.0],
        [4.0, 2.0],
    )
    .unwrap();
    geometry.add_ellipse(ellipse).unwrap();
    geometry.add_point(SketchEntityId(11), [5.0, -7.0]).unwrap();
    geometry
        .set_curve_construction(&[SketchEntityId(10)], true)
        .unwrap();
    owner
        .replace_geometry(DocumentSketchId(7), geometry)
        .unwrap();
    owner
}
#[test]
fn native_package_round_trip_preserves_ellipse_frame_order_and_identity_marks() {
    let owner = ellipse_sketches();
    let package = DocumentPackage::empty(DOCUMENT_PART, "Ellipse")
        .with_sketches(&owner, 7, 11)
        .unwrap();
    let bytes = package.encode().unwrap();
    let decoded = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = decoded.restore_sketches().unwrap();
    assert!(restored.sketches.same_recipe(&owner));
    assert_saved_geometry(&restored.sketches);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 12);
    assert_eq!(decoded.encode().unwrap(), bytes);
}
fn assert_saved_geometry(sketches: &DocumentSketches) {
    let geometry = sketches.get(DocumentSketchId(7)).unwrap().geometry();
    assert_eq!(geometry.recipe().version, 7);
    assert_eq!(
        geometry.ellipse(SketchEntityId(10)).unwrap().major_axis(),
        [0.6, 0.8]
    );
    assert!(geometry.is_construction(SketchEntityId(10)).unwrap());
}

#[test]
fn reservation_marks_cannot_drop_ellipse_or_later_point_identities() {
    let owner = ellipse_sketches();
    let package = DocumentPackage::empty(DOCUMENT_PART, "Ellipse");
    assert!(package.with_sketches(&owner, 7, 9).is_err());
    assert!(package.with_sketches(&owner, 7, 10).is_err());
}
