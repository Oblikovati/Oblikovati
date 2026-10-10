// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    platform::artifact_names,
    process::{BuildResult, Runner},
    release_config::{ReleaseConfig, hex_digest},
    tree::{all_files as source_files, read_text, write_text},
};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::{
    collections::BTreeMap,
    fs,
    path::{Path, PathBuf},
};

#[derive(Debug, Deserialize, Serialize)]
pub(crate) struct ReleaseManifest {
    pub(crate) format: u32,
    pub(crate) version: String,
    pub(crate) application_commit: String,
    pub(crate) toolchain: String,
    pub(crate) dependencies: ReleaseConfig,
    pub(crate) artifacts: BTreeMap<String, String>,
}

pub(crate) fn sha256(path: &Path) -> BuildResult<String> {
    let bytes = fs::read(path).map_err(|error| format!("artifact {}: {error}", path.display()))?;
    if bytes.is_empty() {
        return Err(format!(
            "artifact {}: expected nonempty content",
            path.display()
        ));
    }
    Ok(format!("{:x}", Sha256::digest(bytes)))
}

pub(crate) fn new_manifest(
    root: &Path,
    runner: &dyn Runner,
    config: ReleaseConfig,
    version: &str,
) -> BuildResult<ReleaseManifest> {
    artifact_names(version)?;
    let commit = runner.output(root, "git", &["rev-parse", "HEAD"])?;
    let commit = commit.trim();
    if !hex_digest(commit, 40) {
        return Err(format!(
            "application commit {commit:?}: expected a 40-digit SHA"
        ));
    }
    let toolchain = runner.output(root, "rustc", &["--version"])?;
    Ok(ReleaseManifest {
        format: 1,
        version: version.into(),
        application_commit: commit.into(),
        toolchain: toolchain.trim().into(),
        dependencies: config,
        artifacts: BTreeMap::new(),
    })
}

pub(crate) fn collect(
    input: &Path,
    output: &Path,
    mut manifest: ReleaseManifest,
) -> BuildResult<()> {
    if output.exists() {
        return Err(format!(
            "output {}: expected a new distribution directory",
            output.display()
        ));
    }
    let expected = artifact_names(&manifest.version)?;
    let files = distribution_files(input, &expected)?;
    for name in &expected {
        let path = files.get(name).ok_or_else(|| {
            format!("release artifact {name}: required platform artifact is missing")
        })?;
        manifest.artifacts.insert(name.clone(), sha256(path)?);
    }
    fs::create_dir_all(output).map_err(|error| format!("{}: {error}", output.display()))?;
    for (name, path) in files {
        fs::copy(&path, output.join(&name)).map_err(|error| format!("copy {name}: {error}"))?;
    }
    write_manifest(output, &manifest)
}

fn distribution_files(input: &Path, expected: &[String]) -> BuildResult<BTreeMap<String, PathBuf>> {
    let mut files = BTreeMap::new();
    for path in source_files(input)? {
        let name = path
            .file_name()
            .unwrap_or_default()
            .to_string_lossy()
            .into_owned();
        if !expected.contains(&name) {
            return Err(format!(
                "downloaded artifact {name:?}: expected only final Rust release packages"
            ));
        }
        if files.insert(name.clone(), path).is_some() {
            return Err(format!(
                "artifact {name}: expected exactly one package, found duplicates"
            ));
        }
    }
    Ok(files)
}

fn write_manifest(output: &Path, manifest: &ReleaseManifest) -> BuildResult<()> {
    let text = serde_json::to_string_pretty(manifest)
        .map_err(|error| format!("release manifest: {error}"))?;
    write_text(&output.join("release-manifest.json"), &(text + "\n"))?;
    let mut digests = manifest.artifacts.clone();
    digests.insert(
        "release-manifest.json".into(),
        sha256(&output.join("release-manifest.json"))?,
    );
    let sums = digests
        .iter()
        .map(|(name, digest)| format!("{digest}  {name}\n"))
        .collect::<String>();
    write_text(&output.join("checksums.txt"), &sums)
}

pub(crate) fn verify(output: &Path, version: &str, commit: &str) -> BuildResult<ReleaseManifest> {
    let manifest: ReleaseManifest =
        serde_json::from_str(&read_text(&output.join("release-manifest.json"))?)
            .map_err(|error| format!("release-manifest.json: {error}"))?;
    if manifest.format != 1 || manifest.version != version || manifest.application_commit != commit
    {
        return Err(format!(
            "manifest version {:?} commit {:?}: expected format 1, {version} at {commit}",
            manifest.version, manifest.application_commit
        ));
    }
    let expected = artifact_names(version)?;
    if manifest.artifacts.keys().collect::<Vec<_>>() != expected.iter().collect::<Vec<_>>() {
        return Err(
            "manifest artifacts: expected the complete nine-package platform matrix".into(),
        );
    }
    for (name, digest) in &manifest.artifacts {
        if sha256(&output.join(name))? != *digest {
            return Err(format!(
                "artifact {name}: checksum differs from release manifest"
            ));
        }
    }
    verify_checksums(output, &manifest)?;
    Ok(manifest)
}

fn verify_checksums(output: &Path, manifest: &ReleaseManifest) -> BuildResult<()> {
    let mut digests = manifest.artifacts.clone();
    digests.insert(
        "release-manifest.json".into(),
        sha256(&output.join("release-manifest.json"))?,
    );
    let expected = digests
        .iter()
        .map(|(name, digest)| format!("{digest}  {name}\n"))
        .collect::<String>();
    if read_text(&output.join("checksums.txt"))? != expected {
        return Err("checksums.txt: expected the complete sorted SHA-256 inventory".into());
    }
    let actual = source_files(output)?;
    if actual.len() != manifest.artifacts.len() + 2 {
        return Err(
            "distribution directory: unexpected files outside the verified release inventory"
                .into(),
        );
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/distribution.rs"]
mod tests;
