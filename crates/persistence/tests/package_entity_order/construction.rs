// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn flagged_package() -> DocumentPackage {
    let mut sketches = sketches();
    sketches
        .set_construction(
            DocumentSketchId(7),
            &[SketchEntityId(3), SketchEntityId(1)],
            true,
        )
        .unwrap();
    DocumentPackage::empty(DOCUMENT_PART, "Construction")
        .with_sketches(&sketches, 7, 9)
        .unwrap()
}
#[test]
fn native_yaml_retains_roles_order_and_reservation_marks_in_checked_geometry_version_three() {
    let bytes = flagged_package().encode().unwrap();
    let restored = DocumentPackage::decode(&bytes)
        .unwrap()
        .restore_sketches()
        .unwrap();
    let geometry = restored
        .sketches
        .get(DocumentSketchId(7))
        .unwrap()
        .geometry();
    assert_eq!(geometry.recipe().version, 3);
    assert_eq!(
        geometry.recipe().construction_curves.unwrap(),
        [SketchEntityId(1), SketchEntityId(3)]
    );
    assert!(geometry.profile_curve_ids().is_empty());
    assert_eq!(geometry.entity_ids(), super::geometry().entity_ids());
    assert_stable_encoding(&bytes);
}
fn assert_stable_encoding(bytes: &[u8]) {
    assert_eq!(
        DocumentPackage::decode(bytes).unwrap().encode().unwrap(),
        bytes
    );
}
fn yaml_fixture() -> serde_yaml_ng::Value {
    serde_yaml_ng::from_slice(&flagged_package().encode().unwrap()).unwrap()
}
fn geometry_record(document: &mut serde_yaml_ng::Value) -> &mut serde_yaml_ng::Value {
    &mut document["model"]["planarSketches"]["recipe"]["sketches"][0]["geometry"]
}
fn restore_decline(document: &serde_yaml_ng::Value) -> String {
    let contents = serde_yaml_ng::to_string(document).unwrap();
    DocumentPackage::decode(contents.as_bytes())
        .unwrap()
        .restore_sketches()
        .unwrap_err()
        .to_string()
}
#[test]
fn valid_yaml_with_foreign_or_point_metadata_refuses_owner_restore() {
    for id in [99_u64, 5] {
        let mut document = yaml_fixture();
        geometry_record(&mut document)["construction_curves"] =
            serde_yaml_ng::to_value(vec![id]).unwrap();
        let cause = restore_decline(&document);
        assert!(cause.contains(&id.to_string()), "{cause}");
        assert!(
            cause.contains("expected unique owned line/circle/arc/ellipse IDs"),
            "{cause}"
        );
    }
}
#[test]
fn valid_yaml_with_lower_version_construction_metadata_is_declined() {
    let mut document = yaml_fixture();
    geometry_record(&mut document)["version"] = serde_yaml_ng::to_value(2_u32).unwrap();
    let cause = restore_decline(&document);
    assert!(
        cause.contains("expected no metadata in versions 1/2"),
        "{cause}"
    );
}
#[test]
fn construction_metadata_leaves_saved_identity_reservations_unchanged() {
    let bytes = flagged_package().encode().unwrap();
    let mut restored = DocumentPackage::decode(&bytes)
        .unwrap()
        .restore_sketches()
        .unwrap();
    assert_eq!(restored.sketch_ids.allocate().unwrap(), 8);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 10);
}
