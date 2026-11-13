// SPDX-License-Identifier: GPL-2.0-only
//! The rows under Origin: choosing them, showing them, and walking them with the arrow keys.
use super::*;
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportModifiers, ViewportPhysicalKey};

const YZ: &str = "YZ Plane";
const XZ: &str = "XZ Plane";
const XY: &str = "XY Plane";

fn lit(world: &mut ApplicationWorld, row: &str) -> bool {
    row_accent(world, row)
}
fn expect_plane_chosen(world: &mut ApplicationWorld, row: &str, others: &[&str]) {
    assert!(lit(world, row), "expected the {row} row's accent");
    assert!(plane_shows_at(world, row), "expected {row} in the view");
    for other in others {
        assert!(!lit(world, other), "expected the {other} row plain");
    }
}
fn press_arrow(world: &mut ApplicationWorld, key: ViewportPhysicalKey, shift: bool) {
    let modifiers = ViewportModifiers {
        shift,
        ..Default::default()
    };
    let event = |pressed, modifiers| {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key,
            layout_key: key,
            pressed,
            command: None,
            modifiers,
        })
    };
    world.pump(vec![
        event(true, modifiers),
        event(false, Default::default()),
    ]);
    settle(world);
}
// One plane
#[given("the Origin branch is expanded with its origin planes hidden")]
#[given("the Origin branch is expanded and its planes are hidden")]
fn origin_open(world: &mut ApplicationWorld) {
    world.expect_text(YZ);
    for row in [YZ, XZ, XY] {
        assert!(!plane_shows_at(world, row), "expected {row} hidden");
    }
}
#[given("the YZ origin plane is not visible in the viewport")]
#[given("the YZ origin plane is hidden and its browser row is visible")]
#[given("the origin plane is not persistently visible")]
fn yz_hidden(world: &mut ApplicationWorld) {
    world.expect_text(YZ);
    assert!(!plane_shows_at(world, YZ), "expected the plane hidden");
}
#[when("I select the YZ Plane row")]
#[when("the user selects the YZ Plane row")]
#[when("I select the YZ Plane row in the model browser")]
#[when("the user selects the YZ Plane row in the model browser")]
fn select_yz(world: &mut ApplicationWorld) {
    world.click_text(YZ);
    settle(world);
}
#[given("the YZ Plane row is selected and its plane is highlighted")]
fn yz_selected(world: &mut ApplicationWorld) {
    select_yz(world);
    expect_plane_chosen(world, YZ, &[XZ, XY]);
}
#[then("the row is highlighted")]
fn row_lit(world: &mut ApplicationWorld) {
    assert!(lit(world, YZ), "expected the row's accent");
}
#[then("the plane appears as a translucent blue rectangle in the viewport")]
#[then("the YZ plane is highlighted in the viewport")]
fn yz_tinted(world: &mut ApplicationWorld) {
    assert!(plane_shows_at(world, YZ), "expected the plane in the view");
}
#[then("the temporary plane highlight disappears")]
fn yz_cleared(world: &mut ApplicationWorld) {
    assert!(!plane_shows_at(world, YZ), "expected the plane gone");
}
#[then("the plane highlight disappears and the block remains visible")]
#[then("the plane highlight is cleared and the shaded solid remains visible")]
fn plane_gone_block_stays(world: &mut ApplicationWorld) {
    for row in [YZ, XZ, XY] {
        assert!(!plane_shows_at(world, row), "expected {row} gone");
    }
    expect_solid_visible(world);
}
// Visibility
#[when("I right-click the YZ Plane row again")]
#[when("the user right-clicks the YZ Plane row again")]
fn right_click_yz(world: &mut ApplicationWorld) {
    right_click_row(world, YZ);
}
#[then("the context menu offers Visibility and New Sketch")]
fn plane_menu(world: &mut ApplicationWorld) {
    expect_rows(world, &["Visibility", "New Sketch"]);
}
#[when("I choose Visibility and click empty viewport space")]
#[when("the user chooses Visibility and clicks empty viewport space")]
fn visibility_and_click(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Visibility");
    click_empty_viewport(world);
}
#[then("the plane remains visible as a translucent plane beside the block")]
fn plane_stays(world: &mut ApplicationWorld) {
    assert!(
        plane_shows_at(world, YZ),
        "expected the plane to stay shown"
    );
    expect_solid_visible(world);
}
#[then("the plane is hidden and the block remains visible")]
fn plane_hidden(world: &mut ApplicationWorld) {
    world.escape();
    click_empty_viewport(world);
    assert!(!plane_shows_at(world, YZ), "expected the plane hidden");
    expect_solid_visible(world);
}
// An axis
#[given("the Origin branch is expanded and the X Axis is hidden")]
fn x_axis_hidden(world: &mut ApplicationWorld) {
    world.expect_text("X Axis");
    expect_outline(world, false);
}
#[when("I right-click the X Axis row")]
#[when("the user right-clicks the X Axis row")]
#[when("I right-click the X Axis row again")]
#[when("the user right-clicks the X Axis row again")]
fn right_click_axis(world: &mut ApplicationWorld) {
    right_click_row(world, "X Axis");
}
#[then("the row is highlighted and the axis is highlighted in blue in the viewport")]
fn axis_lit(world: &mut ApplicationWorld) {
    assert!(lit(world, "X Axis"), "expected the row's accent");
    expect_outline(world, true);
}
#[then("Visibility is unchecked and Auto Resize is checked in the context menu")]
fn axis_menu_marks(world: &mut ApplicationWorld) {
    assert!(!world.checked("Visibility"), "expected Visibility clear");
    assert!(world.checked("Auto-Resize"), "expected Auto Resize checked");
}
#[then("the axis remains visible as a line beside the block")]
fn axis_stays(world: &mut ApplicationWorld) {
    let muted = super::super::workspace_layout::browser_feature::colored_pixels(world, MUTED_LINE);
    assert!(muted > 0, "expected the muted axis");
}
#[then("the axis is hidden and the block remains visible")]
fn axis_hidden(world: &mut ApplicationWorld) {
    world.escape();
    click_empty_viewport(world);
    let muted = super::super::workspace_layout::browser_feature::colored_pixels(world, MUTED_LINE);
    assert_eq!(muted, 0, "expected no axis");
    expect_solid_visible(world);
}
const MUTED_LINE: [u8; 3] = [191, 153, 77];
// The center point
#[then("its context menu offers Copy, Measure, Create Note and Visibility")]
fn point_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Copy", "Measure", "Create Note", "Visibility"]);
}
#[then("the menu offers Relationships")]
fn relationships_row(world: &mut ApplicationWorld) {
    world.expect_text("Relationships");
}
// The arrow keys
#[when("I press Down Arrow")]
fn press_down(world: &mut ApplicationWorld) {
    press_arrow(world, ViewportPhysicalKey::DownArrow, false);
}
#[when("I press Up Arrow")]
fn press_up(world: &mut ApplicationWorld) {
    press_arrow(world, ViewportPhysicalKey::UpArrow, false);
}
#[when("I press Left Arrow")]
#[when("the user presses Left Arrow")]
#[when("I press Left Arrow again")]
#[when("the user presses Left Arrow again")]
fn press_left(world: &mut ApplicationWorld) {
    press_arrow(world, ViewportPhysicalKey::LeftArrow, false);
}
#[when("I press Right Arrow")]
#[when("the user presses Right Arrow")]
#[when("I press Right Arrow again")]
#[when("the user presses Right Arrow again")]
fn press_right(world: &mut ApplicationWorld) {
    press_arrow(world, ViewportPhysicalKey::RightArrow, false);
}
#[when("I press Shift and Down Arrow together")]
#[when("the user presses Shift and Down Arrow together")]
#[when("I press Shift and Down Arrow together again")]
#[when("the user presses Shift and Down Arrow together again")]
fn press_shift_down(world: &mut ApplicationWorld) {
    press_arrow(world, ViewportPhysicalKey::DownArrow, true);
}
#[then("the XZ Plane row is selected and the XZ plane is highlighted")]
#[then("the XZ Plane row is selected and its plane is highlighted")]
fn xz_chosen(world: &mut ApplicationWorld) {
    expect_plane_chosen(world, XZ, &[]);
}
#[then("the YZ plane highlight is removed")]
#[then("the YZ plane is no longer highlighted")]
fn yz_gone(world: &mut ApplicationWorld) {
    assert!(!lit(world, YZ), "expected the YZ row plain");
}
#[then("the YZ Plane row is selected and the YZ plane is highlighted again")]
#[then("the YZ Plane row is selected and the YZ plane is highlighted")]
fn yz_chosen(world: &mut ApplicationWorld) {
    expect_plane_chosen(world, YZ, &[XZ]);
}
#[then("the XY Plane row is selected and its plane is highlighted")]
fn xy_chosen(world: &mut ApplicationWorld) {
    expect_plane_chosen(world, XY, &[]);
}
/// The planes overlap on screen, so only the row can say which of them is chosen now.
#[then("the XZ plane is no longer highlighted")]
fn xz_gone(world: &mut ApplicationWorld) {
    assert!(!lit(world, XZ), "expected the XZ row plain");
}
#[then("the Origin parent row is selected and the plane highlight is removed")]
fn parent_chosen(world: &mut ApplicationWorld) {
    assert!(lit(world, "Origin"), "expected the Origin row's accent");
    assert!(!plane_shows_at(world, YZ), "expected the plane gone");
}
#[then("the Origin branch collapses and its children are hidden")]
fn branch_folds(world: &mut ApplicationWorld) {
    world.expect_no_text(YZ);
}
#[then("the Origin branch expands and its children are visible")]
fn branch_opens(world: &mut ApplicationWorld) {
    world.expect_text(YZ);
}
// Folder menu
#[when("I right-click the Origin folder")]
#[when("the user right-clicks the Origin folder")]
fn right_click_origin(world: &mut ApplicationWorld) {
    right_click_row(world, "Origin");
}
#[when("I choose Collapse All Children")]
#[when("the user chooses Collapse All Children")]
fn choose_collapse(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Collapse All Children");
    settle(world);
}
// Show Input
#[when("I choose Show Input from the sketch row context menu")]
#[when("the user chooses Show Input from the sketch row context menu")]
fn show_input(world: &mut ApplicationWorld) {
    right_click_row(world, SKETCH);
    choose_menu_row(world, "Show Input");
    settle(world);
}
#[then("the XY plane row is selected in the model browser")]
fn xy_row_lit(world: &mut ApplicationWorld) {
    assert!(lit(world, XY), "expected the row's accent");
}
#[then("the XY plane is highlighted in blue in the viewport")]
fn xy_tinted(world: &mut ApplicationWorld) {
    assert!(plane_shows_at(world, XY), "expected the plane in the view");
}
