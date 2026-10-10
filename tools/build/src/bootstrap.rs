// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    distribution::sha256,
    process::{BuildResult, Runner, execute},
    tree::write_text,
};
use std::path::Path;

fn actionlint_asset(os: &str, arch: &str) -> BuildResult<(&'static str, &'static str)> {
    match (os, arch) {
        ("linux", "x86_64") => Ok((
            "linux_amd64.tar.gz",
            "023070a287cd8cccd71515fedc843f1985bf96c436b7effaecce67290e7e0757",
        )),
        ("linux", "aarch64") => Ok((
            "linux_arm64.tar.gz",
            "401942f9c24ed71e4fe71b76c7d638f66d8633575c4016efd2977ce7c28317d0",
        )),
        ("macos", "x86_64") => Ok((
            "darwin_amd64.tar.gz",
            "28e5de5a05fc558474f638323d736d822fff183d2d492f0aecb2b73cc44584f5",
        )),
        ("macos", "aarch64") => Ok((
            "darwin_arm64.tar.gz",
            "2693315b9093aeacb4ebd91a993fea54fc215057bf0da2659056b4bc033873db",
        )),
        ("windows", "x86_64") => Ok((
            "windows_amd64.zip",
            "7f12f1801bca3d480d67aaf7774f4c2a6359a3ca8eebe382c95c10c9704aa731",
        )),
        ("windows", "aarch64") => Ok((
            "windows_arm64.zip",
            "76e9514cfac18e5677aa04f3a89873c981f16a2f2353bb97372a86cd09b1f5a8",
        )),
        _ => Err(format!(
            "actionlint host {os}/{arch}: expected a supported desktop target"
        )),
    }
}

pub(crate) fn install(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    let (suffix, digest) = actionlint_asset(std::env::consts::OS, std::env::consts::ARCH)?;
    let url = format!(
        "https://github.com/rhysd/actionlint/releases/download/v1.7.7/actionlint_1.7.7_{suffix}"
    );
    let archive = root.join(format!(".tools/actionlint_1.7.7_{suffix}"));
    write_text(&root.join(".tools/.gitignore"), "*\n")?;
    download(root, runner, &url, digest, &archive)?;
    execute(
        runner,
        root,
        &["tar", "-xf", &archive.to_string_lossy(), "-C", ".tools"],
    )
}

pub(crate) fn download(
    root: &Path,
    runner: &dyn Runner,
    url: &str,
    digest: &str,
    output: &Path,
) -> BuildResult<()> {
    if !url.starts_with("https://") || !crate::release_config::hex_digest(digest, 64) {
        return Err(format!(
            "download {url:?} sha256 {digest:?}: expected HTTPS and a pinned 64-digit SHA-256"
        ));
    }
    execute(
        runner,
        root,
        &[
            "curl",
            "--fail",
            "--location",
            "--proto",
            "=https",
            "--tlsv1.2",
            "--output",
            &output.to_string_lossy(),
            url,
        ],
    )?;
    let actual = sha256(output)?;
    if actual != digest {
        return Err(format!(
            "download {url}: SHA-256 {actual}, expected {digest}"
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/bootstrap.rs"]
mod tests;
