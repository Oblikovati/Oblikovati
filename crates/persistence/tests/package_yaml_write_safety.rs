// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::package::DocumentPackage;

#[test]
fn ambiguous_yaml_merge_keys_cannot_change_meaning_when_saved() {
    for field in [
        "model: {custom: {'<<': {x: 1}}}",
        "model: {custom: {!!str <<: {x: 1}}}",
        "model: {custom: {<<: {x: 1}}}",
        "model: {custom: [{inner: {'<<': {x: 1}}}]}",
        "attributes: [{'<<': {x: 1}}]",
    ] {
        let source = format!("schemaVersion: 2\ndocumentType: 1\n{field}\n");
        let package = DocumentPackage::decode(source.as_bytes()).unwrap();
        let section = if field.starts_with("model:") {
            "model"
        } else {
            "attributes"
        };
        assert!(
            package
                .section_yaml(section)
                .unwrap_err()
                .to_string()
                .contains("<<")
        );
        let error = package.encode().unwrap_err();
        assert!(error.to_string().contains("<<"), "{field}: {error}");
    }
}

#[test]
fn literal_merge_text_as_a_value_remains_writable() {
    let source = b"schemaVersion: 2\ndocumentType: 1\nmodel: {custom: {description: '<<'}}\n";
    let package = DocumentPackage::decode(source).unwrap();
    let reopened = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
    assert_eq!(reopened, package);
}
