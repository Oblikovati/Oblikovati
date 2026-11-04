// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    process::{BuildResult, Runner},
    release_config::{ReleasedDependency, validate_pin},
    tree::read_text,
};
use serde::Deserialize;
use std::path::Path;

#[derive(Deserialize)]
struct UpstreamRelease {
    draft: bool,
    prerelease: bool,
    assets: Vec<ReleaseAsset>,
}
#[derive(Deserialize)]
struct ReleaseAsset {
    name: String,
}
#[derive(Deserialize)]
struct CargoLock {
    package: Vec<LockedPackage>,
}
#[derive(Deserialize)]
struct LockedPackage {
    name: String,
    version: String,
    source: Option<String>,
}

pub(crate) fn check_pins(root: &Path) -> BuildResult<()> {
    let config = crate::release_config::load(root)?;
    for dependency in [&config.api, &config.kernel] {
        locked_crates(&root.join("Cargo.lock"), dependency)?;
    }
    check_renderer_pin(root)?;
    Ok(())
}

fn check_renderer_pin(root: &Path) -> BuildResult<()> {
    let lock: CargoLock = toml::from_str(&read_text(&root.join("Cargo.lock"))?)
        .map_err(|error| format!("renderer lock: expected Cargo packages: {error}"))?;
    for required in ["martensite-vulkan", "martensite-tkv3d"] {
        if !lock.package.iter().any(|package| package.name == required) {
            return Err(format!(
                "renderer lock: missing {required}; expected both renderer toolkits"
            ));
        }
    }
    for package in lock
        .package
        .iter()
        .filter(|item| item.name.starts_with("martensite-"))
    {
        verify_renderer_package(package)?;
    }
    Ok(())
}

fn verify_renderer_package(package: &LockedPackage) -> BuildResult<()> {
    const REVISION: &str = "cf938e4e801d6410649e3eff61e6b0d9c806fcb3";
    let expected =
        format!("git+https://github.com/Oblikovati/Martensite?rev={REVISION}#{REVISION}");
    if package.source.as_deref() != Some(expected.as_str()) {
        return Err(format!(
            "renderer {} source {:?}: expected {expected}",
            package.name, package.source
        ));
    }
    Ok(())
}

pub(crate) fn verify_release(
    root: &Path,
    runner: &dyn Runner,
    dependency: &ReleasedDependency,
    names: &[String],
) -> BuildResult<()> {
    validate_pin(dependency)?;
    let tag = format!("v{}", dependency.version);
    let endpoint = format!("repos/{}/releases/tags/{tag}", dependency.repository);
    let output = runner.output(root, "gh", &["api", &endpoint])?;
    let release: UpstreamRelease = serde_json::from_str(&output).map_err(|error| {
        format!("release {endpoint}: expected GitHub release metadata: {error}")
    })?;
    require_assets(&release, dependency, names)?;
    let repository = format!("https://github.com/{}.git", dependency.repository);
    let tag_ref = format!("refs/tags/{tag}");
    let refs = runner.output(
        root,
        "git",
        &[
            "ls-remote",
            &repository,
            &tag_ref,
            &format!("{tag_ref}^{{}}"),
        ],
    )?;
    let revision = release_commit(&refs, &tag_ref)?;
    if revision != dependency.revision {
        return Err(format!(
            "release {tag} commit {revision:?}: expected {}",
            dependency.revision
        ));
    }
    Ok(())
}

fn require_assets(
    release: &UpstreamRelease,
    dependency: &ReleasedDependency,
    names: &[String],
) -> BuildResult<()> {
    if release.draft || release.prerelease || names.is_empty() {
        return Err(format!(
            "{} v{}: expected a published stable release with consumed crates",
            dependency.repository, dependency.version
        ));
    }
    for name in names {
        let filename = format!("{name}-{}.crate", dependency.version);
        if !release.assets.iter().any(|asset| asset.name == filename) {
            return Err(format!(
                "{}: missing owner-tested release package {filename}",
                dependency.repository
            ));
        }
    }
    Ok(())
}

fn release_commit(refs: &str, tag: &str) -> BuildResult<String> {
    let lines = refs
        .lines()
        .filter_map(|line| line.split_once('\t'))
        .collect::<Vec<_>>();
    lines
        .iter()
        .find(|(_, name)| *name == format!("{tag}^{{}}"))
        .or_else(|| lines.iter().find(|(_, name)| *name == tag))
        .map(|(revision, _)| revision.to_string())
        .ok_or_else(|| format!("tag {tag:?}: expected a fetchable dependency release"))
}

pub(crate) fn locked_crates(
    lock_path: &Path,
    dependency: &ReleasedDependency,
) -> BuildResult<Vec<String>> {
    validate_pin(dependency)?;
    let lock: CargoLock = toml::from_str(&read_text(lock_path)?).map_err(|error| {
        format!(
            "{}: expected Cargo.lock packages: {error}",
            lock_path.display()
        )
    })?;
    let packages = lock
        .package
        .iter()
        .filter(|package| owned_package(&package.name, dependency))
        .collect::<Vec<_>>();
    for package in &packages {
        verify_locked(package, dependency)?;
    }
    if packages.is_empty() {
        return Err(format!(
            "{}: expected a consumed released crate from {}",
            lock_path.display(),
            dependency.repository
        ));
    }
    Ok(packages
        .iter()
        .map(|package| package.name.clone())
        .collect())
}

fn owned_package(name: &str, dependency: &ReleasedDependency) -> bool {
    if dependency.repository.ends_with("Oblikovati.API") {
        return name == "oblikovati-api";
    }
    name.starts_with("rustycad-")
}

fn verify_locked(package: &LockedPackage, dependency: &ReleasedDependency) -> BuildResult<()> {
    let repository = format!("git+https://github.com/{}", dependency.repository);
    let source = package.source.as_deref().unwrap_or_default();
    let base = source
        .split(['?', '#'])
        .next()
        .unwrap_or_default()
        .trim_end_matches(".git");
    if package.version != dependency.version
        || base != repository
        || !source.ends_with(&format!("#{}", dependency.revision))
    {
        return Err(format!(
            "locked {} {} source {source:?}: expected released {} at {}",
            package.name, package.version, dependency.version, dependency.revision
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/dependencies.rs"]
mod tests;
