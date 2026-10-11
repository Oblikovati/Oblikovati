// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn encoded_with_live_constraint(mapping: Option<&str>) -> DocumentPackage {
    let initial = package()
        .with_sketch_reservations(&DocumentSketches::default(), marks(&[]))
        .unwrap();
    let base = replace_map(&initial, mapping);
    let mut root: serde_yaml_ng::Value =
        serde_yaml_ng::from_slice(&base.encode().unwrap()).unwrap();
    root["model"]["planarSketches"]["recipe"] =
        serde_yaml_ng::to_value(sketches().recipe()).unwrap();
    DocumentPackage::decode(serde_yaml_ng::to_string(&root).unwrap().as_bytes()).unwrap()
}
#[test]
fn missing_old_metadata_bootstraps_live_ids_but_explicit_empty_metadata_refuses() {
    let legacy = encoded_with_live_constraint(None);
    assert_eq!(
        legacy.restore_sketches().unwrap().constraint_ids.marks(),
        BTreeMap::from([(1, 7)])
    );
    for mapping in ["{}", "{1: 6}", "{2: 7}"] {
        assert!(
            encoded_with_live_constraint(Some(mapping))
                .restore_sketches()
                .is_err()
        );
    }
}
#[test]
fn invalid_stored_constraint_namespace_or_mark_is_refused() {
    let empty = package()
        .with_sketch_reservations(&DocumentSketches::default(), marks(&[]))
        .unwrap();
    for mapping in [
        "{0: 1}",
        "{4: 1}",
        "{1: -1}",
        "{1: broken}",
        "[]",
        "false",
        "null",
    ] {
        let modified = replace_map(&empty, Some(mapping));
        assert!(
            modified.restore_sketches().is_err(),
            "expected invalid map {mapping}"
        );
    }
}
