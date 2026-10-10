// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    distribution::{new_manifest, sha256},
    platform::{Platform, platform, validate_version},
    process::{BuildResult, Runner},
    release_config::ReleaseConfig,
    tree::{all_files as source_files, relative_name, write_text},
};
use std::{
    fs,
    path::{Component, Path, PathBuf},
};

pub(crate) fn resources(root: &Path, config: &ReleaseConfig) -> BuildResult<PathBuf> {
    let relative = Path::new(&config.product.resources);
    if relative
        .components()
        .any(|component| !matches!(component, Component::Normal(_)))
        || relative.as_os_str().is_empty()
    {
        return Err(format!(
            "resources {:?}: expected a relative path inside the repository",
            config.product.resources
        ));
    }
    let directory = root.join(relative);
    for required in [
        "icons/oblikovati.png",
        "icons/Oblikovati.icns",
        "fonts",
        "shaders",
        "templates",
        "catalogs",
        "NOTICES",
    ] {
        let path = directory.join(required);
        if !path.exists() || source_files(&path)?.is_empty() {
            return Err(format!(
                "resource {}: required nonempty application resource is missing",
                path.display()
            ));
        }
    }
    Ok(directory)
}

pub(crate) fn copy_tree(source: &Path, destination: &Path) -> BuildResult<()> {
    for file in source_files(source)? {
        let relative = relative_name(source, &file)?;
        copy_file(&file, &destination.join(relative))?;
    }
    Ok(())
}

pub(crate) fn copy_file(source: &Path, destination: &Path) -> BuildResult<()> {
    sha256(source)?;
    let parent = destination
        .parent()
        .ok_or_else(|| format!("{}: expected a parent directory", destination.display()))?;
    fs::create_dir_all(parent).map_err(|error| format!("{}: {error}", parent.display()))?;
    fs::copy(source, destination).map_err(|error| {
        format!(
            "copy {} to {}: {error}",
            source.display(),
            destination.display()
        )
    })?;
    Ok(())
}

pub(crate) fn build_product(
    root: &Path,
    runner: &dyn Runner,
    config: &ReleaseConfig,
    target: &str,
    version: &str,
) -> BuildResult<Platform> {
    validate_version(version)?;
    let platform = platform(target)?;
    let manifest = new_manifest(root, runner, config.clone(), version)?;
    let manifest_path = root.join("target/release-build.json");
    write_text(
        &manifest_path,
        &serde_json::to_string_pretty(&manifest).map_err(|error| error.to_string())?,
    )?;
    let mut args = vec![
        "build",
        "--locked",
        "--release",
        "--target",
        target,
        "-p",
        "oblikovati-cli",
        "--bin",
        "oblikovati-cli",
    ];
    if platform.head {
        args.extend(["-p", "oblikovati-head", "--bin", "oblikovati-head"]);
    }
    let epoch = runner.output(root, "git", &["show", "-s", "--format=%ct", "HEAD"])?;
    print!(
        "{}",
        runner.output_env(
            root,
            "cargo",
            &args,
            &[
                (
                    "OBLIKOVATI_BUILD_MANIFEST",
                    &manifest_path.to_string_lossy()
                ),
                ("SOURCE_DATE_EPOCH", epoch.trim())
            ]
        )?
    );
    Ok(platform)
}

pub(crate) fn stage_product(
    root: &Path,
    config: &ReleaseConfig,
    platform: Platform,
) -> BuildResult<PathBuf> {
    let stage = root.join("target/package").join(platform.target);
    if stage.exists() {
        fs::remove_dir_all(&stage)
            .map_err(|error| format!("clear generated stage {}: {error}", stage.display()))?;
    }
    let extension = if platform.target.contains("windows") {
        ".exe"
    } else {
        ""
    };
    for name in ["oblikovati-cli", "oblikovati-head"]
        .iter()
        .take(if platform.head { 2 } else { 1 })
    {
        let binary = format!("{name}{extension}");
        copy_file(
            &root
                .join("target")
                .join(platform.target)
                .join("release")
                .join(&binary),
            &stage.join("bin").join(binary),
        )?;
    }
    copy_file(&root.join("LICENSE"), &stage.join("LICENSE"))?;
    copy_file(
        &root.join("target/release-build.json"),
        &stage.join("release-manifest.json"),
    )?;
    copy_tree(
        &resources(root, config)?,
        &stage.join("share/oblikovati-rust"),
    )?;
    Ok(stage)
}

pub(crate) fn smoke(
    root: &Path,
    runner: &dyn Runner,
    platform: Platform,
    installed: &Path,
    version: &str,
) -> BuildResult<()> {
    let directory = if installed
        .extension()
        .is_some_and(|extension| extension == "app")
    {
        "Contents/MacOS"
    } else {
        "bin"
    };
    let executable = installed
        .join(directory)
        .join(if platform.target.contains("windows") {
            "oblikovati-cli.exe"
        } else {
            "oblikovati-cli"
        });
    let output = runner.output(installed, &executable.to_string_lossy(), &["--version"])?;
    if !output.contains(version) {
        return Err(format!("installed version {output:?}: expected {version}"));
    }
    // The application owns this test: it must open the installed GUI and exercise C ABI add-ins.
    let script = root
        .join("packaging/smoke")
        .join(format!("{}.sh", platform.target));
    if !script.is_file() {
        return Err(format!(
            "{}: required native package smoke test has not been ported",
            script.display()
        ));
    }
    print!(
        "{}",
        runner.output(
            installed,
            "bash",
            &[&script.to_string_lossy(), &installed.to_string_lossy()]
        )?
    );
    Ok(())
}

#[cfg(test)]
#[path = "../tests/packaging.rs"]
pub(crate) mod tests;
