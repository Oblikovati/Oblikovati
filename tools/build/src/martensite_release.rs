// SPDX-License-Identifier: GPL-2.0-only
//! Check the owner's complete SDK release, whose component versions are independent.
use super::*;
use crate::release_config::hex_digest;
use sha2::{Digest, Sha256};
#[derive(Deserialize)]
struct SdkCatalog {
    schema: u32,
    release: String,
    source_consumption: String,
    source_workspace: SourceWorkspace,
    packages: Vec<CatalogPackage>,
}
#[derive(Deserialize)]
struct SourceWorkspace {
    archive: String,
    revision: String,
    sha256: String,
    verification: String,
}
#[derive(Deserialize)]
struct CatalogPackage {
    name: String,
    version: String,
}
pub(super) fn verify(
    root: &Path,
    runner: &dyn Runner,
    dependency: &ReleasedDependency,
    release: &UpstreamRelease,
    names: &[String],
) -> BuildResult<()> {
    if release.draft || release.prerelease || names.is_empty() {
        return Err(format!(
            "Martensite release {}: expected stable metadata and consumed packages",
            dependency.version
        ));
    }
    let catalog = download_catalog(root, runner, dependency, release)?;
    check_catalog(&catalog, dependency, release)?;
    check_packages(&catalog, &read_lockfile(&root.join("Cargo.lock"))?, names)
}
fn download_catalog(
    root: &Path,
    runner: &dyn Runner,
    dependency: &ReleasedDependency,
    release: &UpstreamRelease,
) -> BuildResult<SdkCatalog> {
    let filename = format!("martensite-{}-packages.json", dependency.version);
    let asset = asset_named(release, &filename)?;
    let body = fetch_catalog(root, runner, dependency, &filename)?;
    check_digest(asset, &format!("{:x}", Sha256::digest(body.as_bytes())))?;
    serde_json::from_str(&body).map_err(|cause| {
        format!("SDK catalog {filename:?}: expected schema-2 package records: {cause}")
    })
}

fn fetch_catalog(
    root: &Path,
    runner: &dyn Runner,
    dependency: &ReleasedDependency,
    filename: &str,
) -> BuildResult<String> {
    let tag = format!("v{}", dependency.version);
    let args = [
        "release",
        "download",
        &tag,
        "--repo",
        &dependency.repository,
        "--pattern",
        filename,
        "--output",
        "-",
    ];
    runner.output(root, "gh", &args)
}

fn asset_named<'a>(release: &'a UpstreamRelease, name: &str) -> BuildResult<&'a ReleaseAsset> {
    let mut found = release.assets.iter().filter(|asset| asset.name == name);
    let asset = found
        .next()
        .ok_or_else(|| format!("SDK asset {name:?}: expected a published release asset"))?;
    if found.next().is_some() {
        return Err(format!(
            "SDK asset {name:?}: expected one unique release asset"
        ));
    }
    Ok(asset)
}
fn check_digest(asset: &ReleaseAsset, sha256: &str) -> BuildResult<()> {
    let expected = format!("sha256:{sha256}");
    if !hex_digest(sha256, 64) || asset.digest.as_deref() != Some(&expected) {
        return Err(format!(
            "SDK asset {:?} digest {:?}: expected {expected}",
            asset.name, asset.digest
        ));
    }
    Ok(())
}
fn check_catalog(
    catalog: &SdkCatalog,
    dependency: &ReleasedDependency,
    release: &UpstreamRelease,
) -> BuildResult<()> {
    if catalog.schema != 2
        || catalog.release != dependency.version
        || catalog.source_consumption != "source_workspace"
    {
        return Err(format!(
            "SDK catalog schema {}, release {:?}, consumption {:?}: expected schema 2, release {:?} and source_workspace",
            catalog.schema, catalog.release, catalog.source_consumption, dependency.version
        ));
    }
    check_workspace(&catalog.source_workspace, dependency, release)
}
fn check_workspace(
    workspace: &SourceWorkspace,
    dependency: &ReleasedDependency,
    release: &UpstreamRelease,
) -> BuildResult<()> {
    let name = format!("martensite-{}-source-workspace.tar.gz", dependency.version);
    if workspace.archive != name
        || workspace.revision != dependency.revision
        || workspace.verification
            != "offline-production-libraries-all-features-and-external-consumer"
    {
        return Err(format!(
            "SDK workspace {:?} revision {:?} verification {:?}: expected {name:?} at {} verified offline with external consumers",
            workspace.archive, workspace.revision, workspace.verification, dependency.revision
        ));
    }
    check_digest(asset_named(release, &name)?, &workspace.sha256)
}
fn check_packages(catalog: &SdkCatalog, lock: &CargoLock, names: &[String]) -> BuildResult<()> {
    for name in names {
        let locked = unique_locked_package(lock, name)?;
        let record = unique_catalog_package(catalog, name)?;
        if record.version != locked.version {
            return Err(format!(
                "SDK package {name:?} version {:?}: expected one catalog record for locked version {}",
                record.version, locked.version
            ));
        }
    }
    Ok(())
}
fn unique_catalog_package<'a>(
    catalog: &'a SdkCatalog,
    name: &str,
) -> BuildResult<&'a CatalogPackage> {
    let mut found = catalog
        .packages
        .iter()
        .filter(|package| package.name == name);
    let record = found.next().ok_or_else(|| {
        format!("SDK package {name:?}: expected one catalog record for the consumed package")
    })?;
    if found.next().is_some() {
        return Err(format!(
            "SDK package {name:?}: expected one unique catalog record"
        ));
    }
    Ok(record)
}

#[cfg(test)]
#[path = "../tests/martensite_release.rs"]
mod tests;
