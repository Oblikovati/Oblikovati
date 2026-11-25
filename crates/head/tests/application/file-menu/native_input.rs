// SPDX-License-Identifier: GPL-2.0-only
//! Native keyboard and mouse checks for File menu dismissal.
use super::super::{diagnostics, viewport_orientation, x11_input};
use super::saved_part::{expect_saved_menu_part_on_home, open_saved_menu_part};
use super::*;
use std::path::Path;

const EMPTY_VIEWPORT_PIXEL: [f32; 2] = [1000., 600.];

fn settle(world: &mut ApplicationWorld) {
    for _ in 0..4 {
        world.pump(Vec::new());
    }
}

fn open_with_mouse(world: &mut ApplicationWorld) {
    let location = diagnostics::pointer_pixels(world.run("File").center());
    x11_input::click(location);
    settle(world);
    world.expect_text("Recent documents");
}

#[test]
fn native_file_menu_footer_matches_commands_and_opens_options() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    open_with_mouse(&mut world);
    world.save_capture("file-menu-footer");
    let labels = ["New", "Open...", "Options...", "Exit"];
    let [new, open, options, exit] = labels.map(|name| world.run(name).minimum);
    assert!(
        (open[1] - new[1] - 32.).abs() <= 1.,
        "New {new:?}, Open {open:?}"
    );
    assert!(
        (exit[1] - options[1] - 32.).abs() <= 1.,
        "Options {options:?}, Exit {exit:?}"
    );
    let left_edges = [new, open, options, exit].map(|point| point[0]);
    assert!((left_edges[1] - left_edges[2]).abs() < 1.);
    assert!(
        left_edges
            .iter()
            .all(|left| (left - left_edges[0]).abs() <= 2.)
    );

    for (previous, next) in [
        ("By Ordered List", "By Access Date"),
        ("Name", "Name and Location"),
    ] {
        world.click(diagnostics::pointer_pixels(world.run(previous).center()));
        world.expect_text(next);
    }
    world.click(diagnostics::pointer_pixels(
        world.run("Options...").center(),
    ));
    world.expect_text("Application Options");
    world.close();
}

#[test]
fn native_escape_and_outside_click_preserve_the_active_part() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    open_with_mouse(&mut world);
    world.save_capture("file-menu-native-open-3894");
    x11_input::press_escape();
    settle(&mut world);
    world.expect_no_text("Recent documents");
    assert_eq!(world.active_id(), part);
    open_with_mouse(&mut world);
    x11_input::click(diagnostics::pointer_pixels(EMPTY_VIEWPORT_PIXEL));
    settle(&mut world);
    world.expect_no_text("Recent documents");
    assert_eq!(world.active_id(), part);
    world.close();
}

#[test]
fn saved_part_and_home_show_contextual_file_commands_with_x11_input() {
    let mut world = ApplicationWorld::new();
    let fixture = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/planar-sketch-empty-xy.opd");
    let directory = tempfile::tempdir().unwrap();
    // Keep the whole name visible in Home's narrow Recent Name column.
    let path = directory.path().canonicalize().unwrap().join("Menu.opd");
    let before = std::fs::read(fixture).unwrap();
    std::fs::write(&path, &before).unwrap();
    open_saved_menu_part(&mut world, &path);
    inspect_saved_part_file_menu(&mut world);
    inspect_home_file_menu(&mut world);
    assert_eq!(std::fs::read(path).unwrap(), before);
    assert_eq!(world.state().documents.len(), 1);
    world.close();
}

fn inspect_saved_part_file_menu(world: &mut ApplicationWorld) {
    open_with_mouse(world);
    let at = diagnostics::pointer_pixels(world.run("Save As").center());
    x11_input::move_pointer(at);
    settle(world);
    for command in ["Save Copy As", "Save Copy As Template", "Pack and Go"] {
        world.expect_text(command);
        assert!(
            !world.run_is_enabled(command),
            "expected {command} disabled"
        );
    }
    world.save_capture("saved-native-part-save-as");
    x11_input::click(diagnostics::pointer_pixels(EMPTY_VIEWPORT_PIXEL));
    settle(world);
    world.expect_no_text("Save Copy As");
    world.expect_no_text("Recent documents");
}

fn inspect_home_file_menu(world: &mut ApplicationWorld) {
    let at = diagnostics::pointer_pixels(world.expect_tab("Home").center());
    x11_input::move_pointer(at);
    world.pump(Vec::new());
    x11_input::click(at);
    settle(world);
    assert!(
        diagnostics::window_geometry("##home-workspace").is_some(),
        "expected the Home workspace after clicking its tab"
    );
    expect_saved_menu_part_on_home(world);
    open_with_mouse(world);
    for command in ["Save As", "Document Properties", "Print"] {
        assert!(!world.run_is_enabled(command), "expected {command} muted");
    }
    expect_saved_menu_part_on_home(world);
    world.save_capture("saved-native-part-recent-home");
}
