// SPDX-License-Identifier: GPL-2.0-only
use serde::Deserialize;
use std::{path::PathBuf, process::Command};
#[derive(Deserialize)]
struct CargoPackages {
    packages: Vec<CargoPackage>,
}
#[derive(Deserialize)]
struct CargoPackage {
    name: String,
    manifest_path: PathBuf,
}
pub struct FixtureResources {
    pub directory: tempfile::TempDir,
    pub sdk: PathBuf,
}
impl FixtureResources {
    /// Prepare isolated artifacts using the pinned API source; example: `FixtureResources::new()`.
    pub fn new() -> Self {
        Self {
            directory: tempfile::tempdir().unwrap(),
            sdk: sdk_ffi_path(),
        }
    }
}
pub(crate) fn run(command: &mut Command) {
    let output = command
        .output()
        .expect("expected installed fixture compiler");
    assert!(
        output.status.success(),
        "fixture compile failed: {}",
        String::from_utf8_lossy(&output.stderr)
    );
}
fn sdk_ffi_path() -> PathBuf {
    let output = Command::new("cargo")
        .args(["metadata", "--locked", "--offline", "--format-version=1"])
        .arg("--filter-platform")
        .arg(compiler_host())
        .current_dir(env!("CARGO_MANIFEST_DIR"))
        .output()
        .unwrap();
    assert!(
        output.status.success(),
        "metadata failed: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    let metadata: CargoPackages = serde_json::from_slice(&output.stdout).unwrap();
    ffi_from_metadata(metadata)
}

fn compiler_host() -> String {
    let output = Command::new("rustc").arg("-vV").output().unwrap();
    assert!(output.status.success());
    String::from_utf8(output.stdout)
        .unwrap()
        .lines()
        .find_map(|line| line.strip_prefix("host: "))
        .unwrap()
        .to_owned()
}

fn ffi_from_metadata(metadata: CargoPackages) -> PathBuf {
    let package = metadata
        .packages
        .into_iter()
        .find(|entry| entry.name == "oblikovati-api")
        .unwrap();
    package.manifest_path.parent().unwrap().join("src/ffi.rs")
}
