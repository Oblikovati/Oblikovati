// SPDX-License-Identifier: GPL-2.0-only
//! The Origin branch's menus, and what a chosen plane or axis shows in the view.
use super::super::ApplicationWorld;
use super::browser_feature::{
    choose_menu_row, click_branch_arrow, expect_menu_beside, expect_outline, expect_rows,
    open_extruded_part, right_click_row,
};
use cucumber::{given, then, when};

const PLANE_TINT: u8 = 80;
/// Far along the plane's own axes, in working units.
const PLANE_PROBE: [f64; 2] = [-3., 4.];
const ORIGIN_CHILDREN: [&str; 3] = ["XY Plane", "X Axis", "Center Point"];
// Origin's own menus. These open on the same part, for the solid the scenarios keep watching.
#[given("the Origin branch is collapsed in the model browser")]
fn origin_collapsed(world: &mut ApplicationWorld) {
    if world.state().document.is_none() {
        open_extruded_part(world);
    }
    if world.shows("X Axis") {
        click_branch_arrow(world, "Origin");
    }
    world.expect_no_text("X Axis");
}
#[when("I right-click Origin")]
#[when("the user right-clicks Origin")]
fn right_click_origin(world: &mut ApplicationWorld) {
    right_click_row(world, "Origin");
}
#[when("I choose Expand All Children")]
#[when("the user chooses Expand All Children")]
fn choose_expand(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Expand All Children");
}
#[then("Origin displays its plane, axis and center-point child rows")]
#[then("the direct plane, axis and center-point rows remain visible under Origin")]
fn origin_children(world: &mut ApplicationWorld) {
    expect_rows(world, &ORIGIN_CHILDREN);
}
#[when("I right-click Origin and choose Collapse All Children")]
#[when("the user right-clicks Origin and chooses Collapse All Children")]
fn collapse_origin(world: &mut ApplicationWorld) {
    right_click_origin(world);
    choose_menu_row(world, "Collapse All Children");
}
fn expect_row_accent(world: &mut ApplicationWorld, label: &str, shown: bool) {
    assert_eq!(world.row_highlighted(label), shown, "{label} row accent");
}
/// A place on the YZ plane clear of the solid, the axes and the browser, as a pixel.
fn plane_probe(world: &mut ApplicationWorld) -> [usize; 2] {
    let plane = oblikovati_model::sketch::SketchPlane::yz();
    let pixel = crate::HeadWindow::sketch_pixel(&world.app().window, plane, PLANE_PROBE).unwrap();
    pixel.map(|value| value as usize)
}
/// The view's background is dark and even: the clear surface brightens it well past this.
fn plane_shows(world: &mut ApplicationWorld) -> bool {
    let at = plane_probe(world);
    let pixel = world.pixel_at(at);
    pixel[1] > PLANE_TINT && pixel[1] > pixel[0]
}
/// A chosen row's accent against the same column well below the tree, which reads plain.
pub(in super::super) fn row_accent(world: &mut ApplicationWorld, label: &str) -> bool {
    world.row_highlighted(label)
}
/// A place on a named origin plane clear of the block, as a pixel; a shown plane tints it.
pub(in super::super) fn plane_shows_at(world: &mut ApplicationWorld, row: &str) -> bool {
    use oblikovati_model::sketch::SketchPlane;
    let plane = match row {
        "XY Plane" => SketchPlane::xy(),
        "XZ Plane" => SketchPlane::xz(),
        _ => SketchPlane::yz(),
    };
    let pixel = crate::HeadWindow::sketch_pixel(&world.app().window, plane, PLANE_PROBE).unwrap();
    let at = pixel.map(|value| value as usize);
    let color = world.pixel_at(at);
    color[1] > PLANE_TINT && color[1] > color[0]
}
#[given("Auto-Resize is enabled for the YZ Plane in the captured session")]
fn enable_plane_auto_resize(world: &mut ApplicationWorld) {
    ensure_auto_resize(world, "YZ Plane");
}
#[given("Auto-Resize is enabled for the Y Axis in the captured session")]
fn enable_axis_auto_resize(world: &mut ApplicationWorld) {
    ensure_auto_resize(world, "Y Axis");
}
fn ensure_auto_resize(world: &mut ApplicationWorld, label: &str) {
    right_click_row(world, label);
    if !world.checked("Auto-Resize") {
        world.click_text("Auto-Resize");
        right_click_row(world, label);
    }
    assert!(
        world.checked("Auto-Resize"),
        "{label}: expected Auto-Resize checked"
    );
    world.escape();
}
#[then("Auto-Resize is checked in this session")]
fn auto_resize_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Auto-Resize"));
}
#[when("I right-click the YZ Plane row")]
#[when("the user right-clicks the YZ Plane row")]
fn right_click_plane(world: &mut ApplicationWorld) {
    right_click_row(world, "YZ Plane");
}
#[then("a context menu opens beside the plane row")]
fn plane_menu_beside(world: &mut ApplicationWorld) {
    expect_menu_beside(world, "YZ Plane", "New Sketch");
}
#[then("the menu offers New Sketch, Measure, Visibility and Auto-Resize")]
fn plane_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["New Sketch", "Measure", "Visibility", "Auto-Resize"],
    );
}
#[then("the plane row is highlighted")]
fn plane_row(world: &mut ApplicationWorld) {
    expect_row_accent(world, "YZ Plane", true);
}
#[then("the corresponding plane is highlighted as a translucent surface in the viewport")]
#[then("the corresponding plane is highlighted as a blue translucent surface in the viewport")]
fn plane_surface(world: &mut ApplicationWorld) {
    assert!(
        plane_shows(world),
        "expected the plane's surface in the view"
    );
}
#[then("the plane row and translucent plane remain highlighted")]
fn plane_remains(world: &mut ApplicationWorld) {
    expect_row_accent(world, "YZ Plane", true);
    assert!(
        plane_shows(world),
        "expected the plane's surface in the view"
    );
}
#[then("the plane selection is cleared")]
fn plane_cleared(world: &mut ApplicationWorld) {
    expect_row_accent(world, "YZ Plane", false);
}
#[then("the highlighted plane is no longer displayed")]
fn plane_gone(world: &mut ApplicationWorld) {
    assert!(!plane_shows(world), "expected no plane surface in the view");
}
#[when("I right-click the Y Axis row")]
#[when("the user right-clicks the Y Axis row")]
fn right_click_axis(world: &mut ApplicationWorld) {
    right_click_row(world, "Y Axis");
}
#[then("a context menu opens beside the axis row")]
fn axis_menu_beside(world: &mut ApplicationWorld) {
    expect_menu_beside(world, "Y Axis", "Copy");
}
#[then("the menu offers Copy, Measure, Visibility and Auto-Resize")]
fn axis_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Copy", "Measure", "Visibility", "Auto-Resize"]);
}
#[then("the axis row is highlighted")]
fn axis_row(world: &mut ApplicationWorld) {
    expect_row_accent(world, "Y Axis", true);
}
#[then("the corresponding axis is highlighted in the viewport")]
fn axis_shown(world: &mut ApplicationWorld) {
    expect_outline(world, true);
}
#[then("the axis row and axis remain highlighted")]
fn axis_remains(world: &mut ApplicationWorld) {
    expect_row_accent(world, "Y Axis", true);
    expect_outline(world, true);
}
#[then("the axis is no longer highlighted in the viewport")]
fn axis_gone(world: &mut ApplicationWorld) {
    expect_outline(world, false);
}
#[then("the axis row no longer has the filled selection highlight")]
fn axis_plain(world: &mut ApplicationWorld) {
    expect_row_accent(world, "Y Axis", false);
}
#[when("I right-click the Center Point row")]
#[when("the user right-clicks the Center Point row")]
fn right_click_point(world: &mut ApplicationWorld) {
    right_click_row(world, "Center Point");
}
#[then("a context menu opens beside the point row")]
fn point_menu_beside(world: &mut ApplicationWorld) {
    expect_menu_beside(world, "Center Point", "Copy");
}
#[then("the menu offers Copy, Measure, Create Note, Visibility and Relationships")]
fn point_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Copy",
            "Measure",
            "Create Note",
            "Visibility",
            "Relationships",
        ],
    );
}
#[then("the Center Point row is highlighted")]
fn point_row(world: &mut ApplicationWorld) {
    expect_row_accent(world, "Center Point", true);
}
#[then("the Center Point row remains highlighted")]
fn point_remains(world: &mut ApplicationWorld) {
    expect_row_accent(world, "Center Point", true);
}
#[then("the Center Point row no longer has the filled selection highlight")]
fn point_plain(world: &mut ApplicationWorld) {
    expect_row_accent(world, "Center Point", false);
}
