// SPDX-License-Identifier: GPL-2.0-only
//! Prepare the documented native part for window-frame scenarios.
use super::ApplicationWorld;
use std::path::{Path, PathBuf};

const FRAME_FIXTURE: &str = "main-window-frame-start.opd";

/// Open a disposable copy of the visible-solid part; example: `open_frame_part(world)`.
pub(in super::super) fn open_frame_part(world: &mut ApplicationWorld) -> u64 {
    open_frame_part_from(world, &scenario_frame_fixture())
}

fn scenario_frame_fixture() -> PathBuf {
    super::super::scenario_root()
        .join("fixtures/cad")
        .join(FRAME_FIXTURE)
}

fn open_frame_part_from(world: &mut ApplicationWorld, source: &Path) -> u64 {
    let copy = world.scratch_folder().join("main-window-frame-source.opd");
    std::fs::copy(source, &copy).unwrap_or_else(|error| {
        panic!("native frame fixture {source:?}: expected a readable part: {error}")
    });
    world.open_fixture(&copy);
    let document = world.active_id();
    world.parts.push(document);
    assert!(!world.state().document_storage[&document].unsaved_edits);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
    document
}

/// Make an accepted sketch edit on the copy; example: `open_dirty_frame_part(world)`.
pub(in super::super) fn open_dirty_frame_part(world: &mut ApplicationWorld) -> u64 {
    open_dirty_frame_part_from(world, &scenario_frame_fixture())
}

/// Exercise an edit on a disposable copy of the supplied saved Part.
pub(in super::super) fn open_dirty_frame_part_from(
    world: &mut ApplicationWorld,
    source: &Path,
) -> u64 {
    let document = open_frame_part_from(world, source);
    world.start_sketch();
    world.finish_sketch();
    assert!(world.state().document_storage[&document].unsaved_edits);
    document
}

/// Use the repository's saved Part for standalone tests.
#[cfg(target_os = "linux")]
pub(in super::super) fn bundled_frame_fixture() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures")
        .join(FRAME_FIXTURE)
}

#[cfg(target_os = "linux")]
#[test]
fn native_frame_fixture_opens_with_a_solid_and_accepts_a_dirty_edit() {
    let source = bundled_frame_fixture();
    let before = std::fs::read(&source).unwrap();
    let mut world = ApplicationWorld::new();
    let document = open_dirty_frame_part_from(&mut world, &source);
    assert_eq!(world.active_id(), document);
    super::super::workspace_layout::browser_feature::expect_solid_visible(&mut world);
    assert_eq!(
        std::fs::read(source).unwrap(),
        before,
        "source fixture changed"
    );
    world.close();
}
