// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{process::tests::FakeRunner, release_fixture::ReleaseFixture};
fn fixture() -> ReleaseFixture {
    ReleaseFixture::new("x86_64-unknown-linux-gnu")
}
#[test]
fn source_workspace_catalog_admits_locked_component_versions_independently_of_release_version() {
    let fixture = fixture();
    let mut pin = fixture.config.kernel.clone();
    pin.version = "0.13.0".into();
    let mut fixture = fixture;
    fixture.config.kernel = pin.clone();
    let release: UpstreamRelease = serde_json::from_str(&fixture.sdk_release_metadata()).unwrap();
    verify(
        fixture.tree.root(),
        &fixture,
        &pin,
        &release,
        &["martensite-kernel".into()],
    )
    .unwrap();
}
#[test]
fn changed_catalog_bytes_missing_assets_and_unpublished_releases_are_refused() {
    let fixture = fixture();
    let pin = &fixture.config.kernel;
    let mut release: UpstreamRelease =
        serde_json::from_str(&fixture.sdk_release_metadata()).unwrap();
    let runner = FakeRunner::default();
    runner
        .replies
        .borrow_mut()
        .push_back(Ok(format!("{} ", fixture.sdk_catalog())));
    assert!(
        verify(
            fixture.tree.root(),
            &runner,
            pin,
            &release,
            &["martensite-kernel".into()]
        )
        .unwrap_err()
        .contains("digest")
    );
    release.assets.clear();
    assert!(
        verify(
            fixture.tree.root(),
            &fixture,
            pin,
            &release,
            &["martensite-kernel".into()]
        )
        .is_err()
    );
    release.draft = true;
    assert!(
        verify(
            fixture.tree.root(),
            &fixture,
            pin,
            &release,
            &["martensite-kernel".into()]
        )
        .is_err()
    );
}
#[test]
fn mismatched_source_revision_and_incomplete_verification_are_refused() {
    let fixture = fixture();
    let pin = &fixture.config.kernel;
    let release: UpstreamRelease = serde_json::from_str(&fixture.sdk_release_metadata()).unwrap();
    let mut catalog: SdkCatalog = serde_json::from_str(&fixture.sdk_catalog()).unwrap();
    catalog.source_workspace.revision = "f".repeat(40);
    assert!(
        check_catalog(&catalog, pin, &release)
            .unwrap_err()
            .contains("revision")
    );
    catalog.source_workspace.revision = pin.revision.clone();
    catalog.source_workspace.verification = "compiled-some-crates".into();
    assert!(check_catalog(&catalog, pin, &release).is_err());
    catalog.schema = 1;
    assert!(
        check_catalog(&catalog, pin, &release)
            .unwrap_err()
            .contains("schema")
    );
}
#[test]
fn every_locked_package_requires_one_matching_catalog_record() {
    let fixture = fixture();
    let lock = read_lockfile(&fixture.tree.root().join("Cargo.lock")).unwrap();
    let mut catalog: SdkCatalog = serde_json::from_str(&fixture.sdk_catalog()).unwrap();
    let names = ["martensite-kernel".into()];
    catalog.packages[0].version = "0.0.9".into();
    assert!(
        check_packages(&catalog, &lock, &names)
            .unwrap_err()
            .contains("locked version 0.1.0")
    );
    catalog.packages[0].version = "0.1.0".into();
    catalog.packages.push(CatalogPackage {
        name: "martensite-kernel".into(),
        version: "0.1.0".into(),
    });
    assert!(check_packages(&catalog, &lock, &names).is_err());
    catalog.packages.clear();
    assert!(check_packages(&catalog, &lock, &names).is_err());
}
