// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::package::DocumentPackage;
#[test]
fn body_display_metadata_preserves_names_styles_and_absence() {
    let bytes = b"schemaVersion: 2\ndocumentType: 1\nbodyNames: {key: Bracket, empty: '', null: null}\nbodyColorStyles: {key: blue}\n";
    let package = DocumentPackage::decode(bytes).unwrap();
    for package in [
        &package,
        &DocumentPackage::decode(&package.encode().unwrap()).unwrap(),
    ] {
        assert_eq!(
            package.body_display_metadata("key").unwrap(),
            (Some("Bracket"), Some("blue"))
        );
        for key in ["missing", "empty", "null"] {
            assert_eq!(package.body_display_metadata(key).unwrap(), (None, None));
        }
    }
}
#[test]
fn invalid_body_display_values_are_reported_with_the_reference_and_value() {
    for section in ["bodyNames", "bodyColorStyles"] {
        let bytes = format!("schemaVersion: 2\ndocumentType: 1\n{section}: {{bad: 42}}\n");
        let package = DocumentPackage::decode(bytes.as_bytes()).unwrap();
        let error = package
            .body_display_metadata("bad")
            .unwrap_err()
            .to_string();
        assert!(
            error.contains(section)
                && error.contains("bad")
                && error.contains("42")
                && error.contains("expected")
        );
    }
}
