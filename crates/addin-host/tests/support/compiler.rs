// SPDX-License-Identifier: GPL-2.0-only
use serde::Deserialize;
use std::{
    path::{Path, PathBuf},
    process::Command,
};
#[derive(Deserialize)]
struct CargoPackages {
    packages: Vec<CargoPackage>,
}
#[derive(Deserialize)]
struct CargoPackage {
    name: String,
    manifest_path: PathBuf,
}
pub struct FixtureCompiler {
    directory: tempfile::TempDir,
    sdk: PathBuf,
}
impl FixtureCompiler {
    /// Prepare isolated real-library fixtures against the pinned contract; example: `FixtureCompiler::new()`.
    pub fn new() -> Self {
        Self {
            directory: tempfile::tempdir().unwrap(),
            sdk: sdk_ffi_path(),
        }
    }
    /// Build one trusted Rust ABI fixture; example: `compiler.rust_library("valid", &[], 0, 158)`.
    pub fn rust_library(&self, name: &str, flags: &[&str], major: i32, minor: i32) -> PathBuf {
        let source = self.directory.path().join(format!("{name}.rs"));
        let library = self.directory.path().join(format!("{name}.so"));
        let template =
            include_str!("fixture.rs.txt").replace("SDK_PATH", &format!("{:?}", self.sdk));
        let source_text = template
            .replace("API_MAJOR_VALUE", &major.to_string())
            .replace("API_MINOR_VALUE", &minor.to_string());
        std::fs::write(&source, source_text).unwrap();
        let mut compile = rust_compiler();
        for flag in flags {
            compile.arg("--cfg").arg(flag);
        }
        run(compile.arg(source).arg("-o").arg(&library));
        library
    }
    /// Build the fixed Go reference ABI fixture; example: `compiler.go_library(reference)`.
    pub fn go_library(&self, source: &Path) -> PathBuf {
        let library = self.directory.path().join("go-echo.so");
        run(Command::new("go")
            .env("GOWORK", "off")
            .env("CGO_ENABLED", "1")
            .args(["build", "-buildmode=c-shared", "-o"])
            .arg(&library)
            .arg(source));
        library
    }
}
fn run(command: &mut Command) {
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

fn rust_compiler() -> Command {
    let mut command = Command::new("rustc");
    command.args([
        "--crate-type",
        "cdylib",
        "--edition=2024",
        "-A",
        "dead_code",
    ]);
    command
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
