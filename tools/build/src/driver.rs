// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    cargo_tree, dependencies, distribution, macos, packaging,
    process::{BuildResult, Runner, execute},
    release_config::{self, ReleaseConfig},
    tree::write_text,
};
use std::path::Path;

pub(crate) fn preflight(
    root: &Path,
    runner: &dyn Runner,
    publish: bool,
) -> BuildResult<ReleaseConfig> {
    let config = release_config::load(root)?;
    if publish {
        release_config::publication_ready(&config)?;
    }
    release_config::require_application(&cargo_tree::inventory(root, runner)?)?;
    for dependency in [&config.api, &config.kernel] {
        let names = dependencies::locked_crates(&root.join("Cargo.lock"), dependency)?;
        dependencies::verify_release(root, runner, dependency, &names)?;
    }
    packaging::resources(root, &config)?;
    Ok(config)
}

pub(crate) fn package(
    root: &Path,
    runner: &dyn Runner,
    target: &str,
    version: &str,
    signed: bool,
) -> BuildResult<()> {
    let config = preflight(root, runner, signed)?;
    let host = runner.output(root, "rustc", &["-vV"])?;
    if !host.lines().any(|line| line == format!("host: {target}")) {
        return Err(format!(
            "package target {target:?}: expected the native rustc host from {host:?}"
        ));
    }
    crate::native_package::package(root, runner, &config, target, version, signed)
}

pub(crate) fn mac_bundle(
    root: &Path,
    runner: &dyn Runner,
    version: &str,
    signed: bool,
) -> BuildResult<()> {
    let stage = root.join("target/macos-universal");
    macos::merge(
        root,
        runner,
        &root.join("stage/macos-amd64"),
        &root.join("stage/macos-arm64"),
        &stage,
    )?;
    let app = macos::assemble(root, runner, &stage, version)?;
    packaging::smoke(
        root,
        runner,
        crate::platform::platform("aarch64-apple-darwin")?,
        &app,
        version,
    )?;
    let output = root
        .join("dist")
        .join(format!("oblikovati-rust-{version}-macos-universal.zip"));
    if signed {
        return crate::signing::sign_app(
            root,
            runner,
            &app,
            &output,
            &crate::signing::credentials()?,
        );
    }
    execute(
        runner,
        root,
        &[
            "ditto",
            "-c",
            "-k",
            "--keepParent",
            &app.to_string_lossy(),
            &output.to_string_lossy(),
        ],
    )
}

pub(crate) fn collect(
    root: &Path,
    runner: &dyn Runner,
    version: &str,
    input: &Path,
    output: &Path,
) -> BuildResult<()> {
    let config = release_config::load(root)?;
    let manifest = distribution::new_manifest(root, runner, config, version)?;
    distribution::collect(input, output, manifest)
}

pub(crate) fn verify_release(
    root: &Path,
    runner: &dyn Runner,
    version: &str,
    output: &Path,
) -> BuildResult<()> {
    let config = preflight(root, runner, true)?;
    let commit = runner.output(root, "git", &["rev-parse", "HEAD"])?;
    let manifest = distribution::verify(output, version, commit.trim())?;
    if serde_json::to_value(&manifest.dependencies).map_err(|error| error.to_string())?
        != serde_json::to_value(config).map_err(|error| error.to_string())?
    {
        return Err("release dependencies: expected the exact reviewed release.toml pins and compatibility identifiers".into());
    }
    Ok(())
}

pub(crate) fn release_request(root: &Path, runner: &dyn Runner, channel: &str) -> BuildResult<()> {
    preflight(root, runner, true)?;
    let shallow = runner.output(root, "git", &["rev-parse", "--is-shallow-repository"])?;
    if shallow.trim() != "false" {
        return Err("release checkout: expected complete Git history and tags".into());
    }
    let ancestry = runner.output(
        root,
        "git",
        &["merge-base", "--is-ancestor", "HEAD", "origin/rust"],
    );
    ancestry.map_err(|_| "release commit: expected a commit on origin/rust".to_string())?;
    let config = release_config::load(root)?;
    let version = crate::version::compute(
        root,
        runner,
        config.product.major,
        &config.api.version,
        channel,
        chrono::Utc::now(),
    )?;
    let commit = runner.output(root, "git", &["rev-parse", "HEAD"])?;
    write_text(
        &root.join("target/release-request.json"),
        &serde_json::json!({"version": version, "commit": commit.trim()}).to_string(),
    )
}

#[cfg(test)]
#[path = "../tests/driver.rs"]
mod tests;
