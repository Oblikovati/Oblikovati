// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::package::{DocumentPackage, PackageResource, ResourceContents};

fn package() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {retained: [1, 2]}\ndata: {client: AP8=}\nresources: {asset: {type: SameType, origin: old, encoding: base64, value: AP8=, extension: {revision: 7}}, keep: {encoding: utf8, value: preserved}}\n").unwrap()
}

#[test]
fn every_content_transition_round_trips_and_preserves_extension_metadata() {
    let contents = [
        ResourceContents::Utf8("µm\0\n".into()),
        ResourceContents::Binary(vec![0, 255]),
        ResourceContents::Embedded,
    ];
    for source in &contents {
        for target in &contents {
            let original = package().with_resource(
                "asset",
                &PackageResource {
                    kind: "SameType".into(),
                    origin: "original".into(),
                    contents: source.clone(),
                },
            );
            let resource = PackageResource {
                kind: "OtherType".into(),
                origin: String::new(),
                contents: target.clone(),
            };
            let updated = original.with_resource("asset", &resource);
            let restored = DocumentPackage::decode(&updated.encode().unwrap()).unwrap();
            assert_eq!(restored.resources()["asset"], resource);
            assert_eq!(restored.resources()["keep"], original.resources()["keep"]);
            assert_eq!(
                restored.recipe_digest().unwrap(),
                original.recipe_digest().unwrap()
            );
            assert_eq!(restored.streams(), original.streams());
            let section: serde_yaml_ng::Value =
                serde_yaml_ng::from_slice(&restored.section_yaml("resources").unwrap().unwrap())
                    .unwrap();
            assert_eq!(section["asset"]["extension"]["revision"].as_i64(), Some(7));
            assert!(section["asset"].get("origin").is_none());
            assert_eq!(
                section["asset"].get("value").is_none(),
                *target == ResourceContents::Embedded
            );
            assert_eq!(original.resources()["asset"].contents, *source);
        }
    }
}

#[test]
fn resources_are_owned_and_empty_payloads_remain_distinct_from_app_references() {
    let original =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nresources: null\n").unwrap();
    let mut resource = PackageResource {
        kind: "Extension".into(),
        origin: "source".into(),
        contents: ResourceContents::Binary(vec![1, 2]),
    };
    let updated = original.with_resource("../opaque\0", &resource);
    resource.contents = ResourceContents::Binary(Vec::new());
    assert_eq!(
        updated.resources()["../opaque\0"].contents,
        ResourceContents::Binary(vec![1, 2])
    );
    assert!(original.resources().is_empty());
    for contents in [
        ResourceContents::Binary(Vec::new()),
        ResourceContents::Utf8(String::new()),
        ResourceContents::Embedded,
    ] {
        resource.contents = contents;
        let encoded = updated.with_resource("", &resource).encode().unwrap();
        assert_eq!(
            DocumentPackage::decode(&encoded).unwrap().resources()[""],
            resource
        );
    }
}

#[test]
fn resource_removal_is_local_and_absent_removal_preserves_representation() {
    let original = package();
    assert_eq!(original.without_resource("absent"), original);
    let removed = original.without_resource("asset");
    assert_eq!(removed.resources().len(), 1);
    assert!(!removed.resources().contains_key("asset"));
    assert_eq!(removed.streams(), original.streams());
    let empty = removed.without_resource("keep");
    assert!(empty.resources().is_empty());
    assert!(empty.section_yaml("resources").unwrap().is_none());
    assert_eq!(
        DocumentPackage::decode(&empty.encode().unwrap()).unwrap(),
        empty
    );
    assert_eq!(original.resources().len(), 2);
}

#[test]
fn captured_resources_match_the_go_fixture_yaml_values() {
    let reference = DocumentPackage::decode(include_bytes!("fixtures/go-package.yaml")).unwrap();
    let mut captured = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    for (id, resource) in reference.resources() {
        captured = captured.with_resource(id, resource);
    }
    let values = |package: &DocumentPackage| -> serde_yaml_ng::Value {
        serde_yaml_ng::from_slice(&package.section_yaml("resources").unwrap().unwrap()).unwrap()
    };
    assert_eq!(values(&captured), values(&reference));
    assert_eq!(captured.resources(), reference.resources());
}

#[test]
fn snapshot_capture_updates_only_changed_records_and_preserves_extensions() {
    let original = package();
    assert_eq!(original.with_resources(original.resources()), original);
    let mut resources = original.resources().clone();
    resources.get_mut("asset").unwrap().contents = ResourceContents::Embedded;
    resources.insert(
        "new".into(),
        PackageResource::imported("Custom", "source", &[0, 255]),
    );
    let captured = original.with_resources(&resources);
    let decoded = DocumentPackage::decode(&captured.encode().unwrap()).unwrap();
    assert_eq!(decoded.resources(), &resources);
    let section: serde_yaml_ng::Value =
        serde_yaml_ng::from_slice(&decoded.section_yaml("resources").unwrap().unwrap()).unwrap();
    assert_eq!(section["asset"]["extension"]["revision"].as_i64(), Some(7));
    resources.remove("keep");
    assert!(
        !captured
            .with_resources(&resources)
            .resources()
            .contains_key("keep")
    );
    let cleared = captured.with_resources(&Default::default());
    assert!(cleared.section_yaml("resources").unwrap().is_none());
    assert_eq!(
        cleared.recipe_digest().unwrap(),
        original.recipe_digest().unwrap()
    );
    assert_eq!(cleared.streams(), original.streams());
}

#[test]
fn changing_other_resources_keeps_an_unchanged_record_representation() {
    let original = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nresources: {keep: {encoding: base64, value: \"Zg==\\n\", extension: keep}}\n").unwrap();
    let mut resources = original.resources().clone();
    resources.insert("new".into(), PackageResource::imported("File", "", b"new"));
    let captured = original.with_resources(&resources);
    let section: serde_yaml_ng::Value =
        serde_yaml_ng::from_slice(&captured.section_yaml("resources").unwrap().unwrap()).unwrap();
    assert_eq!(section["keep"]["value"].as_str(), Some("Zg==\n"));
    assert!(section["keep"].get("type").is_none());
}
