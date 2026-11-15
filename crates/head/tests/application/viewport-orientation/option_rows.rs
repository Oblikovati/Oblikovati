// SPDX-License-Identifier: GPL-2.0-only
//! Orientation scenarios on the options dialog's scope and view planes.
use super::*;
// Visibility scope
const NEW_WINDOW: &str = "Show on window creation";
const ALL_VIEWS: &str = "All 3D Views";
const CURRENT_VIEW: &str = "Only in Current View";
#[given("the orientation control options are open")]
fn options_open(world: &mut ApplicationWorld) {
    open_options(world);
}
#[given("showing the orientation control on window creation is enabled")]
fn shows_on_new(world: &mut ApplicationWorld) {
    assert!(world.checked(NEW_WINDOW), "expected it checked");
}
#[given("Only in Current View is selected")]
#[then("Only in Current View remains selected")]
fn current_selected(world: &mut ApplicationWorld) {
    assert!(
        world.radio_selected(CURRENT_VIEW),
        "expected {CURRENT_VIEW}"
    );
}
#[when("I clear the show-on-window-creation option")]
#[when("the user clears the show-on-window-creation option")]
#[when("I enable the show-on-window-creation option")]
#[when("the user enables the show-on-window-creation option")]
fn toggle_new_window(world: &mut ApplicationWorld) {
    click_row(world, NEW_WINDOW);
}
#[then("the All 3D Views and Only in Current View choices are unavailable")]
fn scope_off(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled(ALL_VIEWS), "{ALL_VIEWS}");
    assert!(!world.run_is_enabled(CURRENT_VIEW), "{CURRENT_VIEW}");
}
#[then("the position, size and inactive opacity controls remain available")]
fn placement_on(world: &mut ApplicationWorld) {
    for row in ["On-screen Position", "Size", "Inactive Opacity"] {
        assert!(world.run_is_enabled(row), "{row}");
    }
}
#[then("both visibility scope choices are available")]
fn scope_on(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled(ALL_VIEWS), "{ALL_VIEWS}");
    assert!(world.run_is_enabled(CURRENT_VIEW), "{CURRENT_VIEW}");
}
#[when("I select All 3D Views")]
#[when("the user selects All 3D Views")]
fn select_all(world: &mut ApplicationWorld) {
    click_row(world, ALL_VIEWS);
}
#[then("All 3D Views is selected and Only in Current View is cleared")]
fn all_selected(world: &mut ApplicationWorld) {
    assert!(world.radio_selected(ALL_VIEWS), "expected {ALL_VIEWS}");
    assert!(
        !world.radio_selected(CURRENT_VIEW),
        "expected {CURRENT_VIEW} clear"
    );
}
#[when("I select Only in Current View")]
#[when("the user selects Only in Current View")]
fn select_current(world: &mut ApplicationWorld) {
    click_row(world, CURRENT_VIEW);
}
#[then("Only in Current View is selected and All 3D Views is cleared")]
fn current_only(world: &mut ApplicationWorld) {
    assert!(
        world.radio_selected(CURRENT_VIEW),
        "expected {CURRENT_VIEW}"
    );
    assert!(
        !world.radio_selected(ALL_VIEWS),
        "expected {ALL_VIEWS} clear"
    );
}
#[when("I cancel the options")]
#[when("the user cancels the options")]
fn cancel_options(world: &mut ApplicationWorld) {
    click_row(world, "Cancel");
}
#[then("the options close and the part view remains displayed")]
fn options_closed(world: &mut ApplicationWorld) {
    world.expect_no_text(OPTIONS_TITLE);
    control_geometry();
    world.expect_blank_region([400, 600], [250, 300]);
}

#[test]
fn orientation_options_title_close_discards_pending_scope() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_options(&mut world);
    current_selected(&mut world);
    select_all(&mut world);
    all_selected(&mut world);
    let (position, size) = diagnostics::window_geometry(OPTIONS_TITLE).unwrap();
    let close = diagnostics::pointer_pixels([position[0] + size[0] - 13., position[1] + 10.]);
    world.click(close);
    assert!(diagnostics::window_geometry(OPTIONS_TITLE).is_none());
    open_options(&mut world);
    current_selected(&mut world);
    cancel_options(&mut world);
    world.close();
}
// View planes
fn expect_plane(world: &ApplicationWorld, row: &str, plane: &str) {
    let label = world.run(row).clone();
    let shown = world.frame.runs.iter().any(|run| {
        run.text == plane
            && run.maximum[0] < label.minimum[0]
            && (run.center()[1] - label.center()[1]).abs() < 4.
    });
    assert!(shown, "expected {row} to display {plane}");
}
fn expect_planes(world: &ApplicationWorld, front: &str, top: &str) {
    expect_plane(world, "Front View Plane", front);
    expect_plane(world, "Top View Plane", top);
}
#[given("the Front View Plane is XY(+Z) and the Top View Plane is XZ(+Y)")]
fn starting_planes(world: &mut ApplicationWorld) {
    expect_planes(world, "XY(+Z)", "XZ(+Y)");
}
#[when("I open the Front View Plane list")]
#[when("the user opens the Front View Plane list")]
fn open_front_list(world: &mut ApplicationWorld) {
    choose_list_row(world, "Front View Plane", "");
}
#[then("it offers XY(+Z), XY(-Z), XZ(+Y), XZ(-Y), YZ(+X) and YZ(-X)")]
fn plane_rows(world: &mut ApplicationWorld) {
    for row in ["XY(+Z)", "XY(-Z)", "XZ(+Y)", "XZ(-Y)", "YZ(+X)", "YZ(-X)"] {
        world.expect_text(row);
    }
}
#[when("I select XZ(+Y) as the Front View Plane")]
#[when("the user selects XZ(+Y) as the Front View Plane")]
fn select_front_plane(world: &mut ApplicationWorld) {
    // The list's row, over the field behind it that shows the same words.
    click_row(world, "XZ(+Y)");
}
#[then("the Front View Plane displays XZ(+Y)")]
fn front_now_top(world: &mut ApplicationWorld) {
    expect_plane(world, "Front View Plane", "XZ(+Y)");
}
#[then("the Top View Plane displays XY(+Z)")]
fn top_now_front(world: &mut ApplicationWorld) {
    expect_plane(world, "Top View Plane", "XY(+Z)");
}
#[when("I cancel the options and reopen them")]
#[when("the user cancels the options and reopens them")]
fn cancel_and_reopen(world: &mut ApplicationWorld) {
    click_row(world, "Cancel");
    open_options(world);
}
#[then("the Front View Plane displays XY(+Z)")]
fn front_restored(world: &mut ApplicationWorld) {
    expect_plane(world, "Front View Plane", "XY(+Z)");
}
#[then("the Top View Plane displays XZ(+Y)")]
fn top_restored(world: &mut ApplicationWorld) {
    expect_plane(world, "Top View Plane", "XZ(+Y)");
}
