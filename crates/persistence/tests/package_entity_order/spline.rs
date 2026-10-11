// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DocumentType};
use oblikovati_model::sketch::{SketchOffsetSpline2D, SketchSpline2D, SketchSplineKind};
fn spline_sketches() -> DocumentSketches {
    let mut owner = sketches();
    let mut geometry = geometry();
    geometry.add_point(SketchEntityId(12), [1.0, 2.0]).unwrap();
    add_package_spline_pair(&mut geometry);
    geometry
        .set_curve_construction(&[SketchEntityId(10)], true)
        .unwrap();
    owner
        .replace_geometry(DocumentSketchId(7), geometry)
        .unwrap();
    owner
}
#[test]
fn native_packages_preserve_parent_and_offset_ids_roles_and_signed_distance() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_spline_package(kind);
    }
}
fn assert_spline_package(kind: DocumentType) {
    let owner = spline_sketches();
    let package = DocumentPackage::empty(kind, "Splines")
        .with_sketches(&owner, 7, 12)
        .unwrap();
    let bytes = package.encode().unwrap();
    let decoded = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = decoded.restore_sketches().unwrap();
    assert!(restored.sketches.same_recipe(&owner));
    assert_eq!(restored.entity_ids.allocate().unwrap(), 13);
    assert_eq!(decoded.encode().unwrap(), bytes);
    assert_package_spline_geometry(&restored.sketches);
}
fn assert_package_spline_geometry(sketches: &DocumentSketches) {
    let geometry = sketches.get(DocumentSketchId(7)).unwrap().geometry();
    assert_eq!(geometry.recipe().version, 10);
    assert!(geometry.is_construction(SketchEntityId(10)).unwrap());
    assert_eq!(
        geometry.offset_spline(SketchEntityId(11)).unwrap().parent(),
        SketchEntityId(10)
    );
    assert_eq!(
        geometry
            .offset_spline(SketchEntityId(11))
            .unwrap()
            .distance(),
        -2.0
    );
}
#[test]
fn reservation_marks_cannot_drop_parent_offset_or_defining_point_ids() {
    let owner = spline_sketches();
    let package = DocumentPackage::empty(DOCUMENT_PART, "Splines");
    for mark in [9, 10, 11] {
        assert!(package.with_sketches(&owner, 7, mark).is_err());
    }
}

fn add_package_spline_pair(geometry: &mut Sketch2D) {
    let parent = SketchSpline2D::new(
        SketchEntityId(10),
        vec![SketchEntityId(5), SketchEntityId(12), SketchEntityId(9)],
        true,
        SketchSplineKind::Control,
    )
    .unwrap();
    geometry.add_spline(parent).unwrap();
    geometry
        .add_offset_spline(
            SketchOffsetSpline2D::new(SketchEntityId(11), SketchEntityId(10), -2.0).unwrap(),
        )
        .unwrap();
}
