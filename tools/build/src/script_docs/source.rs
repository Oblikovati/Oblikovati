// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use serde::Deserialize;
use std::path::PathBuf;
#[derive(Deserialize)]
struct HelpMetadata {
    packages: Vec<HelpPackage>,
}
#[derive(Deserialize)]
struct HelpPackage {
    name: String,
    version: String,
    source: Option<String>,
    manifest_path: PathBuf,
}
pub(super) fn api_source(
    root: &Path,
    runner: &dyn Runner,
    pin: &ReleasedDependency,
) -> BuildResult<PathBuf> {
    let metadata = metadata(root, runner)?;
    let package = metadata
        .packages
        .iter()
        .find(|package| package.name == "oblikovati-api")
        .ok_or("Cargo help metadata: expected the pinned oblikovati-api package")?;
    validate_package(package, pin)?;
    let parent = package
        .manifest_path
        .parent()
        .and_then(Path::parent)
        .ok_or_else(|| {
            format!(
                "API manifest {}: expected rust/Cargo.toml inside the source checkout",
                package.manifest_path.display()
            )
        })?;
    Ok(parent.join("go"))
}
fn validate_package(package: &HelpPackage, pin: &ReleasedDependency) -> BuildResult<()> {
    let source = package.source.as_deref().unwrap_or_default();
    let repository = source
        .split(['?', '#'])
        .next()
        .unwrap_or_default()
        .trim_end_matches(".git");
    if package.version != pin.version
        || repository != format!("git+https://github.com/{}", pin.repository)
        || !source.ends_with(&format!("#{}", pin.revision))
    {
        return Err(format!(
            "API help source {} {source:?}: expected released {} at {}",
            package.version, pin.version, pin.revision
        ));
    }
    Ok(())
}

fn metadata(root: &Path, runner: &dyn Runner) -> BuildResult<HelpMetadata> {
    let output = runner.output(
        root,
        "cargo",
        &["metadata", "--locked", "--format-version", "1"],
    )?;
    let metadata: HelpMetadata = serde_json::from_str(&output)
        .map_err(|error| format!("Cargo help metadata: expected package records: {error}"))?;
    Ok(metadata)
}
