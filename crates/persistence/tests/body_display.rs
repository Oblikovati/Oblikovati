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

#[test]
fn authored_body_names_round_trip_without_changing_other_sections_or_legacy_nulls() {
    let original = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nbodyNames: {old: Before, blank: '', unused: null}\nbodyColorStyles: {old: Blue}\nmodel: {opaque: retained}\n").unwrap();
    let mut names = original.restore_body_names().unwrap();
    assert_eq!(names.len(), 1);
    assert_eq!(original.with_body_names(&names).unwrap(), original);
    names.insert("old".into(), " Bracket α ".into());
    names.insert("new".into(), "Second".into());
    let captured = original.with_body_names(&names).unwrap();
    assert_eq!(captured.restore_body_names().unwrap(), names);
    assert_eq!(
        captured.recipe_digest().unwrap(),
        original.recipe_digest().unwrap()
    );
    assert_eq!(
        captured.section_yaml("bodyColorStyles").unwrap(),
        original.section_yaml("bodyColorStyles").unwrap()
    );
    assert!(
        String::from_utf8(captured.section_yaml("bodyNames").unwrap().unwrap())
            .unwrap()
            .contains("unused: null")
    );
    let reopened = DocumentPackage::decode(&captured.encode().unwrap()).unwrap();
    assert_eq!(reopened.restore_body_names().unwrap(), names);
    let cleared = captured.with_body_names(&Default::default()).unwrap();
    assert!(cleared.restore_body_names().unwrap().is_empty());
    assert_eq!(original.restore_body_names().unwrap()["old"], "Before");
}

#[test]
fn malformed_body_names_fail_without_mutating_the_package() {
    for entry in ["42: Name", "key: 42", "key: [Name]"] {
        let bytes = format!("schemaVersion: 2\ndocumentType: 1\nbodyNames: {{{entry}}}\n");
        let package = DocumentPackage::decode(bytes.as_bytes()).unwrap();
        assert!(
            package
                .restore_body_names()
                .unwrap_err()
                .to_string()
                .contains("expected")
        );
        assert!(package.with_body_names(&Default::default()).is_err());
    }
    let package =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nbodyNames: null\n").unwrap();
    assert!(package.restore_body_names().unwrap().is_empty());
    let empty_override = [("key".into(), String::new())].into_iter().collect();
    let error = package
        .with_body_names(&empty_override)
        .unwrap_err()
        .to_string();
    assert!(error.contains("key") && error.contains("nonempty"));
    assert!(package.restore_body_names().unwrap().is_empty());
}
