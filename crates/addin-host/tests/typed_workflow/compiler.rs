// SPDX-License-Identifier: GPL-2.0-only
use std::{
    path::{Path, PathBuf},
    process::Command,
};
#[path = "../support/resources.rs"]
mod resources;
use resources::{FixtureResources, run};
pub struct WorkflowCompiler {
    resources: FixtureResources,
}
impl WorkflowCompiler {
    /// Prepare an API-only fixture project; example: `WorkflowCompiler::new()`.
    pub fn new() -> Self {
        Self {
            resources: FixtureResources::new(),
        }
    }
    /// Build an API-only Rust document add-in; example: `compiler.rust_workflow(path)`.
    pub fn rust_workflow(&self, output: &Path) -> PathBuf {
        let project = self.resources.directory.path().join("typed-rust");
        prepare_workflow_project(&project, &self.resources.sdk, output);
        run(Command::new("cargo")
            .args(["build", "--offline", "--manifest-path"])
            .arg(project.join("Cargo.toml"))
            .arg("--target-dir")
            .arg(project.join("target")));
        project.join("target/debug/libtyped_workflow_fixture.so")
    }
}
fn prepare_workflow_project(project: &Path, sdk: &Path, output: &Path) {
    let source = project.join("src");
    std::fs::create_dir_all(&source).unwrap();
    let api = sdk.parent().unwrap().parent().unwrap();
    let manifest = format!(
        "[package]\nname = \"typed-workflow-fixture\"\nversion = \"0.0.0\"\nedition = \"2024\"\n[lib]\ncrate-type = [\"cdylib\"]\n[dependencies]\noblikovati-api = {{ path = {api:?} }}\nserde_json = \"=1.0.151\"\nserde = {{ version = \"1\", features = [\"derive\"] }}\n"
    );
    std::fs::write(project.join("Cargo.toml"), manifest).unwrap();
    let library = include_str!("../support/workflow_addin.rs.txt")
        .replace("OUTPUT_PATH", &format!("{output:?}"));
    std::fs::write(source.join("lib.rs"), library).unwrap();
    copy_workflow_sources(&source);
}
fn copy_workflow_sources(source: &Path) {
    let fixtures = Path::new(env!("CARGO_MANIFEST_DIR")).join("tests/typed_workflow");
    for name in ["client.rs", "transport.rs"] {
        std::fs::copy(fixtures.join(name), source.join(name)).unwrap();
    }
}
