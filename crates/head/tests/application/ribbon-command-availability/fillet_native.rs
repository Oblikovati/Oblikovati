// SPDX-License-Identifier: GPL-2.0-only
//! Drive the Sketch Fillet ribbon command with native X11 input.
use super::super::x11_input;
use super::{ApplicationWorld, diagnostics, select_sketch_tab};

#[test]
fn sketch_fillet_from_ribbon_cancels_and_round_trips_in_history() {
    let mut world = rectangle_outside_sketch();
    start_fillet(&mut world);
    let before = world.edited_geometry();
    native_line_click(&mut world, 0);
    escape_fillet(&mut world);
    assert_eq!(world.edited_geometry(), before);
    start_fillet(&mut world);
    native_line_click(&mut world, 0);
    native_line_click(&mut world, 1);
    let rounded = world.edited_geometry();
    assert_eq!(rounded.arcs().len(), 1);
    assert_eq!(rounded.lines().len(), 4);
    capture_fillet(&mut world);
    escape_fillet(&mut world);
    history_round_trip(&mut world, &before, &rounded);
    world.close();
}

fn rectangle_outside_sketch() -> ApplicationWorld {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.start_sketch();
    world.draw_rectangle();
    world.finish_sketch();
    select_sketch_tab(&mut world);
    assert!(world.app().owner.active_sketch_edit().is_none());
    world
}

fn start_fillet(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Fillet"));
    x11_input::focus_window();
    x11_input::click(diagnostics::pointer_pixels(world.run("Fillet").center()));
    world.pump(Vec::new());
    assert!(world.app().owner.active_sketch_edit().is_some());
    assert!(!world.state().sketch_at_ready);
}

fn native_line_click(world: &mut ApplicationWorld, index: usize) {
    x11_input::click(world.line_point(index, 0.5));
    world.pump(Vec::new());
}

fn escape_fillet(world: &mut ApplicationWorld) {
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    assert!(world.state().sketch_at_ready);
}

fn capture_fillet(world: &mut ApplicationWorld) {
    // The focused capture has its own output directory. Repeated gate runs share the default.
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("sketch-fillet-from-ribbon");
    }
}

fn history_round_trip(
    world: &mut ApplicationWorld,
    before: &oblikovati_model::sketch::Sketch2D,
    rounded: &oblikovati_model::sketch::Sketch2D,
) {
    x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
    assert_eq!(&world.edited_geometry(), before);
    x11_input::press_control_letter(b'y');
    world.pump(Vec::new());
    assert_eq!(&world.edited_geometry(), rounded);
}
