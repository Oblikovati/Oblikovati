// SPDX-License-Identifier: GPL-2.0-only
use std::{
    path::{Path, PathBuf},
    process::Command,
};
#[path = "resources.rs"]
mod resources;
use resources::{FixtureResources, run};
pub struct FixtureCompiler {
    resources: FixtureResources,
}
impl FixtureCompiler {
    /// Prepare isolated native fixtures; example: `FixtureCompiler::new()`.
    pub fn new() -> Self {
        Self {
            resources: FixtureResources::new(),
        }
    }
    /// Build one trusted Rust ABI fixture; example: `compiler.rust_library("valid", &[], 0, 158)`.
    pub fn rust_library(&self, name: &str, flags: &[&str], major: i32, minor: i32) -> PathBuf {
        let source = self.resources.directory.path().join(format!("{name}.rs"));
        let library = self.resources.directory.path().join(format!("{name}.so"));
        let template = include_str!("fixture.rs.txt")
            .replace("SDK_PATH", &format!("{:?}", self.resources.sdk));
        let source_text = fixture_source(template, flags, major, minor);
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
        let library = self.resources.directory.path().join("go-echo.so");
        run(Command::new("go")
            .env("GOWORK", "off")
            .env("CGO_ENABLED", "1")
            .args(["build", "-buildmode=c-shared", "-o"])
            .arg(&library)
            .arg(source));
        library
    }
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
fn fixture_source(template: String, flags: &[&str], major: i32, minor: i32) -> String {
    let template = if flags.contains(&"lifecycle") {
        let start = template
            .find("#[unsafe(no_mangle)]\npub extern \"C\" fn ObkAddInActivate")
            .unwrap();
        format!(
            "{}{}",
            &template[..start],
            include_str!("fixture_lifecycle.rs.txt")
        )
    } else {
        template
    };
    template
        .replace("API_MAJOR_VALUE", &major.to_string())
        .replace("API_MINOR_VALUE", &minor.to_string())
}
