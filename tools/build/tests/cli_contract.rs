// SPDX-License-Identifier: GPL-2.0-only

use std::{
    path::Path,
    process::{Command, Output},
};

struct GitFixture(tempfile::TempDir);

impl GitFixture {
    fn new() -> Self {
        let fixture = Self(tempfile::tempdir().unwrap());
        std::fs::write(
            fixture.root().join("release.toml"),
            include_str!("fixtures/release.toml"),
        )
        .unwrap();
        fixture.git(&["init", "--quiet"]);
        fixture.git(&["commit", "--allow-empty", "-m", "feat: first Rust behavior"]);
        fixture
    }

    fn root(&self) -> &Path {
        self.0.path()
    }

    fn git(&self, args: &[&str]) {
        let output = Command::new("git")
            .args([
                "-c",
                "user.name=Fixture",
                "-c",
                "user.email=fixture@example.invalid",
            ])
            .args(args)
            .current_dir(self.root())
            .output()
            .unwrap();
        assert!(
            output.status.success(),
            "{}",
            String::from_utf8_lossy(&output.stderr)
        );
    }

    fn run(&self, args: &[&str]) -> Output {
        Command::new(env!("CARGO_BIN_EXE_oblikovati-build"))
            .args(args)
            .current_dir(self.root())
            .output()
            .unwrap()
    }
}

#[test]
fn command_versions_follow_real_git_history_and_ignore_go_releases() {
    let fixture = GitFixture::new();
    fixture.git(&["tag", "v9.999999.99.99"]);
    let first = fixture.run(&["version", "stable"]);
    assert!(first.status.success());
    assert_eq!(
        String::from_utf8(first.stdout).unwrap().trim(),
        "0.0015400.1.0"
    );
    fixture.git(&["tag", "rust-v0.0015400.1.0"]);
    fixture.git(&["commit", "--allow-empty", "-m", "fix: repair a workflow"]);
    let next = fixture.run(&["version", "stable"]);
    assert_eq!(
        String::from_utf8(next.stdout).unwrap().trim(),
        "0.0015400.1.1"
    );
    assert!(
        String::from_utf8(fixture.run(&["version", "nightly"]).stdout)
            .unwrap()
            .starts_with("0.0015400.1.1-nightly.")
    );
}

#[test]
fn invalid_commands_and_incomplete_publication_have_nonzero_exit_codes() {
    let fixture = GitFixture::new();
    for args in [
        vec!["unknown"],
        vec!["version", "beta"],
        vec!["preflight", "publish"],
    ] {
        let output = fixture.run(&args);
        assert!(!output.status.success());
        assert!(!output.stderr.is_empty());
    }
    let matrix = fixture.run(&["matrix"]);
    let matrix: serde_json::Value = serde_json::from_slice(&matrix.stdout).unwrap();
    assert_eq!(matrix["include"].as_array().unwrap().len(), 6);
}
