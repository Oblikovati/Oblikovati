// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    dependencies,
    fetched_source::locked_source_package,
    process::{BuildResult, Runner, execute},
};
use std::path::{Path, PathBuf};

/// #3795: use the reviewed SDK's installer; example: `prepare(root, runner, cache, github_env)?`.
pub(crate) fn prepare(
    root: &Path,
    runner: &dyn Runner,
    directory: &Path,
    environment: &Path,
) -> BuildResult<()> {
    let installer = locked_vulkan_installer(root, runner)?;
    execute(
        runner,
        root,
        &[
            "bash",
            &installer.to_string_lossy(),
            &directory.to_string_lossy(),
            "--github-env",
            &environment.to_string_lossy(),
        ],
    )
}

fn locked_vulkan_installer(root: &Path, runner: &dyn Runner) -> BuildResult<PathBuf> {
    dependencies::check_pins(root)?;
    let package = locked_source_package(root, runner, "martensite-vulkan")?;
    dependencies::require_locked_source(root, &package)?;
    installer_from_manifest(&package.manifest_path)
}

fn installer_from_manifest(manifest: &Path) -> BuildResult<PathBuf> {
    let root = manifest.parent().and_then(Path::parent).and_then(Path::parent)
        .ok_or_else(|| format!("Vulkan manifest {}: expected crates/martensite-vulkan/Cargo.toml inside the pinned SDK", manifest.display()))?;
    let installer = root.join("scripts/install/vulkan-validation.sh");
    if !installer.is_file() {
        return Err(format!(
            "Vulkan installer {}: expected the pinned SDK's checksum-verified installer",
            installer.display()
        ));
    }
    Ok(installer)
}

#[cfg(test)]
#[path = "../tests/native_validation.rs"]
mod tests;
