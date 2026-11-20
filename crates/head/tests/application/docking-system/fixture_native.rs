// SPDX-License-Identifier: GPL-2.0-only
//! Save and reopen a visible solid and its sketch for docking observations.
use super::ApplicationWorld;

#[test]
fn docking_solid_and_visible_sketch_survive_save_and_reopen() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    show_profile(&mut world);
    assert_fixture_visible(&mut world);
    let document = world.active_id();
    let path = world.save_part_to_file(document, "docking-visible-solid.opd");
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    assert_fixture_visible(&mut world);
    world.save_capture("docking-visible-solid-reopened");
    world.close_active_tab();
    let fixture = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/docking-visible-solid.opd");
    world.reopen_through_dialog(&fixture);
    assert_fixture_visible(&mut world);
    if let Some(output) = std::env::var_os("OBK_EXPORT_DOCKING_FIXTURE") {
        export_fixture(&path, &std::path::PathBuf::from(output));
    }
    world.close();
}

fn show_profile(world: &mut ApplicationWorld) {
    let point = world.run("Profile").center();
    super::super::x11_input::right_click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    world.click_text("Visibility");
    world.pump(Vec::new());
}

fn assert_fixture_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected model browser");
    assert_eq!(browser.sketches.len(), 1);
    assert!(browser.sketches[0].visible, "expected visible sketch");
    world.expect_text("Extrude1");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(super::super::scene_tests::solid_pixels(&capture) > 0);
}

fn export_fixture(source: &std::path::Path, output: &std::path::Path) {
    use std::io::Write;
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(output)
        .unwrap();
    file.write_all(&std::fs::read(source).unwrap()).unwrap();
}
