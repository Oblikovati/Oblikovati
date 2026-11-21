// SPDX-License-Identifier: GPL-2.0-only
//! Prepare the documented native part for window-frame scenarios.
use super::ApplicationWorld;

/// Open a disposable copy of the visible-solid part; example: `open_frame_part(world)`.
pub(in super::super) fn open_frame_part(world: &mut ApplicationWorld) -> u64 {
    let source = super::super::scenario_root().join("fixtures/cad/main-window-frame-start.opd");
    let copy = world.scratch_folder().join("main-window-frame-source.opd");
    std::fs::copy(&source, &copy).unwrap_or_else(|error| {
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
    let document = open_frame_part(world);
    world.start_sketch();
    world.finish_sketch();
    assert!(world.state().document_storage[&document].unsaved_edits);
    document
}

#[cfg(target_os = "linux")]
#[test]
fn native_frame_fixture_opens_with_a_solid_and_accepts_a_dirty_edit() {
    let source = super::super::scenario_root().join("fixtures/cad/main-window-frame-start.opd");
    let before = std::fs::read(&source).unwrap();
    let mut world = ApplicationWorld::new();
    let document = open_dirty_frame_part(&mut world);
    assert_eq!(world.active_id(), document);
    super::super::workspace_layout::browser_feature::expect_solid_visible(&mut world);
    assert_eq!(
        std::fs::read(source).unwrap(),
        before,
        "source fixture changed"
    );
    world.close();
}
