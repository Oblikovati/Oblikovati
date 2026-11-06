// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::package::DocumentPackage;

fn package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {parameters: []}\nattributes: [0, 0, 0, 0]\ndata: {retained: \"Zg==\\n\"}\nresources: {image: {type: extension, encoding: utf8, value: text, custom: keep}}\n").unwrap()
}

#[test]
fn stream_replacement_preserves_other_sections_and_owns_its_input() {
    let original = package();
    let mut bytes = vec![0, 255, 42];
    let updated = original.with_stream("addin/state", &bytes);
    bytes[0] = 99;
    assert_eq!(updated.streams()["addin/state"], [0, 255, 42]);
    assert!(!original.streams().contains_key("addin/state"));
    assert_eq!(
        updated.recipe_digest().unwrap(),
        original.recipe_digest().unwrap()
    );
    for section in ["model", "attributes", "resources"] {
        assert_eq!(
            updated.section_yaml(section).unwrap(),
            original.section_yaml(section).unwrap()
        );
    }
    assert_eq!(
        DocumentPackage::decode(&updated.encode().unwrap()).unwrap(),
        updated
    );
    let replaced = updated.with_stream("addin/state", &[8]);
    assert_eq!(replaced.streams()["addin/state"], [8]);
    assert_eq!(updated.streams()["addin/state"], [0, 255, 42]);
    assert!(
        String::from_utf8(replaced.section_yaml("data").unwrap().unwrap())
            .unwrap()
            .contains("Zg==")
    );
}

#[test]
fn empty_payloads_and_arbitrary_names_round_trip_and_missing_removal_is_a_noop() {
    let mut package =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\ndata: null\n").unwrap();
    assert_eq!(package.without_stream("missing"), package);
    for name in ["", "../relative", "µm", "a\0b", "cache"] {
        package = package.with_stream(name, &[]);
    }
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    assert_eq!(reopened, package);
    for name in ["", "../relative", "µm", "a\0b", "cache"] {
        package = package.without_stream(name);
    }
    assert!(package.streams().is_empty());
    assert!(package.section_yaml("data").unwrap().is_none());
}

#[test]
fn compaction_removes_only_exact_cache_prefix_and_preserves_recipe() {
    let original = package()
        .with_stream("cache/tessellation", &[1; 4096])
        .with_stream("cache/preview", &[2; 1024])
        .with_stream("Cache/keep", &[3])
        .with_stream("cache", &[4])
        .with_stream("addin/cache/value", &[5]);
    let (compacted, reclaimed) = original.without_caches().unwrap();
    assert_eq!(reclaimed, 5120);
    assert_eq!(original.streams().len() - compacted.streams().len(), 2);
    for name in ["retained", "Cache/keep", "cache", "addin/cache/value"] {
        assert_eq!(compacted.streams()[name], original.streams()[name]);
    }
    assert_eq!(
        compacted.recipe_digest().unwrap(),
        original.recipe_digest().unwrap()
    );
    assert_eq!(compacted.without_caches().unwrap(), (compacted.clone(), 0));
    assert_eq!(
        DocumentPackage::decode(&compacted.encode().unwrap()).unwrap(),
        compacted
    );
}
