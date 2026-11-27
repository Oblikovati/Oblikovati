// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::{DocumentGuid, GuidSource, IdentityTransitionError};
use oblikovati_persistence::package::{
    DocumentPackage, PackageResource, ResourceContents, ResourceImportError,
};
use std::error::Error;

struct ResourceGuids {
    calls: usize,
    fail: bool,
    repeated: bool,
}
impl GuidSource for ResourceGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.calls += 1;
        if self.fail {
            return Err(IdentityTransitionError::Randomness(
                "fixture exhausted".into(),
            ));
        }
        let suffix = if self.repeated { 1 } else { self.calls };
        Ok(format!("00000000-0000-4000-8000-{suffix:012x}")
            .parse()
            .unwrap())
    }
}

fn source() -> ResourceGuids {
    ResourceGuids {
        calls: 0,
        fail: false,
        repeated: false,
    }
}

fn empty() -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel: {retain: [1, 2]}\n")
        .unwrap()
}

#[test]
fn import_encoding_preserves_exact_bytes_and_owns_the_input() {
    for (bytes, text) in [
        (b"".as_slice(), true),
        ("µm\n".as_bytes(), true),
        (b"\0", false),
        (b"\xff", false),
    ] {
        let mut input = bytes.to_vec();
        let resource = PackageResource::imported("Custom", "/missing/path", &input);
        input.clear();
        let encoded = empty().with_resource("asset", &resource).encode().unwrap();
        let restored = DocumentPackage::decode(&encoded).unwrap();
        let contents = &restored.resources()["asset"].contents;
        assert_eq!(matches!(contents, ResourceContents::Utf8(_)), text);
        match contents {
            ResourceContents::Utf8(value) => assert_eq!(value.as_bytes(), bytes),
            ResourceContents::Binary(value) => assert_eq!(value, bytes),
            ResourceContents::Embedded => panic!("imports must retain bytes"),
        }
    }
}

#[test]
fn identical_imports_reuse_existing_ids_without_entropy_even_after_reopen() {
    let original = empty();
    let resource = PackageResource::imported("ObjFile", "mesh.obj", b"v 0 0 0\n");
    let mut guids = source();
    let (imported, id) = original
        .with_imported_resource(&resource, &mut guids)
        .unwrap();
    assert_eq!(guids.calls, 1);
    assert!(original.resources().is_empty());
    assert_eq!(
        imported.recipe_digest().unwrap(),
        original.recipe_digest().unwrap()
    );
    guids.fail = true;
    let reopened = DocumentPackage::decode(&imported.encode().unwrap()).unwrap();
    let (unchanged, repeated) = reopened
        .with_imported_resource(&resource, &mut guids)
        .unwrap();
    assert_eq!(repeated, id);
    assert_eq!(unchanged, reopened);
    assert_eq!(guids.calls, 1);
    let duplicates = imported
        .with_resource("z-legacy", &resource)
        .with_resource("a-legacy", &resource)
        .without_resource(&id);
    let (retained, chosen) = duplicates
        .with_imported_resource(&resource, &mut guids)
        .unwrap();
    assert_eq!(chosen, "a-legacy");
    assert_eq!(retained, duplicates);
    assert_eq!(guids.calls, 1);
}

#[test]
fn type_origin_encoding_and_payload_all_participate_in_import_identity() {
    let mut package = empty();
    let mut guids = source();
    let base = PackageResource::imported("Font", "origin", b"abc");
    let variants = [
        base.clone(),
        PackageResource {
            kind: "Other".into(),
            ..base.clone()
        },
        PackageResource {
            origin: "other".into(),
            ..base.clone()
        },
        PackageResource {
            contents: ResourceContents::Binary(b"abc".to_vec()),
            ..base.clone()
        },
        PackageResource {
            contents: ResourceContents::Utf8("abcd".into()),
            ..base.clone()
        },
        PackageResource {
            contents: ResourceContents::Embedded,
            ..base
        },
    ];
    for resource in variants {
        let (next, id) = package
            .with_imported_resource(&resource, &mut guids)
            .unwrap();
        assert!(!package.resources().contains_key(&id));
        assert_eq!(next.resources()[&id], resource);
        let (same, reused) = next.with_imported_resource(&resource, &mut guids).unwrap();
        assert_eq!(reused, id);
        assert_eq!(same, next);
        package = next;
    }
    assert_eq!(package.resources().len(), 6);
    assert_eq!(guids.calls, 6);
}

#[test]
fn entropy_failure_and_collisions_preserve_the_package_without_retrying() {
    let mut guids = source();
    let first = PackageResource::imported("File", "origin", b"first");
    let (package, _) = empty().with_imported_resource(&first, &mut guids).unwrap();
    let before = package.clone();
    let next = PackageResource::imported("File", "origin", b"next");
    guids.repeated = true;
    let error = package
        .with_imported_resource(&next, &mut guids)
        .unwrap_err();
    assert!(matches!(error, ResourceImportError::Collision(_)));
    assert!(
        error
            .to_string()
            .contains("00000000-0000-4000-8000-000000000001")
    );
    assert!(error.source().is_none());
    assert_eq!(guids.calls, 2);
    guids.fail = true;
    let error = package
        .with_imported_resource(&next, &mut guids)
        .unwrap_err();
    assert!(error.to_string().contains("fixture exhausted"));
    assert!(error.source().unwrap().is::<IdentityTransitionError>());
    assert_eq!(guids.calls, 3);
    assert_eq!(package, before);
}
