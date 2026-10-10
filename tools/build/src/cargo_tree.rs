// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    process::{BuildResult, Runner},
    tree::{read_text, source_files},
};
use serde::Deserialize;
use std::{
    collections::{BTreeMap, BTreeSet},
    path::{Path, PathBuf},
};

#[derive(Debug, Deserialize)]
pub(crate) struct Metadata {
    pub(crate) workspace_root: PathBuf,
    pub(crate) workspace_members: Vec<String>,
    pub(crate) packages: Vec<Package>,
}

#[derive(Debug, Deserialize)]
pub(crate) struct Package {
    pub(crate) id: String,
    pub(crate) name: String,
    pub(crate) manifest_path: PathBuf,
    pub(crate) license: Option<String>,
    pub(crate) targets: Vec<Target>,
    pub(crate) dependencies: Vec<Dependency>,
}

#[derive(Debug, Deserialize)]
pub(crate) struct Target {
    pub(crate) name: String,
    pub(crate) kind: Vec<String>,
}

#[derive(Debug, Deserialize)]
pub(crate) struct Dependency {
    pub(crate) name: String,
    pub(crate) source: Option<String>,
    pub(crate) path: Option<PathBuf>,
}

pub(crate) fn inventory(root: &Path, runner: &dyn Runner) -> BuildResult<Vec<Metadata>> {
    let manifests = source_files(root)?
        .into_iter()
        .filter(|path| path.file_name().is_some_and(|name| name == "Cargo.toml"));
    let mut workspaces = BTreeMap::new();
    for manifest in manifests {
        let output = runner.output(
            root,
            "cargo",
            &[
                "metadata",
                "--locked",
                "--no-deps",
                "--format-version",
                "1",
                "--manifest-path",
                &manifest.to_string_lossy(),
            ],
        )?;
        let metadata: Metadata = serde_json::from_str(&output)
            .map_err(|error| format!("Cargo metadata for {}: {error}", manifest.display()))?;
        workspaces.insert(metadata.workspace_root.clone(), metadata);
    }
    if workspaces.is_empty() {
        return Err(format!(
            "{}: expected at least one Cargo.toml",
            root.display()
        ));
    }
    for workspace in workspaces.values() {
        validate_workspace(root, workspace)?;
    }
    Ok(workspaces.into_values().collect())
}

fn validate_workspace(root: &Path, workspace: &Metadata) -> BuildResult<()> {
    if !workspace.workspace_root.starts_with(root) {
        return Err(format!(
            "workspace {}: expected a workspace inside {}",
            workspace.workspace_root.display(),
            root.display()
        ));
    }
    let lock = workspace.workspace_root.join("Cargo.lock");
    if !lock.is_file() {
        return Err(format!(
            "{}: required committed lockfile is missing",
            lock.display()
        ));
    }
    let members = workspace.workspace_members.iter().collect::<BTreeSet<_>>();
    for package in workspace
        .packages
        .iter()
        .filter(|package| members.contains(&package.id))
    {
        validate_package(root, package)?;
    }
    Ok(())
}

fn validate_package(root: &Path, package: &Package) -> BuildResult<()> {
    if package.license.as_deref() != Some("GPL-2.0-only") {
        return Err(format!(
            "package {} license {:?}: expected GPL-2.0-only",
            package.name, package.license
        ));
    }
    for dependency in &package.dependencies {
        if dependency
            .path
            .as_ref()
            .is_some_and(|path| !path.starts_with(root))
        {
            return Err(format!(
                "{} dependency {}: expected a released dependency, found sibling path {:?}",
                package.name, dependency.name, dependency.path
            ));
        }
        validate_dependency(package, dependency)?;
    }
    Ok(())
}

fn validate_dependency(package: &Package, dependency: &Dependency) -> BuildResult<()> {
    let kernel =
        dependency.name.starts_with("rustycad-") || dependency.name.starts_with("martensite-");
    if kernel && !["oblikovati-martensite", "oblikovati-viewport"].contains(&package.name.as_str())
    {
        return Err(format!(
            "{} imports {}: expected a Martensite adapter or viewport boundary",
            package.name, dependency.name
        ));
    }
    if (kernel || dependency.name == "oblikovati-api") && dependency.source.is_none() {
        return Err(format!(
            "{} dependency {}: expected a released Git or registry source",
            package.name, dependency.name
        ));
    }
    Ok(())
}

pub(crate) fn requires_display(package: &Package) -> BuildResult<bool> {
    let manifest: toml::Value = toml::from_str(&read_text(&package.manifest_path)?)
        .map_err(|error| format!("{}: {error}", package.manifest_path.display()))?;
    let declared = manifest
        .get("package")
        .and_then(|value| value.get("metadata"))
        .and_then(|value| value.get("oblikovati"))
        .and_then(|value| value.get("requires-display"))
        .and_then(toml::Value::as_bool)
        .unwrap_or(false);
    Ok(declared || package.name == "oblikovati-head")
}

#[cfg(test)]
#[path = "../tests/cargo_tree.rs"]
pub(crate) mod tests;
