// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    bootstrap::download,
    process::{BuildResult, Runner, execute},
    tree::write_text,
};
use std::path::Path;
const KACHE_VERSION: &str = "2.0.0";
#[derive(Debug)]
struct KacheAsset {
    name: String,
    digest: String,
    binaries: [&'static str; 2],
}
const KACHE_ASSETS: [(&str, &str, &str, &str); 6] = [
    (
        "linux",
        "x86_64",
        "x86_64-unknown-linux-musl",
        "6c59665faf1b14d0944391e6f2a1b510195e6be92c63060a9d1770ed00676824",
    ),
    (
        "linux",
        "aarch64",
        "aarch64-unknown-linux-musl",
        "8c5ca4d1d4b6fc66e5175c8a1150e1daa85099080646322058d9083c72986bba",
    ),
    (
        "macos",
        "x86_64",
        "x86_64-apple-darwin",
        "0ab8b29f659d159de78f70376c6ffa7730879f18fcde397567b683796eb5fd73",
    ),
    (
        "macos",
        "aarch64",
        "aarch64-apple-darwin",
        "b46139f7d42895c62962450886dd35f4554c17531209ea69b2db053d2a142bcb",
    ),
    (
        "windows",
        "x86_64",
        "x86_64-pc-windows-msvc",
        "e4d974025ffb64c8017ab3dfca9b37023e607fb5a720cb65b441b834e269546f",
    ),
    (
        "windows",
        "aarch64",
        "aarch64-pc-windows-msvc",
        "dd6b3135f7949cb3132697b6ae4affcc50b15ca04f566872b576377394bfaccd",
    ),
];
fn kache_asset(os: &str, arch: &str) -> BuildResult<KacheAsset> {
    let &(_, _, target, digest) = KACHE_ASSETS
        .iter()
        .find(|(host_os, host_arch, _, _)| *host_os == os && *host_arch == arch)
        .ok_or_else(|| format!("Kache host {os}/{arch}: expected a supported desktop target"))?;
    let (extension, binaries) = if os == "windows" {
        ("zip", ["kache.exe", "kache-engine.exe"])
    } else {
        ("tar.gz", ["kache", "kache-engine"])
    };
    Ok(KacheAsset {
        name: format!("kache-{target}.{extension}"),
        digest: digest.into(),
        binaries,
    })
}
/// Install the pinned client and engine locally; for example, `install(root, runner)?`.
pub(crate) fn install(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    let asset = kache_asset(std::env::consts::OS, std::env::consts::ARCH)?;
    install_archive(root, runner, &asset)
}
fn install_archive(root: &Path, runner: &dyn Runner, asset: &KacheAsset) -> BuildResult<()> {
    write_text(&root.join(".tools/.gitignore"), "*\n")?;
    let archive = root.join(format!(".tools/{}", asset.name));
    let url = format!(
        "https://github.com/kunobi-ninja/kache/releases/download/v{KACHE_VERSION}/{}",
        asset.name
    );
    download(root, runner, &url, &asset.digest, &archive)?;
    extract_pair(root, runner, asset, &archive)?;
    std::fs::remove_file(&archive).map_err(|cause| {
        format!("Kache archive {archive:?}: expected removal after extraction: {cause}")
    })
}
fn extract_pair(
    root: &Path,
    runner: &dyn Runner,
    asset: &KacheAsset,
    archive: &Path,
) -> BuildResult<()> {
    execute(
        runner,
        root,
        &[
            "tar",
            "-xf",
            &archive.to_string_lossy(),
            "-C",
            ".tools",
            asset.binaries[0],
            asset.binaries[1],
        ],
    )
}
#[cfg(test)]
#[path = "../tests/kache.rs"]
mod tests;
