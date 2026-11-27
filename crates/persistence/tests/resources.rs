// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::package::{DocumentPackage, ResourceContents};

fn document(
    section: &str,
) -> Result<DocumentPackage, oblikovati_persistence::package::PackageCodecError> {
    DocumentPackage::decode(format!("schemaVersion: 2\ndocumentType: 1\n{section}\n").as_bytes())
}

#[test]
fn go_fixture_decodes_exact_resource_and_stream_bytes() {
    let package = DocumentPackage::decode(include_bytes!("fixtures/go-package.yaml")).unwrap();
    assert_eq!(package.streams()["addin/state"], [0, 255, 128, 10]);
    let font = &package.resources()["font"];
    assert_eq!(font.kind, "TrueTypeFont");
    assert_eq!(font.contents, ResourceContents::Binary(vec![0, 255, 1]));
    let mesh = &package.resources()["mesh"];
    assert_eq!(mesh.kind, "ObjFile");
    assert_eq!(
        mesh.contents,
        ResourceContents::Utf8("v 0 0 0\nv 1 0 0\n".into())
    );
}

#[test]
fn encoding_alone_controls_resource_classification_and_origin_is_preserved() {
    let text = document("resources:\n  asset: {type: UnknownExtension, encoding: utf8, value: 'Ω', origin: 'source/part.stl'}").unwrap();
    let resource = &text.resources()["asset"];
    assert_eq!(resource.kind, "UnknownExtension");
    assert_eq!(resource.origin, "source/part.stl");
    assert_eq!(resource.contents, ResourceContents::Utf8("Ω".into()));
    let binary =
        document("resources:\n  asset: {type: ObjFile, encoding: base64, value: '/w=='}").unwrap();
    assert_eq!(
        binary.resources()["asset"].contents,
        ResourceContents::Binary(vec![255])
    );
    let embedded =
        document("resources:\n  asset: {type: ObjFile, encoding: embedded, origin: builtin}")
            .unwrap();
    assert_eq!(
        embedded.resources()["asset"].contents,
        ResourceContents::Embedded
    );
    assert_eq!(embedded.resources()["asset"].origin, "builtin");
}

#[test]
fn empty_missing_and_null_tables_have_no_entries() {
    for section in ["", "data: {}\nresources: {}", "data: null\nresources: null"] {
        let package = document(section).unwrap();
        assert!(package.streams().is_empty());
        assert!(package.resources().is_empty());
    }
    let package = document("data: {empty: ''}\nresources: {empty: {encoding: base64}}").unwrap();
    assert!(package.streams()["empty"].is_empty());
    assert_eq!(
        package.resources()["empty"].contents,
        ResourceContents::Binary(vec![])
    );
}

#[test]
fn base64_matches_go_newline_and_unused_bit_rules() {
    for value in ["Zg==", "Zh==", "Zg==\\r\\n", "Z\\ng=="] {
        let package = document(&format!("data: {{stream: \"{value}\"}}")).unwrap();
        assert_eq!(package.streams()["stream"], b"f");
    }
    for invalid in ["Zg", "Zg=", "Z g==", "Zg===", "!!!!", "_w=="] {
        let error = document(&format!("data: {{stream: '{invalid}'}}")).unwrap_err();
        assert!(error.to_string().contains("stream"));
        assert!(error.to_string().contains(invalid));
        assert!(
            error
                .to_string()
                .contains("expected padded standard base64")
        );
    }
}

#[test]
fn invalid_resources_are_refused_before_a_package_is_returned() {
    for fields in [
        "encoding: future, value: abc",
        "encoding: base64, value: '??'",
        "encoding: embedded, value: discarded",
        "type: ObjFile",
        "encoding: utf8, value: []",
    ] {
        assert!(
            document(&format!("resources: {{asset: {{{fields}}}}}")).is_err(),
            "{fields}"
        );
    }
    assert!(document("resources: {asset: []}").is_err());
    assert!(document("data: {stream: []}").is_err());
}

#[test]
fn typed_decoding_does_not_strip_resource_extension_fields() {
    let package = document(
        "resources:\n  asset: {encoding: utf8, value: 'text', futureMetadata: {revision: 4}}",
    )
    .unwrap();
    let encoded = package.encode().unwrap();
    assert!(
        String::from_utf8(encoded.clone())
            .unwrap()
            .contains("futureMetadata")
    );
    let restored = DocumentPackage::decode(&encoded).unwrap();
    assert_eq!(restored, package);
    assert_eq!(restored.resources(), package.resources());
}
