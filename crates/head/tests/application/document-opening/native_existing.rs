// SPDX-License-Identifier: GPL-2.0-only
//! Reopen an already open part with native keyboard and pointer input (#3889).
use super::selections::{
    drawing_on_top, drawing_tab_kept, location_known, no_extra_tab, part_open, part_tab_active,
};
use super::*;

#[test]
fn native_input_activates_existing_part_without_an_extra_tab() {
    let mut world = ApplicationWorld::new();
    part_open(&mut world);
    drawing_on_top(&mut world);
    location_known(&mut world);
    super::super::x11_input::press_control_o();
    settle(&mut world);
    world.expect_text("Look in");
    let folder = models_folder(&mut world);
    look_in(&mut world, &folder);
    native_click_run(&mut world, PART_FILE);
    world.save_capture("native-open-existing-selected-3889");
    native_click_run(&mut world, "Open");
    world.expect_no_text("Look in");
    part_tab_active(&mut world);
    no_extra_tab(&mut world);
    drawing_tab_kept(&mut world);
    world.save_capture("native-open-existing-3889");
    world.close();
}

fn native_click_run(world: &mut ApplicationWorld, label: &str) {
    let center = world.run(label).center();
    let position = super::super::diagnostics::pointer_pixels(center);
    super::super::x11_input::click(position);
    settle(world);
}

#[test]
fn native_input_opens_saved_drawing_with_its_sheet_and_view() {
    use super::drawing_fixture::{drawing_available, sheet_and_border};
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    drawing_available(&mut world);
    super::super::x11_input::press_control_o();
    settle(&mut world);
    world.expect_text("Look in");
    let folder = models_folder(&mut world).join("Caps_Fixture 2");
    std::fs::create_dir_all(&folder).unwrap();
    std::fs::rename(
        world.saved_path.as_ref().unwrap(),
        folder.join(DRAWING_FILE),
    )
    .unwrap();
    native_enter_folder(&mut world, &folder);
    native_click_run(&mut world, DRAWING_FILE);
    native_click_run(&mut world, "Open");
    world.expect_no_text("Look in");
    let drawing = world.active_id();
    assert_ne!(drawing, part);
    assert!(world.state().drawing_documents.contains_key(&drawing));
    sheet_and_border(&mut world);
    let part_name = world.document_name(part);
    world.expect_tab(&part_name);
    world.save_capture("native-open-drawing-3889");
    world.close();
}

fn native_enter_folder(world: &mut ApplicationWorld, folder: &std::path::Path) {
    let label = world.run("Look in").clone();
    let field = [label.maximum[0] + 60., label.center()[1]];
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(field));
    settle(world);
    super::super::x11_input::press_control_a();
    settle(world);
    for character in folder.to_str().unwrap().chars() {
        super::super::x11_input::press_path_character(character);
        world.pump(Vec::new());
    }
    super::super::x11_input::press_keysym(0xff0d);
    settle(world);
    world.expect_text(DRAWING_FILE);
}
