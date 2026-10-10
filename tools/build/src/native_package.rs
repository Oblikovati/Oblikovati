// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    bootstrap::download,
    distribution::sha256,
    packaging::{build_product, copy_tree, resources, smoke, stage_product},
    platform::Platform,
    process::{BuildResult, Runner, execute},
    release_config::ReleaseConfig,
    tree::write_text,
};
use std::{fs, path::Path};

pub(crate) fn package(
    root: &Path,
    runner: &dyn Runner,
    config: &ReleaseConfig,
    target: &str,
    version: &str,
    signed: bool,
) -> BuildResult<()> {
    resources(root, config)?;
    let platform = build_product(root, runner, config, target, version)?;
    let stage = stage_product(root, config, platform)?;
    native_libraries(root, runner, platform, &stage)?;
    smoke(root, runner, platform, &stage, version)?;
    if signed && target.contains("apple") {
        crate::signing::sign_stage(root, runner, &stage, &crate::signing::credentials()?)?;
    }
    let output = root.join("dist");
    fs::create_dir_all(&output).map_err(|error| error.to_string())?;
    cli_archive(root, runner, platform, &stage, version)?;
    if target.contains("apple") {
        return mac_stage(root, runner, platform, &stage);
    }
    if target.contains("linux") && platform.head {
        return linux_image(root, runner, config, &stage, version);
    }
    if platform.head {
        archive(
            root,
            runner,
            &stage,
            &output.join(format!("oblikovati-rust-{version}-{}.zip", platform.suffix)),
        )?;
    }
    Ok(())
}

pub(crate) fn archive(
    root: &Path,
    runner: &dyn Runner,
    stage: &Path,
    output: &Path,
) -> BuildResult<()> {
    let _ = (root, runner);
    crate::archive::create(stage, output)
}

fn cli_archive(
    root: &Path,
    runner: &dyn Runner,
    platform: Platform,
    stage: &Path,
    version: &str,
) -> BuildResult<()> {
    let cli_stage = root.join("target/cli-package").join(platform.target);
    if cli_stage.exists() {
        fs::remove_dir_all(&cli_stage).map_err(|error| error.to_string())?;
    }
    copy_tree(stage, &cli_stage)?;
    let head = cli_stage
        .join("bin")
        .join(if platform.target.contains("windows") {
            "oblikovati-head.exe"
        } else {
            "oblikovati-head"
        });
    if head.exists() {
        fs::remove_file(head).map_err(|error| error.to_string())?;
    }
    let extension = if platform.target.contains("windows") {
        "zip"
    } else {
        "tar.gz"
    };
    let output = root.join("dist").join(format!(
        "oblikovati-rust-cli-{version}-{}.{}",
        platform.suffix, extension
    ));
    archive(root, runner, &cli_stage, &output)
}

fn native_libraries(
    root: &Path,
    runner: &dyn Runner,
    platform: Platform,
    stage: &Path,
) -> BuildResult<()> {
    if platform.target.contains("apple") {
        return crate::macos::stage_libraries(root, runner, stage);
    }
    if platform.target.contains("windows") {
        return crate::windows::bundle(root, runner, platform, stage);
    }
    // Linux CLI archives must also run outside the checkout. Bundle linked libraries
    // through linuxdeploy, not a hard-coded list of current Rust dependencies.
    crate::linux::bundle(root, runner, stage)
}

fn mac_stage(
    root: &Path,
    runner: &dyn Runner,
    platform: Platform,
    stage: &Path,
) -> BuildResult<()> {
    let output = root
        .join("target")
        .join(format!("macos-stage-{}.tar.gz", platform.suffix));
    archive(root, runner, stage, &output)
}

fn linux_image(
    root: &Path,
    runner: &dyn Runner,
    config: &ReleaseConfig,
    stage: &Path,
    version: &str,
) -> BuildResult<()> {
    let deploy = root.join("target/linuxdeploy.AppImage");
    download(
        root,
        runner,
        &config.linuxdeploy.url,
        &config.linuxdeploy.sha256,
        &deploy,
    )?;
    execute(runner, root, &["chmod", "+x", &deploy.to_string_lossy()])?;
    let appdir = root.join("target/OblikovatiRust.AppDir");
    if appdir.exists() {
        fs::remove_dir_all(&appdir).map_err(|error| error.to_string())?;
    }
    copy_tree(stage, &appdir.join("usr"))?;
    let desktop = root.join("target/oblikovati-rust.desktop");
    write_text(
        &desktop,
        "[Desktop Entry]\nType=Application\nName=Oblikovati Rust\nExec=oblikovati-head\nIcon=oblikovati\nCategories=Graphics;Engineering;\nTerminal=false\n",
    )?;
    let output = root
        .join("dist")
        .join(format!("oblikovati-rust-{version}-linux-amd64.AppImage"));
    let args = linuxdeploy_arguments(&appdir, &desktop);
    print!(
        "{}",
        runner.output_env(
            root,
            &deploy.to_string_lossy(),
            &args.iter().map(String::as_str).collect::<Vec<_>>(),
            &[
                ("APPIMAGE_EXTRACT_AND_RUN", "1"),
                ("VERSION", version),
                ("OUTPUT", &output.to_string_lossy())
            ]
        )?
    );
    sha256(&output)?;
    Ok(())
}

fn linuxdeploy_arguments(appdir: &Path, desktop: &Path) -> Vec<String> {
    vec![
        "--appdir".into(),
        appdir.to_string_lossy().into_owned(),
        "--executable".into(),
        appdir
            .join("usr/bin/oblikovati-head")
            .to_string_lossy()
            .into_owned(),
        "--executable".into(),
        appdir
            .join("usr/bin/oblikovati-cli")
            .to_string_lossy()
            .into_owned(),
        "--desktop-file".into(),
        desktop.to_string_lossy().into_owned(),
        "--icon-file".into(),
        appdir
            .join("usr/share/oblikovati-rust/icons/oblikovati.png")
            .to_string_lossy()
            .into_owned(),
        "--output".into(),
        "appimage".into(),
    ]
}

#[cfg(test)]
#[path = "../tests/native_package.rs"]
mod tests;
