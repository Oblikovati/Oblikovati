// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::fetched_source::{FetchedPackage, locked_source_package};
use std::path::PathBuf;
pub(super) fn api_source(
    root: &Path,
    runner: &dyn Runner,
    pin: &ReleasedDependency,
) -> BuildResult<PathBuf> {
    let package = locked_source_package(root, runner, "oblikovati-api")?;
    validate_package(&package, pin)?;
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
fn validate_package(package: &FetchedPackage, pin: &ReleasedDependency) -> BuildResult<()> {
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
