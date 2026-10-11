// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::{cargo_tree::tests::package, process::tests::FakeRunner, tree::tests::FixtureTree};

#[test]
fn command_plan_tests_all_targets_and_docs_with_locked_dependencies() {
    let runner = FakeRunner::default();
    for task in ["build", "test", "lint"] {
        workspace_command(&runner, Path::new("."), task).unwrap();
    }
    let calls = runner.calls.borrow();
    assert_eq!(calls.len(), 5);
    assert!(calls[0].contains(&"--all-targets".into()));
    assert!(calls[2].contains(&"--doc".into()));
    assert!(calls[4].contains(&"warnings".into()));
    assert!(workspace_command(&runner, Path::new("."), "skip").is_err());
}

#[test]
fn one_untested_crate_fails_even_when_another_has_full_coverage() {
    let tree = FixtureTree::new();
    let mut workspace = Metadata {
        workspace_root: tree.root().into(),
        workspace_members: vec!["fixture".into()],
        packages: vec![package(&tree)],
    };
    let report = |covered| {
        serde_json::json!({"data":[{"files":[{"filename":tree.root().join("src/lib.rs"), "summary":{"lines":{"count":100,"covered":covered}}}]}]}).to_string()
    };
    assert!(check_coverage(&report(81), &workspace).is_ok());
    assert!(check_coverage(&report(80), &workspace).is_err());
    let mut untested = package(&tree);
    untested.name = "untested".into();
    untested.manifest_path = tree.root().join("nested/Cargo.toml");
    workspace.packages.push(untested);
    assert!(
        check_coverage(&report(100), &workspace)
            .unwrap_err()
            .contains("untested")
    );
    assert!(check_coverage("invalid", &workspace).is_err());
}

#[test]
fn missing_required_display_and_wrong_tools_fail_by_name() {
    let tree = FixtureTree::new();
    tree.put(
        "Cargo.toml",
        "[package.metadata.oblikovati]\nrequires-display = true",
    );
    let workspace = Metadata {
        workspace_root: tree.root().into(),
        workspace_members: vec!["fixture".into()],
        packages: vec![package(&tree)],
    };
    assert!(
        check_display(&[workspace], true, false)
            .unwrap_err()
            .contains("DISPLAY")
    );
    let runner = FakeRunner::default();
    assert!(require_tool(&runner, tree.root(), "rustc", &["--version"], "1.99.0").is_err());
}

struct GateRunner {
    tree: FixtureTree,
    calls: std::cell::RefCell<Vec<String>>,
}

impl Runner for GateRunner {
    fn output(&self, root: &Path, program: &str, arguments: &[&str]) -> BuildResult<String> {
        self.calls
            .borrow_mut()
            .push(format!("{program} {}", arguments.join(" ")));
        match (program, arguments) {
            ("cargo", ["metadata", ..]) => Ok(serde_json::json!({"workspace_root": self.tree.root(), "workspace_members":["fixture"], "packages":[{"id":"fixture","name":"fixture","manifest_path":self.tree.root().join("Cargo.toml"),"license":"GPL-2.0-only","targets":[],"dependencies":[]}]}).to_string()),
            ("rustc", _) => Ok("rustc 1.99.0 (fixture)".into()),
            ("cargo", ["llvm-cov", "--version"]) => Ok("cargo-llvm-cov 0.9.1".into()),
            ("cargo", ["llvm-cov", ..]) => {
                let index = arguments.iter().position(|value| *value == "--output-path").unwrap();
                let report = serde_json::json!({"data":[{"files":[{"filename":root.join("src/lib.rs"),"summary":{"lines":{"count":100,"covered":90}}}]}]});
                write_text(Path::new(arguments[index + 1]), &report.to_string())?;
                Ok(String::new())
            }
            _ => Ok(String::new()),
        }
    }

    fn output_env(
        &self,
        root: &Path,
        program: &str,
        arguments: &[&str],
        _environment: &[(&str, &str)],
    ) -> BuildResult<String> {
        self.output(root, program, arguments)
    }
}

#[test]
fn full_gate_schedules_coverage_docs_workflow_lint_and_duplication_without_skips() {
    let runner = GateRunner {
        tree: FixtureTree::new(),
        calls: std::cell::RefCell::new(Vec::new()),
    };
    for name in ["Cargo.toml", "Cargo.lock", "LICENSE"] {
        runner.tree.put(name, "fixture");
    }
    runner.tree.put("Cargo.toml", "[package]\nname = 'fixture'");
    gate(runner.tree.root(), &runner).unwrap();
    run(runner.tree.root(), &runner, "test").unwrap();
    let calls = runner.calls.borrow();
    for fragment in [
        "llvm-cov --locked",
        "--doc",
        "markdownlint-cli2",
        "actionlint",
        "jscpd",
        "git diff --check",
    ] {
        assert!(
            calls.iter().any(|call| call.contains(fragment)),
            "{fragment}"
        );
    }
}
