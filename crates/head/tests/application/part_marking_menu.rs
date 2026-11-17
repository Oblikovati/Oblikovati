// SPDX-License-Identifier: GPL-2.0-only
//! Native pointer coverage for configured Part marking-menu commands (#4049).
use super::{ApplicationWorld, diagnostics, x11_input};

fn start_extrude_from_part_menu(world: &mut ApplicationWorld) {
    x11_input::focus_window();
    x11_input::move_pointer([900, 450]);
    world.pump(Vec::new());
    x11_input::right_click([900, 450]);
    world.pump(Vec::new());
    let button = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Extrude" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("Part menu: expected Extrude; rendered {:?}", world.text()));
    let point = diagnostics::pointer_pixels(button.center());
    x11_input::click(point);
    world.pump(Vec::new());
}

fn click_native_label(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run(label).center());
    x11_input::click(point);
    world.pump(Vec::new());
}

#[test]
fn native_part_menu_extrude_starts_sketch_when_part_has_none() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    start_extrude_from_part_menu(&mut world);
    assert!(
        diagnostics::window_geometry("##plane-selection").is_some(),
        "expected plane-selection overlay"
    );
    let plane = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.replace(' ', "") == "XYPlane" && run.center()[0] > 310.)
        .unwrap_or_else(|| panic!("expected XY plane label in viewport: {:?}", world.text()))
        .center();
    x11_input::click(diagnostics::pointer_pixels(plane));
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    click_native_label(&mut world, "Rectangle");
    for point in super::documents::RECTANGLE {
        x11_input::click(point);
        world.pump(Vec::new());
    }
    click_native_label(&mut world, "Finish Sketch");
    world.pump(Vec::new());
    world.expect_text("Distance A");
    assert!(world.state().extrude_preview_active);
    click_native_label(&mut world, "Cancel");
    assert!(!world.state().extrude_preview_active);
    world.close();
}

#[test]
fn native_part_menu_extrude_opens_dialog_for_finished_sketch() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    start_extrude_from_part_menu(&mut world);
    world.expect_text("Distance A");
    world.expect_text("Profiles");
    click_native_label(&mut world, "Cancel");
    world.expect_no_text("Distance A");
    world.close();
}
