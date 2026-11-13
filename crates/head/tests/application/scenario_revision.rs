// SPDX-License-Identifier: GPL-2.0-only
//! Describe the actual scenario checkout, including an unversioned source.
use serde::Serialize;
use std::{
    path::{Path, PathBuf},
    process::Command,
};

#[derive(Debug, PartialEq, Eq, Serialize)]
pub(super) struct ScenarioRevision {
    root: PathBuf,
    commit: Option<String>,
    dirty: Option<bool>,
}

pub(super) fn inspect(feature_root: &Path) -> ScenarioRevision {
    let root = feature_root
        .ancestors()
        .find(|path| path.file_name().is_some_and(|name| name == "features"))
        .and_then(Path::parent)
        .expect("scenario path: expected a features directory")
        .to_owned();
    if !root.join(".git").exists() {
        return ScenarioRevision {
            root,
            commit: None,
            dirty: None,
        };
    }
    let commit = git_output(&root, &["rev-parse", "HEAD"]);
    let dirty = !git_output(&root, &["status", "--porcelain", "--untracked-files=all"]).is_empty();
    ScenarioRevision {
        root,
        commit: Some(commit),
        dirty: Some(dirty),
    }
}
fn git_output(root: &Path, args: &[&str]) -> String {
    let result = Command::new("git")
        .arg("-C")
        .arg(root)
        .args(args)
        .output()
        .unwrap_or_else(|error| panic!("scenario checkout {root:?}: expected Git to run: {error}"));
    assert!(
        result.status.success(),
        "scenario checkout {root:?}: git {args:?} failed: {}",
        String::from_utf8_lossy(&result.stderr)
    );
    String::from_utf8(result.stdout)
        .expect("Git output: expected UTF-8")
        .trim()
        .to_owned()
}
pub(super) fn save(
    path: &Path,
    before: &ScenarioRevision,
    after: &ScenarioRevision,
    source_changed: bool,
) {
    let record =
        serde_json::json!({"before": before, "after": after, "source_changed": source_changed});
    std::fs::create_dir_all(path.parent().unwrap()).unwrap();
    std::fs::write(path, serde_json::to_vec_pretty(&record).unwrap()).unwrap();
}
pub(super) fn require_unchanged(
    before: &ScenarioRevision,
    after: &ScenarioRevision,
    source_changed: bool,
    results: &[&Path],
) {
    if !source_changed && before == after {
        return;
    }
    for result in results {
        if result.exists() {
            std::fs::remove_file(result).unwrap_or_else(|error| {
                panic!("invalid scenario result {result:?}: expected removal: {error}")
            });
        }
    }
    panic!("scenario checkout changed during run; results discarded");
}
#[cfg(test)]
#[path = "scenario_revision_tests.rs"]
mod tests;
