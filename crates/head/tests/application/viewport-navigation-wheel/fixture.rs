// SPDX-License-Identifier: GPL-2.0-only
//! Keep the wheel fixture available to the pinned and published benchmarks.

use super::ApplicationWorld;
use cucumber::given;
use std::path::{Path, PathBuf};

const WHEEL_FIXTURE: &str = "main-window-frame-start.opd";

/// Open a disposable copy of the documented fixture, as in `open_block(world)`.
#[given("`fixtures/cad/main-window-frame-start.opd` is open and active in the MCAD Software")]
pub(in super::super) fn open_block(world: &mut ApplicationWorld) {
    let pinned = std::env::var_os("OBK_BDD_ROOT").is_none()
        || std::env::var_os("OBK_BDD_REQUIRE_PINNED").is_some();
    let source = wheel_fixture_path(&super::super::scenario_root(), pinned);
    let copy = world.scratch_folder().join(WHEEL_FIXTURE);
    std::fs::copy(&source, &copy).unwrap_or_else(|error| {
        panic!("wheel fixture {source:?}: expected a readable saved Part: {error}")
    });
    world.open_fixture(&copy);
    let document = world.active_id();
    assert!(!world.state().document_storage[&document].unsaved_edits);
    world.parts.push(document);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

/// Use the named Part when the scenario opened it before establishing the view.
pub(in super::super) fn ensure_block(world: &mut ApplicationWorld) {
    if let Some(&document) = world.parts.last() {
        assert_eq!(
            world.active_id(),
            document,
            "expected the wheel fixture active"
        );
        return;
    }
    open_block(world);
}

fn wheel_fixture_path(source_root: &Path, pinned: bool) -> PathBuf {
    let specified = source_root.join("fixtures/cad").join(WHEEL_FIXTURE);
    // The original pinned feature describes the block but predates this asset.
    if pinned && !specified.is_file() {
        return Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../test-utilities/fixtures")
            .join(WHEEL_FIXTURE);
    }
    specified
}

#[test]
fn pinned_wheel_fixture_supports_older_snapshots_and_preserves_named_inputs() {
    let source = tempfile::tempdir().unwrap();
    let specified = source.path().join("fixtures/cad").join(WHEEL_FIXTURE);
    assert_eq!(wheel_fixture_path(source.path(), false), specified);
    let bundled = wheel_fixture_path(source.path(), true);
    assert!(bundled.is_file());
    std::fs::create_dir_all(specified.parent().unwrap()).unwrap();
    std::fs::copy(&bundled, &specified).unwrap();
    assert_eq!(wheel_fixture_path(source.path(), true), specified);
}
