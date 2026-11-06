// SPDX-License-Identifier: GPL-2.0-only
use crate::process::{BuildResult, Runner};
use serde::Deserialize;
use std::path::{Path, PathBuf};

#[derive(Debug, Deserialize)]
pub(crate) struct FetchedPackage {
    pub(crate) name: String,
    pub(crate) version: String,
    pub(crate) source: Option<String>,
    pub(crate) manifest_path: PathBuf,
}
#[derive(Deserialize)]
struct FetchedCatalog {
    packages: Vec<FetchedPackage>,
}

/// Resolve one package from the locked graph; example: `locked_source_package(root, runner, "oblikovati-api")?`.
pub(crate) fn locked_source_package(
    root: &Path,
    runner: &dyn Runner,
    name: &str,
) -> BuildResult<FetchedPackage> {
    let output = runner.output(
        root,
        "cargo",
        &["metadata", "--locked", "--format-version", "1"],
    )?;
    unique_source_package(&output, name)
}

fn unique_source_package(output: &str, name: &str) -> BuildResult<FetchedPackage> {
    let catalog: FetchedCatalog = serde_json::from_str(output)
        .map_err(|error| format!("Cargo source {name:?}: expected package records: {error}"))?;
    let mut found = catalog
        .packages
        .into_iter()
        .filter(|package| package.name == name);
    let selected = found.next().ok_or_else(|| {
        format!("Cargo source {name:?}: expected exactly one fetched package, found none")
    })?;
    if found.next().is_some() {
        return Err(format!(
            "Cargo source {name:?}: expected exactly one fetched package, found multiple versions"
        ));
    }
    Ok(selected)
}

#[cfg(test)]
#[path = "../tests/fetched_source.rs"]
mod tests;
