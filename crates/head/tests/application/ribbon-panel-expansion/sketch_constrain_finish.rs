// SPDX-License-Identifier: GPL-2.0-only
//! Native finish and keyboard paths for Sketch ribbon expansion.
use super::*;

#[cfg(target_os = "linux")]
#[test]
fn x11_finishing_sketch_closes_constrain_area() {
    let mut world = ApplicationWorld::new();
    part_with_sketch(&mut world);
    sketch_visible_outside_edit(&mut world);
    let row = diagnostics::pointer_pixels(world.run(SKETCH).center());
    super::super::super::x11_input::right_click(row);
    world.pump(Vec::new());
    click_native_label(&mut world, "Edit Sketch");
    sketch_commands_available(&mut world);
    click_native_label(&mut world, "Constrain");
    constrain_commands_visible(&mut world);
    click_native_label(&mut world, "Finish Sketch");
    returns_to_part(&mut world);
    constrain_area_closes(&mut world);
    world.close();
}
#[cfg(target_os = "linux")]
fn click_native_label(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run(label).center());
    super::super::super::x11_input::click(point);
    world.pump(Vec::new());
}

#[when("the user presses Alt, then S, then D")]
fn open_create_by_keys(world: &mut ApplicationWorld) {
    use oblikovati_viewport::ViewportPhysicalKey;
    for key in [
        ViewportPhysicalKey::LeftAlt,
        ViewportPhysicalKey::S,
        ViewportPhysicalKey::D,
    ] {
        world.key(key, false);
    }
}
#[then("the Create additional-command area opens")]
fn create_area_visible(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-additional-Create").is_some());
}
#[then("the Create additional-command area closes")]
fn create_area_closes(world: &mut ApplicationWorld) {
    create_closed(world);
}
#[then("the full Sketch ribbon is displayed")]
fn full_sketch_ribbon_displayed(world: &mut ApplicationWorld) {
    world.expect_text("Constrain");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
}
#[cfg(target_os = "linux")]
#[test]
fn x11_sketch_tab_closes_unpinned_modify_expansion() {
    let mut world = ApplicationWorld::new();
    part_with_sketch(&mut world);
    let part = world.active_id();
    super::super::opened_unpinned(&mut world);
    let tab = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Sketch")
        .unwrap();
    super::super::super::x11_input::click(diagnostics::pointer_pixels(tab.center()));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    full_sketch_ribbon_displayed(&mut world);
    world.expect_active(part);
    expect_solid_visible(&mut world);
    world.close();
}
