// SPDX-License-Identifier: GPL-2.0-only
//! Native browser interaction with the visible rendered sketch row.
use super::dockspace_restart::native_click_label;
use super::*;

fn right_click_profile(world: &mut ApplicationWorld) {
    let point = world.run("Profile").center();
    super::super::x11_input::right_click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}

#[test]
fn sketch_context_uses_clicked_row_for_visibility_and_edit() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let before = world.state().browser.unwrap().sketches[0].clone();
    right_click_profile(&mut world);
    let browser = world.state().browser.unwrap();
    assert!(browser.selected_sketches.contains(&before.id));
    native_click_label(&mut world, "Visibility");
    assert_eq!(
        world.state().browser.unwrap().sketches[0].visible,
        !before.visible
    );
    right_click_profile(&mut world);
    native_click_label(&mut world, "Edit Sketch");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.save_capture("browser-sketch-context-edit");
    world.close();
}
