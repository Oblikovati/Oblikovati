// SPDX-License-Identifier: GPL-2.0-only
use crate::tree::tests::FixtureTree;
use std::{os::unix::fs::PermissionsExt, path::PathBuf, process::Command};
const CACHE_ENVIRONMENT: [&str; 5] = [
    "RUSTC_WRAPPER",
    "KACHE_CONFIG",
    "KACHE_CACHE_DIR",
    "GIT_DIR",
    "GIT_WORK_TREE",
];
struct FakeCacheCargo {
    tree: FixtureTree,
    root: PathBuf,
}
impl FakeCacheCargo {
    fn new() -> Self {
        let tree = FixtureTree::new();
        tree.put(
            "source with spaces/Makefile",
            include_str!("../../../../Makefile"),
        );
        write_fake_cargo(&tree);
        let root = tree.root().join("source with spaces");
        Self { tree, root }
    }
    fn command(&self, mode: &str) -> Command {
        let path = format!(
            "PATH={}:/usr/bin:/bin",
            self.tree.root().join("bin").display()
        );
        let mode = format!("BUILD_CACHE={mode}");
        let mut command = Command::new("make");
        command.args(["--no-print-directory", "build", &path, &mode]);
        command.current_dir(&self.root);
        for name in CACHE_ENVIRONMENT {
            command.env_remove(name);
        }
        command
    }
    fn build_environment(&self, mode: &str) -> serde_json::Value {
        let output = self.command(mode).output().unwrap();
        assert!(
            output.status.success(),
            "make cache {mode}: expected successful fake Cargo: {}",
            String::from_utf8_lossy(&output.stderr)
        );
        let stdout = String::from_utf8(output.stdout).unwrap();
        let begin = stdout.find('{').unwrap();
        serde_json::from_str(&stdout[begin..]).unwrap()
    }
}
fn write_fake_cargo(tree: &FixtureTree) {
    tree.put("bin/cargo", "#!/bin/sh\nprintf '{\"wrapper\":\"%s\",\"config\":\"%s\",\"store\":\"%s\"}' \"$RUSTC_WRAPPER\" \"$KACHE_CONFIG\" \"$KACHE_CACHE_DIR\"\n");
    std::fs::set_permissions(
        tree.root().join("bin/cargo"),
        std::fs::Permissions::from_mode(0o755),
    )
    .unwrap();
}

#[test]
fn make_build_scopes_cache_settings_and_preserves_spaces_in_paths() {
    let fixture = FakeCacheCargo::new();
    let cache = fixture.build_environment("kache");
    assert_eq!(cache["wrapper"], "kache");
    assert_eq!(
        cache["config"],
        fixture.root.join(".kache.toml").to_str().unwrap()
    );
    assert_eq!(
        cache["store"],
        fixture.root.join(".cache/kache").to_str().unwrap()
    );
}
#[test]
fn cache_off_runs_cargo_without_a_compiler_wrapper() {
    let fixture = FakeCacheCargo::new();
    let cache = fixture.build_environment("off");
    assert_eq!(cache["wrapper"], "");
    assert_eq!(cache["config"], "");
    assert_eq!(cache["store"], "");
}

#[test]
fn make_preserves_the_ci_action_cache_environment() {
    let fixture = FakeCacheCargo::new();
    let output = fixture
        .command("kache")
        .env("RUSTC_WRAPPER", "ci-wrapper")
        .env("KACHE_CONFIG", "/ci/config.toml")
        .env("KACHE_CACHE_DIR", "/ci/cache")
        .output()
        .unwrap();
    assert!(output.status.success());
    let stdout = String::from_utf8(output.stdout).unwrap();
    let cache: serde_json::Value =
        serde_json::from_str(&stdout[stdout.find('{').unwrap()..]).unwrap();
    assert_eq!(cache["wrapper"], "ci-wrapper");
    assert_eq!(cache["config"], "/ci/config.toml");
    assert_eq!(cache["store"], "/ci/cache");
}

#[test]
fn make_refuses_an_unknown_cache_mode_before_building() {
    let fixture = FakeCacheCargo::new();
    let output = fixture.command("unknown").output().unwrap();
    assert!(!output.status.success());
    let stderr = String::from_utf8(output.stderr).unwrap();
    assert!(stderr.contains("BUILD_CACHE=unknown: expected kache or off"));
}
