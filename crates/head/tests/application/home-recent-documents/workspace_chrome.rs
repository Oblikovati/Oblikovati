// SPDX-License-Identifier: GPL-2.0-only
//! Verify the visible workspace independently of the document kept open behind Home.
use super::*;

const RIBBON: &str = "##application-ribbon";
const STATUS: &str = "##status-bar";

#[then("the Get Started tab is selected alongside the Tools tab")]
fn get_started_selected(world: &mut ApplicationWorld) {
    home_ribbon(world);
    world.run_in_window(RIBBON, "NewPart");
    assert!(!world.window_text(RIBBON).contains("ApplicationOptions"));
}

#[then("the Home left rail displays New Part, New, and Open actions")]
#[then("the Home left rail shows the same New Part, New, and Open actions as cold startup")]
fn home_rail(world: &mut ApplicationWorld) {
    for label in ["NewPart", "New", "Open"] {
        world.run_in_window(HOME, label);
    }
}

#[then(
    "document-only side docks such as Model, Material Browser, and Appearance Browser are hidden"
)]
#[then("document-only side docks are hidden and the Recent list spans the workspace")]
fn home_spans_workspace(world: &mut ApplicationWorld) {
    for dock in ["Model", "Material Browser", "Appearance Browser"] {
        assert!(window_geometry(dock).is_none(), "{dock} is visible on Home");
    }
    let [left, _, right, _] = world.viewport_bounds();
    assert_eq!([left, right], [0, world.extent()[0]]);
    assert!(home_shows(world, "Recent"));
}

#[then("the Recent list occupies the remaining workspace width with readable columns")]
fn recent_uses_home_area(world: &mut ApplicationWorld) {
    home_spans_workspace(world);
    let name = world.run_in_window(HOME, "Name");
    let location = world.run_in_window(HOME, "Location");
    let modified = world.run_in_window(HOME, "DateModified");
    assert!(name.maximum[0] < location.minimum[0]);
    assert!(location.maximum[0] < modified.minimum[0]);
}

#[then(
    "the ribbon shows the Home-context Get Started and Tools tabs with its display controls intact"
)]
fn home_ribbon(world: &mut ApplicationWorld) {
    world.run_in_window(RIBBON, "GetStarted");
    world.run_in_window(RIBBON, "Tools");
    super::super::ribbon_display::display_control_point(world);
    assert!(!world.window_text(RIBBON).contains("3DModel"));
}

#[then(
    "the application title bar displays the application title without an inactive document name"
)]
fn startup_title(world: &mut ApplicationWorld) {
    world.run_in_window(RIBBON, "Oblikovati");
    assert_eq!(
        world.app().window.native.view.native_title().unwrap(),
        "Oblikovati"
    );
}

#[then("the application title bar reflects Home rather than the inactive part document")]
fn home_title(world: &mut ApplicationWorld) {
    world.run_in_window(RIBBON, "Home—Oblikovati");
    assert_eq!(
        world.app().window.native.view.native_title().unwrap(),
        "Home - Oblikovati"
    );
}

#[then("the status bar clears document viewport zoom telemetry")]
fn home_clears_zoom(world: &mut ApplicationWorld) {
    assert!(window_geometry(STATUS).is_some());
    assert!(!world.window_text(STATUS).contains('%'));
}

#[then("the part's docked panels, ribbon tabs, title bar, and viewport telemetry are restored")]
fn restores_part_chrome(world: &mut ApplicationWorld) {
    let id = world.previous_document.expect("expected the open Part");
    world.expect_active(id);
    world.expect_browser_of(id);
    assert!(window_geometry(HOME).is_none());
    world.run_in_window(RIBBON, "3DModel");
    world.run_in_window(RIBBON, "Tools");
    let title = world.app().window.native.view.native_title().unwrap();
    assert!(title.contains(&world.document_name(id)), "{title:?}");
    assert!(world.window_text(STATUS).contains('%'));
}
