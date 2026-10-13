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
    for call in &calls[1..=2] {
        assert!(call.ends_with(&["--".into(), "--test-threads=1".into()]));
    }
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
    help: crate::script_docs::tests::HelpSourceFixture,
    calls: std::cell::RefCell<Vec<String>>,
}

impl Runner for GateRunner {
    fn output(&self, root: &Path, program: &str, arguments: &[&str]) -> BuildResult<String> {
        self.calls
            .borrow_mut()
            .push(format!("{program} {}", arguments.join(" ")));
        match (program, arguments) {
            ("cargo", ["metadata", "--locked", "--format-version", "1"]) => Ok(self.help.metadata()),
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
    let runner = gate_fixture();
    gate(runner.tree.root(), &runner).unwrap();
    run(runner.tree.root(), &runner, "test").unwrap();
    let calls = runner.calls.borrow();
    let coverage = calls
        .iter()
        .find(|call| call.starts_with("cargo llvm-cov --locked"))
        .unwrap();
    assert!(coverage.ends_with(" -- --test-threads=1"));
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

#[test]
fn local_gate_refuses_stale_api_and_kernel_pins_before_external_checks() {
    let tree = FixtureTree::new();
    tree.put("Cargo.lock", include_str!("../../../Cargo.lock"));
    let runner = FakeRunner::default();
    for owner in ["api", "kernel"] {
        write_stale_pin(&tree, owner);
        let error = gate(tree.root(), &runner).unwrap_err();
        assert!(
            error.contains("locked") && error.contains(&"0".repeat(40)),
            "{error}"
        );
        assert!(runner.calls.borrow().is_empty());
    }
}

fn gate_fixture() -> GateRunner {
    let tree = FixtureTree::new();
    let help = crate::script_docs::tests::HelpSourceFixture::new(&tree);
    let runner = GateRunner {
        tree,
        help,
        calls: std::cell::RefCell::new(Vec::new()),
    };
    for name in ["Cargo.toml", "LICENSE"] {
        runner.tree.put(name, "fixture");
    }
    runner.tree.put("Cargo.toml", "[package]\nname = 'fixture'");
    runner
        .tree
        .put("release.toml", include_str!("../../../release.toml"));
    runner
        .tree
        .put("Cargo.lock", include_str!("../../../Cargo.lock"));
    runner
}

fn write_stale_pin(tree: &FixtureTree, owner: &str) {
    let mut config = toml::from_str::<crate::release_config::ReleaseConfig>(include_str!(
        "../../../release.toml"
    ))
    .unwrap();
    let pin = if owner == "api" {
        &mut config.api
    } else {
        &mut config.kernel
    };
    pin.revision = "0".repeat(40);
    tree.put("release.toml", &toml::to_string(&config).unwrap());
}
