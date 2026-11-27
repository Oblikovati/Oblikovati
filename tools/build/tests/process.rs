// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use std::cell::RefCell;

#[derive(Default)]
pub(crate) struct FakeRunner {
    pub(crate) calls: RefCell<Vec<Vec<String>>>,
    pub(crate) replies: RefCell<std::collections::VecDeque<BuildResult<String>>>,
}

impl Runner for FakeRunner {
    fn output(&self, _root: &Path, program: &str, arguments: &[&str]) -> BuildResult<String> {
        self.calls.borrow_mut().push(
            std::iter::once(program)
                .chain(arguments.iter().copied())
                .map(String::from)
                .collect(),
        );
        self.replies
            .borrow_mut()
            .pop_front()
            .unwrap_or_else(|| Ok(String::new()))
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
fn subprocess_errors_are_not_empty_successes() {
    let root = Path::new(".");
    assert!(
        SystemRunner
            .output(root, "missing-oblikovati-command", &[])
            .unwrap_err()
            .contains("required command")
    );
    assert!(
        SystemRunner
            .output(root, "git", &["not-a-git-command"])
            .unwrap_err()
            .contains("failed")
    );
    assert!(
        SystemRunner
            .output(root, "git", &["--version"])
            .unwrap()
            .starts_with("git version")
    );
}

#[test]
fn execute_preserves_arguments_and_failures() {
    let runner = FakeRunner::default();
    execute(&runner, Path::new("."), &["tool", "path with spaces"]).unwrap();
    runner.replies.borrow_mut().push_back(Err("failure".into()));
    assert!(execute(&runner, Path::new("."), &["tool"]).is_err());
    assert_eq!(runner.calls.borrow()[0], ["tool", "path with spaces"]);
}

#[test]
fn stream_env_preserves_child_environment_and_failure_status() {
    let root = Path::new(".");
    let environment = &[("GIT_CONFIG_COUNT", "invalid")];
    SystemRunner
        .stream_env(root, "git", &["--version"], environment)
        .unwrap();
    let error = SystemRunner
        .stream_env(root, "git", &["config", "--list"], environment)
        .unwrap_err();
    assert!(error.contains("config") && error.contains("failed ("));
}
