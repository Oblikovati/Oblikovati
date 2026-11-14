// SPDX-License-Identifier: GPL-2.0-only
//! Monitor benchmark inputs without treating discovery as scenario acceptance.
mod ledger;
mod source;
use crate::process::BuildResult;
use serde::Serialize;
use sha2::{Digest, Sha256};
use source::{NativeScenarioSource, ScenarioSource};
use std::path::Path;
#[derive(Debug, Serialize, PartialEq, Eq)]
struct ScenarioFeatureSource {
    path: String,
    area: String,
    feature: String,
    sha256: String,
}
#[derive(Debug, Serialize, PartialEq, Eq)]
struct ScenarioInventory {
    features: Vec<ScenarioFeatureSource>,
}
pub(crate) fn print_ledger(
    root: &Path,
    revision: &str,
    runner: &dyn crate::process::Runner,
) -> BuildResult<()> {
    ledger::verify_checkout(root, revision, runner)?;
    let ledger = ledger::inventory(root, revision, &mut NativeScenarioSource)?;
    let second = ledger::inventory(root, revision, &mut NativeScenarioSource)?;
    ledger::verify_checkout(root, revision, runner)?;
    if ledger != second {
        return Err(format!(
            "scenario root {root:?}: expected unchanged files during inventory"
        ));
    }
    println!(
        "{}",
        serde_json::to_string(&ledger)
            .map_err(|error| format!("scenario ledger: expected JSON: {error}"))?
    );
    Ok(())
}
pub(crate) fn print_inventory(root: &Path) -> BuildResult<()> {
    let inventory = scan(root, &mut NativeScenarioSource)?;
    println!(
        "{}",
        serde_json::to_string_pretty(&inventory)
            .map_err(|cause| format!("scenario inventory: expected JSON: {cause}"))?
    );
    Ok(())
}
fn scan(root: &Path, source: &mut impl ScenarioSource) -> BuildResult<ScenarioInventory> {
    let mut paths = source.feature_paths(root)?;
    paths.sort();
    let features = paths
        .into_iter()
        .map(|path| describe(root, &path, source))
        .collect::<BuildResult<Vec<_>>>()?;
    if features.is_empty() {
        return Err(format!(
            "scenario root {root:?}: expected features/<area>/<feature>/*.feature files"
        ));
    }
    Ok(ScenarioInventory { features })
}
fn describe(
    root: &Path,
    path: &Path,
    source: &mut impl ScenarioSource,
) -> BuildResult<ScenarioFeatureSource> {
    let relative = path.strip_prefix(root).map_err(|cause| {
        format!("scenario path {path:?}: expected a child of {root:?}: {cause}")
    })?;
    let components: Vec<&str> = relative
        .iter()
        .map(|part| {
            part.to_str()
                .ok_or_else(|| format!("scenario path {path:?}: expected UTF-8 components"))
        })
        .collect::<BuildResult<_>>()?;
    if components.len() != 4
        || components[0] != "features"
        || path.extension().and_then(|extension| extension.to_str()) != Some("feature")
    {
        return Err(format!(
            "scenario path {relative:?}: expected features/<area>/<feature>/<name>.feature"
        ));
    }
    let bytes = source.read(path)?;
    Ok(ScenarioFeatureSource {
        path: components.join("/"),
        area: components[1].into(),
        feature: components[2].into(),
        sha256: format!("{:x}", Sha256::digest(bytes)),
    })
}
#[cfg(test)]
#[path = "../../tests/scenarios/inventory.rs"]
mod tests;
