// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::package::{
    ApplicationResources, DocumentPackage, PackageResource, ResourceContents, ResourceReadError,
};
use std::{borrow::Cow, error::Error, io};

struct FakeApplicationResources {
    requests: Vec<(String, String)>,
    payload: Vec<u8>,
    failure: Option<io::ErrorKind>,
}
impl ApplicationResources for FakeApplicationResources {
    fn read_resource(&mut self, kind: &str, origin: &str) -> io::Result<Vec<u8>> {
        self.requests.push((kind.into(), origin.into()));
        if let Some(failure) = self.failure {
            return Err(io::Error::new(failure, "bundle fixture failure"));
        }
        Ok(self.payload.clone())
    }
}

fn provider(failure: Option<io::ErrorKind>) -> FakeApplicationResources {
    FakeApplicationResources {
        requests: Vec::new(),
        payload: vec![0, 255, 1],
        failure,
    }
}

fn package(contents: ResourceContents) -> DocumentPackage {
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_resource(
            "asset",
            &PackageResource {
                kind: "TrueTypeFont".into(),
                origin: "/missing/import/or/bundle/key".into(),
                contents,
            },
        )
}

#[test]
fn stored_resources_resolve_exact_bytes_without_consulting_the_bundle_provider() {
    let cases = [
        (
            ResourceContents::Utf8("µm\0\n".into()),
            "µm\0\n".as_bytes().to_vec(),
        ),
        (ResourceContents::Binary(vec![0, 255]), vec![0, 255]),
        (ResourceContents::Utf8(String::new()), Vec::new()),
        (ResourceContents::Binary(Vec::new()), Vec::new()),
    ];
    let mut bundles = provider(Some(io::ErrorKind::PermissionDenied));
    for (contents, expected) in cases {
        let original = package(contents);
        let restored = DocumentPackage::decode(&original.encode().unwrap()).unwrap();
        let bytes = restored.resource_bytes("asset", &mut bundles).unwrap();
        assert!(matches!(bytes, Cow::Borrowed(_)));
        assert_eq!(bytes.as_ref(), expected);
    }
    assert!(bundles.requests.is_empty());
}

#[test]
fn embedded_resources_pass_exact_keys_and_return_owned_bytes() {
    let package = package(ResourceContents::Embedded);
    let original = package.clone();
    let mut bundles = provider(None);
    let bytes = package.resource_bytes("asset", &mut bundles).unwrap();
    assert!(matches!(bytes, Cow::Owned(_)));
    bundles.payload.clear();
    assert_eq!(bytes.as_ref(), [0, 255, 1]);
    assert_eq!(
        bundles.requests,
        [(
            "TrueTypeFont".into(),
            "/missing/import/or/bundle/key".into()
        )]
    );
    assert_eq!(package, original);
    assert!(
        package
            .resource_bytes("asset", &mut bundles)
            .unwrap()
            .is_empty()
    );
}

#[test]
fn missing_package_ids_do_not_trigger_bundle_or_origin_fallbacks() {
    let package = package(ResourceContents::Embedded);
    let mut bundles = provider(None);
    let error = package.resource_bytes("unknown", &mut bundles).unwrap_err();
    assert!(matches!(&error, ResourceReadError::Missing { id } if id == "unknown"));
    assert!(error.to_string().contains("unknown"));
    assert!(error.source().is_none());
    assert!(bundles.requests.is_empty());
}

#[test]
fn bundle_failure_retains_resource_context_and_io_cause_without_substitution() {
    let package = package(ResourceContents::Embedded);
    for kind in [
        io::ErrorKind::NotFound,
        io::ErrorKind::PermissionDenied,
        io::ErrorKind::InvalidData,
    ] {
        let mut bundles = provider(Some(kind));
        let error = package.resource_bytes("asset", &mut bundles).unwrap_err();
        let message = error.to_string();
        assert!(message.contains("asset"));
        assert!(message.contains("TrueTypeFont"));
        assert!(message.contains("/missing/import/or/bundle/key"));
        assert!(message.contains("bundle fixture failure"));
        assert_eq!(
            error
                .source()
                .unwrap()
                .downcast_ref::<io::Error>()
                .unwrap()
                .kind(),
            kind
        );
        assert_eq!(bundles.requests.len(), 1);
    }
}
