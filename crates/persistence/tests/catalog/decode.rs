// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::assets::{CatalogSource, decode_catalog_sources};
#[test]
fn malformed_yaml_shapes_and_source_overrides_name_the_input_file() {
    for text in [
        "appearances: [",
        "unrecognized: []",
        "appearances: [8]",
        "appearances: [{id: bad, name: Bad, source: project}]",
        "appearances: [{id: bad, name: A, displayName: B}]",
        "appearances: [{id: bad, name: Bad, base: {weight: .nan}}]",
        "materials: [{id: bad, name: Bad, density: -2}]",
        "materials: [{id: bad, name: Bad, density: no}]",
    ] {
        let refusal = decode_catalog_sources(&[CatalogSource {
            name: "bad.yaml",
            text,
        }])
        .unwrap_err();
        assert!(refusal.contains("bad.yaml"), "{refusal}");
        assert!(refusal.contains("expected"), "{refusal}");
    }
}
#[test]
fn empty_catalog_is_refused_instead_of_returning_a_defaultless_library() {
    assert!(decode_catalog_sources(&[]).unwrap_err().contains("default"));
}
#[test]
fn canonical_name_fields_can_also_decode_without_an_alias() {
    let text = "appearances: [{id: default, displayName: Default, geometry: {opacity: 1}}]";
    let catalog = decode_catalog_sources(&[CatalogSource {
        name: "minimal.yaml",
        text,
    }])
    .unwrap();
    assert_eq!(catalog.appearances().ordered().count(), 1);
}
