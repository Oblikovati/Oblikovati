// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::package::DocumentPackage;

const GO_PACKAGE: &[u8] = include_bytes!("fixtures/go-package.yaml");

#[test]
fn go_package_sections_survive_native_yaml_round_trip() {
    let package = DocumentPackage::decode(GO_PACKAGE).unwrap();
    assert_eq!(package.manifest().display_name, "fixture Ω");
    assert_eq!(package.manifest().sub_type, "custom");
    let encoded = package.encode().unwrap();
    let restored = DocumentPackage::decode(&encoded).unwrap();
    assert_eq!(restored, package);
    assert_eq!(restored.encode().unwrap(), encoded);
    for name in [
        "identity",
        "resources",
        "model",
        "data",
        "attributes",
        "bodyNames",
    ] {
        assert_eq!(
            restored.section_yaml(name).unwrap(),
            package.section_yaml(name).unwrap()
        );
    }
    assert!(package.section_yaml("absent").unwrap().is_none());
}

#[test]
fn quoted_strings_binary_text_and_native_recipe_shape_are_retained() {
    let package = DocumentPackage::decode(GO_PACKAGE).unwrap();
    let model = String::from_utf8(package.section_yaml("model").unwrap().unwrap()).unwrap();
    assert!(model.contains("parameters:"));
    assert!(model.contains("width: 25.4"));
    assert!(model.contains("label: '001'") || model.contains("label: \"001\""));
    let resources = String::from_utf8(package.section_yaml("resources").unwrap().unwrap()).unwrap();
    assert!(resources.contains("AP8B"));
    assert!(resources.contains("v 0 0 0\n"));
    let streams = String::from_utf8(package.section_yaml("data").unwrap().unwrap()).unwrap();
    assert!(streams.contains("AP+ACg=="));
}

#[test]
fn unsupported_schemas_zip_and_malformed_roots_are_refused() {
    for source in [
        "PK\u{3}\u{4}legacy",
        "schemaVersion: 1\ndocumentType: 1",
        "schemaVersion: 4\ndocumentType: 1",
        "schemaVersion: 2",
        "schemaVersion: 2\ndocumentType: -1",
        "[]",
        "",
        "schemaVersion: [",
    ] {
        let error = DocumentPackage::decode(source.as_bytes()).unwrap_err();
        assert!(error.to_string().contains("expected"));
        assert!(std::error::Error::source(&error).is_none());
    }
}

#[test]
fn duplicates_multiple_documents_unknown_fields_and_wrong_section_shapes_are_refused() {
    let header = "schemaVersion: 2\ndocumentType: 1\n";
    for suffix in [
        "schemaVersion: 2\n",
        "---\nschemaVersion: 2\ndocumentType: 1\n",
        "futureSection: {}\n",
        "model: scalar\n",
        "references: {}\n",
        "42: value\n",
        "model:\n  key: one\n  key: two\n",
    ] {
        assert!(
            DocumentPackage::decode(format!("{header}{suffix}").as_bytes()).is_err(),
            "{suffix}"
        );
    }
}

#[test]
fn every_known_section_has_a_lossless_envelope_path() {
    let mapping = [
        "identity",
        "displaySettings",
        "sketchSettings",
        "bodyNames",
        "bodyColorStyles",
        "resources",
        "model",
        "data",
    ];
    let sequence = [
        "references",
        "attachments",
        "interests",
        "pointClouds",
        "attributes",
    ];
    for (names, value) in [(&mapping[..], "{}"), (&sequence[..], "[]")] {
        for name in names {
            let input = format!("schemaVersion: 2\ndocumentType: 1\n{name}: {value}\n");
            let package = DocumentPackage::decode(input.as_bytes()).unwrap();
            assert_eq!(
                DocumentPackage::decode(&package.encode().unwrap()).unwrap(),
                package
            );
        }
    }
}

#[test]
fn aliases_and_nested_extension_values_keep_their_meaning() {
    let input = b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  first: &shared {label: 'yes', flag: true}\n  second: *shared\n";
    let package = DocumentPackage::decode(input).unwrap();
    assert_eq!(
        DocumentPackage::decode(&package.encode().unwrap()).unwrap(),
        package
    );
    let null =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: null\n").unwrap();
    assert_eq!(
        DocumentPackage::decode(&null.encode().unwrap()).unwrap(),
        null
    );
}
