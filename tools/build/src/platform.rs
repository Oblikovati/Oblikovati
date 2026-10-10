// SPDX-License-Identifier: GPL-2.0-only

use crate::process::BuildResult;
use serde::Serialize;

#[derive(Clone, Copy, Debug, Serialize)]
pub(crate) struct Platform {
    pub(crate) target: &'static str,
    pub(crate) runner: &'static str,
    pub(crate) suffix: &'static str,
    pub(crate) head: bool,
}

pub(crate) const PLATFORMS: [Platform; 6] = [
    Platform {
        target: "x86_64-unknown-linux-gnu",
        runner: "ubuntu-24.04",
        suffix: "linux-amd64",
        head: true,
    },
    Platform {
        target: "aarch64-unknown-linux-gnu",
        runner: "ubuntu-24.04-arm",
        suffix: "linux-arm64",
        head: false,
    },
    Platform {
        target: "x86_64-pc-windows-msvc",
        runner: "windows-2025",
        suffix: "windows-amd64",
        head: true,
    },
    Platform {
        target: "aarch64-pc-windows-msvc",
        runner: "windows-11-arm",
        suffix: "windows-arm64",
        head: false,
    },
    Platform {
        target: "x86_64-apple-darwin",
        runner: "macos-15-intel",
        suffix: "macos-amd64",
        head: true,
    },
    Platform {
        target: "aarch64-apple-darwin",
        runner: "macos-14",
        suffix: "macos-arm64",
        head: true,
    },
];

pub(crate) fn platform(target: &str) -> BuildResult<Platform> {
    PLATFORMS
        .iter()
        .find(|platform| platform.target == target)
        .copied()
        .ok_or_else(|| {
            format!("target {target:?}: expected a target in the six-platform release matrix")
        })
}

pub(crate) fn validate_version(version: &str) -> BuildResult<()> {
    let pattern = regex::Regex::new(r"^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+(?:-nightly\.[0-9]{8})?$")
        .expect("constant product version pattern");
    if !pattern.is_match(version) {
        return Err(format!(
            "version {version:?}: expected MAJOR.API.MINOR.PATCH[-nightly.YYMMDDHH]"
        ));
    }
    Ok(())
}

pub(crate) fn artifact_names(version: &str) -> BuildResult<Vec<String>> {
    validate_version(version)?;
    let mut names = PLATFORMS
        .iter()
        .map(|platform| {
            let extension = if platform.target.contains("windows") {
                "zip"
            } else {
                "tar.gz"
            };
            format!(
                "oblikovati-rust-cli-{version}-{}.{}",
                platform.suffix, extension
            )
        })
        .collect::<Vec<_>>();
    for suffix in [
        "linux-amd64.AppImage",
        "windows-amd64.zip",
        "macos-universal.zip",
    ] {
        names.push(format!("oblikovati-rust-{version}-{suffix}"));
    }
    names.sort();
    Ok(names)
}

#[cfg(test)]
#[path = "../tests/platform.rs"]
mod tests;
