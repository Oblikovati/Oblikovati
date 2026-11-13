// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn git(root: &Path, args: &[&str]) {
    assert!(
        Command::new("git")
            .arg("-C")
            .arg(root)
            .args(args)
            .status()
            .unwrap()
            .success()
    );
}
#[test]
fn unversioned_source_is_not_confused_with_an_ancestor_repository() {
    let temp = tempfile::tempdir().unwrap();
    git(temp.path(), &["init", "-q"]);
    let root = temp.path().join("scenarios/features/application");
    std::fs::create_dir_all(&root).unwrap();
    let revision = inspect(&root);
    assert_eq!((revision.commit, revision.dirty), (None, None));
}
#[test]
fn checkout_revision_and_dirty_state_are_recorded() {
    let temp = tempfile::tempdir().unwrap();
    let root = temp.path();
    git(root, &["init", "-q"]);
    git(
        root,
        &[
            "-c",
            "user.name=Test",
            "-c",
            "user.email=test@example.com",
            "commit",
            "-q",
            "--allow-empty",
            "-m",
            "snapshot",
        ],
    );
    let feature = root.join("features/application");
    std::fs::create_dir_all(&feature).unwrap();
    let clean = inspect(&feature);
    assert_eq!(clean.commit.as_ref().unwrap().len(), 40);
    assert_eq!(clean.dirty, Some(false));
    std::fs::write(feature.join("new.feature"), "Feature: changed\n").unwrap();
    let dirty = inspect(&feature);
    assert_eq!(dirty.commit, clean.commit);
    assert_eq!(dirty.dirty, Some(true));
    let report = root.join("revision.json");
    save(&report, &clean, &dirty, true);
    let record: serde_json::Value =
        serde_json::from_slice(&std::fs::read(report).unwrap()).unwrap();
    assert_eq!(record["before"]["dirty"], false);
    assert_eq!(record["after"]["dirty"], true);
    assert_eq!(record["source_changed"], true);
}

#[test]
fn changed_source_discards_result_files() {
    let temp = tempfile::tempdir().unwrap();
    let result = temp.path().join("result.json");
    std::fs::write(&result, "old result").unwrap();
    let revision = ScenarioRevision {
        root: temp.path().to_owned(),
        commit: None,
        dirty: None,
    };
    let failure =
        std::panic::catch_unwind(|| require_unchanged(&revision, &revision, true, &[&result]));
    assert!(failure.is_err());
    assert!(!result.exists());
}
