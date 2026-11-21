// SPDX-License-Identifier: GPL-2.0-only
//! Enter and clear a Recent search through the native desktop input path (#3898).
use super::super::{diagnostics::pointer_pixels, toolbar_icon_center, x11_input};
use super::{
    ASSEMBLY, ApplicationWorld, OTHERS, SEARCHED, home_run, home_shows, save_recent_files,
};

#[test]
fn home_recent_grid_filters_and_restores_with_x11_input() {
    let mut world = ApplicationWorld::new();
    save_recent_files(&mut world);
    x11_input::focus_window();
    x11_input::click(toolbar_icon_center("Home"));
    world.pump(Vec::new());
    click_recent_control(&mut world, "Grid View");
    type_recent_search(&mut world, "part26");
    assert!(home_shows(&world, SEARCHED));
    for name in [OTHERS[0], OTHERS[1], ASSEMBLY] {
        assert!(!home_shows(&world, name), "expected {name} filtered out");
    }
    world.save_capture("x11-home-grid-search");
    click_recent_control(&mut world, "x");
    click_recent_control(&mut world, "Search");
    for name in [SEARCHED, OTHERS[0], OTHERS[1], ASSEMBLY] {
        assert!(home_shows(&world, name), "expected {name} restored");
    }
    world.save_capture("x11-home-grid-restored");
    world.close();
}

fn type_recent_search(world: &mut ApplicationWorld, query: &str) {
    click_recent_control(world, "Search recent");
    for character in query.chars() {
        x11_input::press_keysym(u32::from(character));
        world.pump(Vec::new());
    }
    click_recent_control(world, "Search");
}

fn click_recent_control(world: &mut ApplicationWorld, label: &str) {
    let control = home_run(world, label).unwrap_or_else(|| panic!("expected {label:?} in Home"));
    x11_input::click(pointer_pixels(control.center()));
    world.pump(Vec::new());
}
