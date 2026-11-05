// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn axis_sketches(centerline: bool, construction: bool) -> DocumentSketches {
    let mut collection = sketches();
    let id = DocumentSketchId(3);
    let mut geometry = collection.get(id).unwrap().geometry().clone();
    geometry
        .set_curve_construction(&[SketchEntityId(7)], construction)
        .unwrap();
    geometry
        .set_line_centerline(SketchEntityId(7), centerline)
        .unwrap();
    collection.replace_geometry(id, geometry).unwrap();
    collection
}
#[test]
fn native_centerline_roles_round_trip_without_changing_document_schema_or_other_sections() {
    let original = package();
    let saved = original
        .with_sketches(&axis_sketches(true, false), 9, 15)
        .unwrap();
    assert_eq!(saved.manifest().schema_version, 3);
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    let restored = reopened.restore_sketches().unwrap();
    let placed = restored.sketches.get(DocumentSketchId(3)).unwrap();
    let geometry = placed.geometry();
    assert_eq!(geometry.recipe().version, 4);
    assert_eq!(
        geometry.centerline_axis().unwrap().unwrap().id,
        SketchEntityId(7)
    );
    assert!(geometry.profile_curve_ids().is_empty());
    assert_display_untouched(&reopened, &original);
}
#[test]
fn construction_and_centerline_remain_independent_after_native_restore() {
    for (centerline, construction) in [(false, false), (true, false), (false, true), (true, true)] {
        assert_restored_roles(centerline, construction);
    }
}
fn assert_restored_roles(centerline: bool, construction: bool) {
    let expected = axis_sketches(centerline, construction);
    let saved = package().with_sketches(&expected, 9, 15).unwrap();
    let restored = saved.restore_sketches().unwrap();
    assert!(restored.sketches.same_recipe(&expected));
    let mut geometry = restored
        .sketches
        .get(DocumentSketchId(3))
        .unwrap()
        .geometry()
        .clone();
    geometry
        .set_line_centerline(SketchEntityId(7), false)
        .unwrap();
    assert_eq!(
        geometry.is_construction(SketchEntityId(7)).unwrap(),
        construction
    );
}
#[test]
fn malformed_centerline_metadata_is_refused_through_the_package_boundary() {
    let saved = package()
        .with_sketches(&axis_sketches(true, false), 9, 15)
        .unwrap();
    for ids in ["[7, 7]", "[2]", "[99]", "null", "false"] {
        let mut root: serde_yaml_ng::Value =
            serde_yaml_ng::from_slice(&saved.encode().unwrap()).unwrap();
        root["model"]["planarSketches"]["recipe"]["sketches"][0]["geometry"]["centerlines"] =
            serde_yaml_ng::from_str(ids).unwrap();
        let modified =
            DocumentPackage::decode(serde_yaml_ng::to_string(&root).unwrap().as_bytes()).unwrap();
        assert!(
            modified.restore_sketches().is_err(),
            "expected refusal for centerline IDs {ids}"
        );
    }
}
fn assert_display_untouched(actual: &DocumentPackage, expected: &DocumentPackage) {
    let display = actual.section_yaml("displaySettings").unwrap();
    assert_eq!(display, expected.section_yaml("displaySettings").unwrap());
}
