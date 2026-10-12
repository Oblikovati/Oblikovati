// SPDX-License-Identifier: GPL-2.0-only
use super::fixture::*;
#[test]
fn malformed_assets_and_assignment_references_are_refused_before_adoption() {
    for section in [
        "materials: [4]",
        "materials: [{id: x, name: X, density: .nan}]",
        "appearances: [{id: old, albedo: '#zzzzzz'}]",
        "assignments: {bodyMaterial: {body: 4}}",
        "assignments: {bodyMaterial: {'': steel}}",
        "assignments: {bodyMaterial: {? {referenceKey: body, occurrencePath: [left]}: steel}}",
    ] {
        let package = package_with_model(&format!("  materials: {{{section}}}"));
        assert!(
            package.restore_assets().is_err(),
            "section {section}: expected admission failure"
        );
        let before = package.clone();
        assert!(package.with_assets(&Default::default()).is_err());
        assert_eq!(package, before);
    }
}
#[test]
fn duplicate_asset_definitions_and_alias_conflicts_are_named() {
    let saved = empty_package().with_assets(&owned_assets()).unwrap();
    let mut raw = raw_package(&saved);
    let material = raw["model"]["materials"]["materials"][0].clone();
    raw["model"]["materials"]["materials"]
        .as_sequence_mut()
        .unwrap()
        .push(material);
    assert!(
        yaml_package(&raw)
            .restore_assets()
            .unwrap_err()
            .to_string()
            .contains("unique")
    );
    let conflict =
        package_with_model("  materials: {materials: [{id: x, name: A, displayName: B}]}");
    assert!(
        conflict
            .restore_assets()
            .unwrap_err()
            .to_string()
            .contains("only one")
    );
}

#[test]
fn semantically_duplicate_target_forms_and_bad_occurrence_components_are_refused() {
    for path in ["[]", "['']", "[left]"] {
        let section = format!(
            "schemaVersion: 3\ndocumentType: 1\nmodel:\n  materials:\n    assignments:\n      bodyMaterial:\n        body: steel\n        ? referenceKey: body\n          occurrencePath: {path}\n        : oak\n"
        );
        let package =
            oblikovati_persistence::package::DocumentPackage::decode(section.as_bytes()).unwrap();
        if path == "[left]" {
            assert!(package.restore_assets().is_ok());
        } else {
            assert!(package.restore_assets().is_err());
        }
    }
}
