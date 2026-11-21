// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    history::SketchReservationMarks,
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchPlane},
};
use oblikovati_persistence::package::DocumentPackage;
use std::collections::BTreeMap;
fn package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 3\ndocumentType: 1\nmodel:\n  parameters: []\n")
        .unwrap()
}
fn sketches() -> DocumentSketches {
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(2), [1.0, 2.0]).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(7),
            PlanarConstraint::Fix {
                point: SketchEntityId(2),
                position: [1.0, 2.0],
            },
        )
        .unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add(DocumentSketchId(1), "Fixed", SketchPlane::xy(), geometry)
        .unwrap();
    sketches
}
fn marks(constraints: &[(u64, u64)]) -> SketchReservationMarks {
    SketchReservationMarks {
        sketch: 3,
        entity: 2,
        constraints: constraints.iter().copied().collect(),
        dimension_names: BTreeMap::new(),
    }
}
fn replace_map(package: &DocumentPackage, mapping: Option<&str>) -> DocumentPackage {
    let mut root: serde_yaml_ng::Value =
        serde_yaml_ng::from_slice(&package.encode().unwrap()).unwrap();
    let collection = root["model"]["planarSketches"].as_mapping_mut().unwrap();
    let key = serde_yaml_ng::Value::String("constraintHighWater".into());
    collection.remove(&key);
    if let Some(mapping) = mapping {
        collection.insert(key, serde_yaml_ng::from_str(mapping).unwrap());
    }
    DocumentPackage::decode(serde_yaml_ng::to_string(&root).unwrap().as_bytes()).unwrap()
}
#[test]
fn explicit_live_and_retired_marks_round_trip_in_stable_order() {
    let expected = BTreeMap::from([(1, 19), (3, 40)]);
    let saved = package()
        .with_sketch_reservations(&sketches(), marks(&[(3, 40), (1, 19)]))
        .unwrap();
    let encoded = saved.encode().unwrap();
    let reopened = DocumentPackage::decode(&encoded).unwrap();
    let mut restored = reopened.restore_sketches().unwrap();
    assert_eq!(restored.constraint_ids.marks(), expected);
    for (sketch, expected) in [(1, 20), (3, 41)] {
        let allocated = restored
            .constraint_ids
            .allocate(DocumentSketchId(sketch), 3)
            .unwrap();
        assert_eq!(allocated.0, expected);
    }
    assert_eq!(reopened.encode().unwrap(), encoded);
}
#[test]
fn strict_capture_refuses_missing_or_short_live_marks() {
    let original = package();
    let retained = original.encode().unwrap();
    for constraints in [vec![], vec![(1, 6)], vec![(0, 1)], vec![(4, 9)]] {
        assert!(
            original
                .with_sketch_reservations(&sketches(), marks(&constraints))
                .is_err()
        );
    }
    assert_eq!(original.encode().unwrap(), retained);
}
#[test]
fn deleting_every_sketch_cannot_drop_or_rewind_constraint_marks() {
    let saved = package()
        .with_sketch_reservations(&sketches(), marks(&[(1, 19), (3, 40)]))
        .unwrap();
    let deleted = saved
        .with_sketch_reservations(&DocumentSketches::default(), marks(&[(1, 19), (3, 40)]))
        .unwrap();
    assert_no_constraint_rewind(&deleted);
    let restored = deleted.restore_sketches().unwrap();
    assert!(restored.sketches.sketches().is_empty());
    assert_eq!(
        restored.constraint_ids.marks(),
        BTreeMap::from([(1, 19), (3, 40)])
    );
}
fn assert_no_constraint_rewind(deleted: &DocumentPackage) {
    for constraints in [
        vec![(1, 19)],
        vec![(1, 18), (3, 40)],
        vec![(1, 19), (3, 39)],
    ] {
        let cause = deleted
            .with_sketch_reservations(&DocumentSketches::default(), marks(&constraints))
            .unwrap_err()
            .to_string();
        assert!(cause.contains("constraint reservation mark"), "{cause}");
    }
}
#[test]
fn standalone_capture_bootstraps_once_and_preserves_retired_marks() {
    let initial = package().with_sketches(&sketches(), 3, 2).unwrap();
    assert_eq!(
        initial.restore_sketches().unwrap().constraint_ids.marks(),
        BTreeMap::from([(1, 7)])
    );
    let reserved = initial
        .with_sketch_reservations(&sketches(), marks(&[(1, 15), (3, 22)]))
        .unwrap();
    let deleted = reserved
        .with_sketches(&DocumentSketches::default(), 3, 2)
        .unwrap();
    assert_eq!(
        deleted.restore_sketches().unwrap().constraint_ids.marks(),
        BTreeMap::from([(1, 15), (3, 22)])
    );
}
#[test]
fn empty_explicit_mapping_is_saved_and_zero_marks_are_normalized() {
    let empty = package()
        .with_sketch_reservations(&DocumentSketches::default(), marks(&[(1, 0)]))
        .unwrap();
    let yaml = String::from_utf8(empty.encode().unwrap()).unwrap();
    assert!(yaml.contains("constraintHighWater: {}"), "{yaml}");
    assert!(
        empty
            .restore_sketches()
            .unwrap()
            .constraint_ids
            .marks()
            .is_empty()
    );
}
#[path = "package_constraint_reservations/malformed.rs"]
mod malformed;
