// SPDX-License-Identifier: GPL-2.0-only
//! Verify View menu marking visibility in part and sketch contexts (#4134).
use super::*;

#[test]
fn marking_menu_toggle_preserves_part_commands_and_restart() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    set_marking(&mut world, false);
    assert_part_menu(&mut world, false, true);
    world.reopen_application();
    world.add_part();
    assert_part_menu(&mut world, false, false);
    set_marking(&mut world, true);
    assert_part_menu(&mut world, true, true);
    assert_part_menu(&mut world, true, false);
    world.close();
}

#[test]
fn marking_menu_toggle_preserves_sketch_commands_and_cancel() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.start_sketch();
    set_marking(&mut world, false);
    assert_sketch_menu(&mut world, false, true);
    assert_sketch_menu(&mut world, false, false);
    set_marking(&mut world, true);
    assert_sketch_menu(&mut world, true, true);
    assert_sketch_menu(&mut world, true, false);
    world.close();
}

fn set_marking(world: &mut ApplicationWorld, enabled: bool) {
    super::dockspace_restart::native_click_label(world, "View");
    super::dockspace_restart::native_click_label(world, "User Interface");
    assert_ne!(world.checked("Marking Menu"), enabled);
    super::dockspace_restart::native_click_label(world, "Marking Menu");
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
}

fn open_canvas(world: &mut ApplicationWorld) {
    super::super::x11_input::move_pointer([900, 450]);
    world.pump(Vec::new());
    super::super::x11_input::right_click([900, 450]);
    world.pump(Vec::new());
}

fn canvas_row<'a>(
    world: &'a ApplicationWorld,
    label: &str,
) -> &'a super::super::diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text
                .chars()
                .filter(|character| !character.is_whitespace())
                .eq(label.chars().filter(|character| !character.is_whitespace()))
                && run.center()[0] > 700.
                && run.center()[1] > 250.
        })
        .unwrap_or_else(|| {
            panic!(
                "canvas menu {label:?}: expected visible command in {:?}",
                world.text()
            )
        })
}

fn assert_menu_layout(world: &ApplicationWorld, first: &str, second: &str, radial: bool) {
    let a = canvas_row(world, first);
    let b = canvas_row(world, second);
    assert_eq!(
        (a.minimum[0] - b.minimum[0]).abs() > 10.,
        radial,
        "menu {a:?} {b:?}: expected radial={radial}"
    );
}

fn select_canvas_command(world: &mut ApplicationWorld, label: &str) {
    let point = super::super::diagnostics::pointer_pixels(canvas_row(world, label).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}

fn cancel(world: &mut ApplicationWorld) {
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
}

fn assert_part_menu(world: &mut ApplicationWorld, radial: bool, choose: bool) {
    open_canvas(world);
    assert_menu_layout(world, "Start 2D Sketch", "Extrude", radial);
    if choose {
        select_canvas_command(world, "Extrude");
        assert!(window_geometry("##plane-selection").is_some());
    }
    cancel(world);
    assert!(window_geometry("##part-canvas-menu").is_none());
    assert!(window_geometry("##plane-selection").is_none());
}

fn assert_sketch_menu(world: &mut ApplicationWorld, radial: bool, choose: bool) {
    open_canvas(world);
    assert_menu_layout(world, "Line", "Circle", radial);
    if choose {
        select_canvas_command(world, "Line");
        assert!(!world.state().sketch_at_ready);
    }
    cancel(world);
    assert!(window_geometry("##sketch-canvas-menu").is_none());
    assert!(world.state().sketch_at_ready);
}
