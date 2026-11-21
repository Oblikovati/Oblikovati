// SPDX-License-Identifier: GPL-2.0-only
use super::xml_document::BlenderThemeDocument;

#[test]
fn oversized_and_deep_theme_documents_are_refused_before_mapping() {
    let oversized = vec![b' '; 16 * 1024 * 1024 + 1];
    let error = BlenderThemeDocument::parse(&oversized).unwrap_err();
    assert!(error.contains("16777217"));
    assert!(error.contains("16777216"));
    let depth = format!("{}{}", "<n>".repeat(65), "</n>".repeat(65));
    let error = BlenderThemeDocument::parse(depth.as_bytes()).unwrap_err();
    assert!(error.contains("depth 65"));
    assert!(error.contains("64"));
}

#[test]
fn depth_limit_and_first_child_paths_include_only_direct_children() {
    let depth = format!("{}{}", "<n>".repeat(64), "</n>".repeat(64));
    assert!(BlenderThemeDocument::parse(depth.as_bytes()).is_ok());
    let document = BlenderThemeDocument::parse(
        b"<root><wrapper><slot color='nested'/></wrapper><slot color='first'/><slot color='second'/></root>",
    ).unwrap();
    assert_eq!(document.attribute(&["slot"], "color"), Some("first"));
    assert_eq!(
        document.attribute(&["wrapper", "slot"], "color"),
        Some("nested")
    );
    assert_eq!(document.attribute(&["absent"], "color"), None);
}

#[test]
fn attribute_updates_and_inserts_preserve_qualified_unknown_attributes() {
    let mut document = BlenderThemeDocument::parse(
        b"<root xmlns:v='urn:vendor'><slot v:extra='keep' color='old'/></root>",
    )
    .unwrap();
    document
        .set_attribute(&["slot"], "color", "new".into())
        .unwrap();
    document
        .set_attribute(&["slot"], "added", "yes".into())
        .unwrap();
    assert!(
        document
            .set_attribute(&["absent"], "color", "no".into())
            .is_err()
    );
    let bytes = document.encode("Workshop", Vec::new()).unwrap();
    let saved = String::from_utf8(bytes.clone()).unwrap();
    assert!(saved.contains("v:extra=\"keep\""));
    let restored = BlenderThemeDocument::parse(&bytes).unwrap();
    assert_eq!(restored.attribute(&["slot"], "color"), Some("new"));
    assert_eq!(restored.attribute(&["slot"], "added"), Some("yes"));
}

#[test]
fn snapshot_replacement_removes_duplicates_but_keeps_nested_vendor_elements() {
    let document = BlenderThemeDocument::parse(
        b"<root><oblikovati_tokens name='old'/><vendor><oblikovati_tokens name='vendor'/></vendor><oblikovati_tokens name='stale'/></root>",
    ).unwrap();
    let encoded = document
        .encode("Workshop", vec![("chrome.text".into(), "#123456ff".into())])
        .unwrap();
    let restored = BlenderThemeDocument::parse(&encoded).unwrap();
    assert_eq!(restored.snapshot().get("name").unwrap(), "Workshop");
    assert_eq!(restored.snapshot().get("chrome.text").unwrap(), "#123456ff");
    assert_eq!(
        restored.attribute(&["vendor", "oblikovati_tokens"], "name"),
        Some("vendor")
    );
    let text = String::from_utf8(encoded).unwrap();
    assert!(!text.contains("stale"));
    assert!(!text.contains("name=\"old\""));
}
