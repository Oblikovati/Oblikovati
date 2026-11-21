// SPDX-License-Identifier: GPL-2.0-only
//! The Relationships dialog: reading and walking what a row depends on.
use super::*;

const XY: &str = "XY Plane";
const CENTER: &str = "Center Point";

/// Everything from one heading down to the next, or to the dialog's end.
fn section(world: &ApplicationWorld, heading: &str, next: &str) -> String {
    let top = world.run(heading).center()[1];
    let bottom = world.run(next).center()[1];
    let rows = world.frame.runs.iter().filter(|run| {
        let y = run.center()[1];
        y > top && y < bottom
    });
    rows.map(|run| run.text.as_str()).collect()
}
fn parents(world: &ApplicationWorld) -> String {
    section(world, "Parents", "Children")
}
fn children(world: &ApplicationWorld) -> String {
    section(world, "Children", "Done")
}
fn expect_selected(world: &ApplicationWorld, name: &str) {
    world.expect_text(&format!("Selected: {name}"));
}
fn expect_lists(list: String, rows: &[&str]) {
    for row in rows {
        assert!(
            list.contains(&row.replace(' ', "")),
            "expected {row} in {list:?}"
        );
    }
}
fn expect_empty(list: String) {
    assert!(list.contains("None"), "expected no rows in {list:?}");
}
/// The rows of a dialog list, as drawn in the Relationships window.
fn row_in_section(world: &ApplicationWorld, name: &str, heading: &str, next: &str) -> [f32; 2] {
    let top = world.run(heading).center()[1];
    let bottom = world.run(next).center()[1];
    let wanted = name.replace(' ', "");
    let row = world.frame.runs.iter().rfind(|run| {
        let y = run.center()[1];
        run.text == wanted && y > top && y < bottom
    });
    row.unwrap_or_else(|| panic!("expected {name} under {heading}"))
        .center()
}
fn choose_row(world: &mut ApplicationWorld, name: &str, heading: &str, next: &str) {
    let at = row_in_section(world, name, heading, next);
    world.click(at.map(|value| value as i32));
    settle(world);
}
/// The navigation control stands just left of its row.
fn navigate(world: &mut ApplicationWorld, name: &str, heading: &str, next: &str) {
    let name_run = name.replace(' ', "");
    let top = world.run(heading).center()[1];
    let bottom = world.run(next).center()[1];
    let run = world
        .frame
        .runs
        .iter()
        .rfind(|run| run.text == name_run && run.center()[1] > top && run.center()[1] < bottom)
        .cloned()
        .unwrap_or_else(|| panic!("expected {name} under {heading}"));
    world.click([run.minimum[0] as i32 - 18, run.center()[1] as i32]);
    settle(world);
}
fn open_on_row(world: &mut ApplicationWorld, row: &str) {
    right_click_row(world, row);
    choose_menu_row(world, "Relationships");
    settle(world);
}
#[given("the sketch references the center point")]
fn sketch_uses_center(world: &mut ApplicationWorld) {
    world.expect_text(CENTER);
}
#[when("I choose Relationships from the sketch row context menu")]
#[when("the user chooses Relationships from the sketch row context menu")]
fn from_sketch(world: &mut ApplicationWorld) {
    open_on_row(world, SKETCH);
}
#[then("the Relationships dialog identifies the sketch as the selected item")]
#[then("the dialog identifies the sketch as the selected item again")]
#[then("the sketch becomes the selected item in the dialog")]
fn sketch_selected(world: &mut ApplicationWorld) {
    expect_selected(world, SKETCH);
}
#[then("its Parents list contains the XY plane and center point")]
#[then("the Parents list contains the XY plane and center point")]
fn sketch_parents(world: &mut ApplicationWorld) {
    expect_lists(parents(world), &[XY, CENTER]);
}
#[then("its Children list contains the extrusion")]
#[then("the Children list contains the extrusion")]
fn sketch_children(world: &mut ApplicationWorld) {
    expect_lists(children(world), &[FEATURE]);
}
#[when("I select the XY plane in the Parents list")]
#[when("the user selects the XY plane in the Parents list")]
fn choose_plane(world: &mut ApplicationWorld) {
    choose_row(world, XY, "Parents", "Children");
}
#[then("the XY plane row is highlighted in the model browser")]
fn plane_row_lit(world: &mut ApplicationWorld) {
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == XY.replace(' ', ""))
        .cloned()
        .expect("expected the row in the tree");
    let at = [run_x(&row), row.center()[1] as usize];
    assert_ne!(
        world.pixel_at(at),
        world.pixel_at([at[0], 700]),
        "expected the accent"
    );
}
fn run_x(run: &diagnostics::RenderedRun) -> usize {
    run.maximum[0] as usize + 20
}
#[then("the plane is highlighted in the viewport")]
fn plane_tinted(world: &mut ApplicationWorld) {
    assert!(plane_shows_at(world, XY), "expected the plane in the view");
}
#[when("I select the extrusion in the Children list")]
#[when("the user selects the extrusion in the Children list")]
fn choose_extrusion(world: &mut ApplicationWorld) {
    choose_row(world, FEATURE, "Children", "Done");
}
#[then("the extrusion row and its solid edges are highlighted")]
#[then("the extrusion row and block edges are highlighted")]
fn extrusion_lit(world: &mut ApplicationWorld) {
    expect_outline(world, true);
}
#[when("I activate the navigation control beside the extrusion")]
#[when("the user activates the navigation control beside the extrusion")]
fn go_to_extrusion(world: &mut ApplicationWorld) {
    navigate(world, FEATURE, "Children", "Done");
}
#[then("the dialog identifies the extrusion as the selected item")]
fn extrusion_selected(world: &mut ApplicationWorld) {
    expect_selected(world, FEATURE);
}
#[then("its Parents list contains the XY plane and the sketch")]
fn extrusion_parents(world: &mut ApplicationWorld) {
    expect_lists(parents(world), &[XY, SKETCH]);
}
#[then("its Children list is empty")]
#[then("the Children list is empty")]
fn no_children(world: &mut ApplicationWorld) {
    expect_empty(children(world));
}
#[when("I activate the navigation control beside the sketch in the Parents list")]
#[when("the user activates the navigation control beside the sketch in the Parents list")]
fn go_to_sketch_parent(world: &mut ApplicationWorld) {
    navigate(world, SKETCH, "Parents", "Children");
}
#[then("the Relationships dialog closes")]
fn dialog_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Done");
}

#[cfg(target_os = "linux")]
#[test]
fn native_done_closes_relationships_and_keeps_the_part() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    let document = world.active_id();
    from_xy(&mut world);
    xy_selected(&mut world);
    let done = world.run_in_window("Relationships", "Done").center();
    super::super::x11_input::click(diagnostics::pointer_pixels(done));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Relationships").is_none());
    world.expect_active(document);
    world.close();
}
// From a plane
#[when("I choose Relationships from the XY plane row context menu")]
#[when("the user chooses Relationships from the XY plane row context menu")]
fn from_xy(world: &mut ApplicationWorld) {
    open_on_row(world, XY);
}
#[then("the dialog identifies the XY plane as the selected item")]
#[then("the XY plane becomes the selected item again")]
fn xy_selected(world: &mut ApplicationWorld) {
    expect_selected(world, XY);
}
#[then("the Parents list is empty")]
fn no_parents(world: &mut ApplicationWorld) {
    expect_empty(parents(world));
}
#[then("the Children list contains the sketch and the extrusion")]
fn plane_children(world: &mut ApplicationWorld) {
    expect_lists(children(world), &[SKETCH, FEATURE]);
}
#[when("I activate the navigation control beside the sketch")]
#[when("the user activates the navigation control beside the sketch")]
fn go_to_sketch(world: &mut ApplicationWorld) {
    navigate(world, SKETCH, "Children", "Done");
}
#[when("I activate the navigation control beside the XY plane")]
#[when("the user activates the navigation control beside the XY plane")]
fn go_to_plane(world: &mut ApplicationWorld) {
    navigate(world, XY, "Parents", "Children");
}
#[when("I choose Relationships from the YZ plane row context menu")]
#[when("the user chooses Relationships from the YZ plane row context menu")]
fn from_yz(world: &mut ApplicationWorld) {
    open_on_row(world, "YZ Plane");
}
#[then("the dialog identifies the YZ plane as the selected item")]
fn yz_selected(world: &mut ApplicationWorld) {
    expect_selected(world, "YZ Plane");
}
#[then("the YZ plane remains highlighted in the viewport")]
fn yz_stays(world: &mut ApplicationWorld) {
    assert!(
        plane_shows_at(world, "YZ Plane"),
        "expected the plane in the view"
    );
}
