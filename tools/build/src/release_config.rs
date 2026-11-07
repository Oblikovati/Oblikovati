// SPDX-License-Identifier: GPL-2.0-only

use crate::{cargo_tree::Metadata, process::BuildResult, tree::read_text, version::api_field};
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, path::Path};

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ReleaseConfig {
    pub(crate) product: Product,
    pub(crate) api: ReleasedDependency,
    pub(crate) kernel: ReleasedDependency,
    #[serde(
        default,
        rename = "kernel-oracle",
        skip_serializing_if = "Option::is_none"
    )]
    pub(crate) kernel_oracle: Option<ReleasedDependency>,
    pub(crate) linuxdeploy: Download,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "kebab-case")]
pub(crate) struct Product {
    pub(crate) major: u64,
    pub(crate) resources: String,
    pub(crate) publication_ready: bool,
    pub(crate) document_schema: String,
    pub(crate) reference_key: String,
    pub(crate) c_abi: String,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ReleasedDependency {
    pub(crate) repository: String,
    pub(crate) version: String,
    pub(crate) revision: String,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub(crate) packages: BTreeMap<String, String>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct Download {
    pub(crate) url: String,
    pub(crate) sha256: String,
}

pub(crate) fn load(root: &Path) -> BuildResult<ReleaseConfig> {
    toml::from_str(&read_text(&root.join("release.toml"))?).map_err(|error| {
        format!("release.toml: expected the documented release configuration: {error}")
    })
}

pub(crate) fn hex_digest(value: &str, length: usize) -> bool {
    value.len() == length
        && value
            .bytes()
            .all(|byte| byte.is_ascii_digit() || (b'a'..=b'f').contains(&byte))
}

pub(crate) fn validate_pin(dependency: &ReleasedDependency) -> BuildResult<()> {
    api_field(&dependency.version)?;
    if !hex_digest(&dependency.revision, 40) {
        return Err(format!(
            "{} revision {:?}: expected the released package's 40-digit commit",
            dependency.repository, dependency.revision
        ));
    }
    if ![
        "Oblikovati/Oblikovati.API",
        "Oblikovati/RustyCADKernel",
        "Oblikovati/Martensite",
    ]
    .contains(&dependency.repository.as_str())
    {
        return Err(format!(
            "dependency repository {:?}: expected the API or kernel owner",
            dependency.repository
        ));
    }
    validate_package_versions(dependency)
}

fn validate_package_versions(dependency: &ReleasedDependency) -> BuildResult<()> {
    if dependency.repository == "Oblikovati/Martensite" && dependency.packages.is_empty() {
        return Err("Martensite packages: expected individually versioned toolkit pins".into());
    }
    for (name, version) in &dependency.packages {
        if !name.starts_with("martensite-") {
            return Err(format!(
                "kernel package {name:?}: expected a martensite- toolkit name"
            ));
        }
        api_field(version)?;
    }
    Ok(())
}

pub(crate) fn package_version<'a>(
    dependency: &'a ReleasedDependency,
    name: &str,
) -> BuildResult<&'a str> {
    if dependency.packages.is_empty() {
        return Ok(&dependency.version);
    }
    match dependency.packages.get(name) {
        Some(version) => Ok(version),
        None => Err(format!(
            "{} package {name:?}: expected an explicit toolkit version",
            dependency.repository
        )),
    }
}

pub(crate) fn require_application(workspaces: &[Metadata]) -> BuildResult<()> {
    for name in ["oblikovati-cli", "oblikovati-head"] {
        let found = workspaces
            .iter()
            .flat_map(|workspace| &workspace.packages)
            .any(|package| {
                package.name == name
                    && package.targets.iter().any(|target| {
                        target.name == name && target.kind.iter().any(|kind| kind == "bin")
                    })
            });
        if !found {
            return Err(format!(
                "application target {name:?}: required Rust package and binary are not ported"
            ));
        }
    }
    Ok(())
}

pub(crate) fn publication_ready(config: &ReleaseConfig) -> BuildResult<()> {
    if !config.product.publication_ready {
        return Err("publication-ready = false: R9 application, package, install/update/rollback, and add-in acceptance is incomplete (#3560)".into());
    }
    for (name, value) in [
        ("document-schema", &config.product.document_schema),
        ("reference-key", &config.product.reference_key),
        ("c-abi", &config.product.c_abi),
    ] {
        if value.is_empty() || value == "pending" {
            return Err(format!(
                "{name} = {value:?}: expected a verified compatibility identifier"
            ));
        }
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/release_config.rs"]
pub(crate) mod tests;
