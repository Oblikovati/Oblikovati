// SPDX-License-Identifier: GPL-2.0-only
//! Native orientation cube navigation scenarios.
use super::*;

// View the top of a part and return to the home view
#[given("a part containing a rectangular block is active in the MCAD Software")]
#[given("a rectangular block is active in the MCAD Software")]
fn given_block(world: &mut ApplicationWorld) {
    open_block(world);
}
#[given("its home view shows the block isometrically")]
fn home_iso(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.home);
    expect_control_shows(world, "TopFrontRight");
}
#[given("the viewport orientation control is visible")]
#[given("the orientation control is visible in the viewport")]
fn control_visible(_: &mut ApplicationWorld) {
    control_geometry();
}
#[when("I select the Top face of the orientation control")]
#[when("the user selects the Top face of the orientation control")]
fn select_top(world: &mut ApplicationWorld) {
    press_top_in_iso(world);
}
#[then("the viewport shows the top of the block face-on as a rectangle")]
pub(super) fn top_face_on(world: &mut ApplicationWorld) {
    // 40 mm wide by 20 mm deep from above.
    let [wide, deep] = solid_extent(world);
    expect_proportion(wide, deep, 2, 1);
    expect_camera_direction(world, [0., 1., 0.]);
}
#[then("the orientation control displays Top")]
fn displays_top(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Top");
}
#[then("the same part and model browser remain active")]
fn same_part(world: &mut ApplicationWorld) {
    expect_same_part(world);
}
#[when("I select the orientation control's Home action")]
#[when("the user selects the orientation control's Home action")]
fn press_home(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.home);
}
#[then("the viewport returns to the block's isometric home view")]
fn back_home(world: &mut ApplicationWorld) {
    expect_control_shows(world, "TopFrontRight");
    solid_extent(world);
}
// Step between adjacent orthogonal views
#[given("the orientation control displays the Front view")]
#[given("the orientation control displays Front with its label upright")]
#[given("the orientation control displays an upright Front view")]
pub(super) fn front_displayed(world: &mut ApplicationWorld) {
    press_at_front(world);
    expect_control_shows(world, "Front");
    assert!(label_is_upright(world), "expected an upright label");
}
#[when("I select the right-side navigation arrow beside the orientation control")]
#[when("the user selects the right-side navigation arrow beside the orientation control")]
fn step_right(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.step_right);
}
#[then("the control displays Right")]
fn displays_right(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Right");
}
#[then("the block is displayed face-on from the right")]
fn right_face_on(world: &mut ApplicationWorld) {
    // 20 mm deep by 30 mm tall from the side.
    let [wide, tall] = solid_extent(world);
    expect_proportion(wide, tall, 2, 3);
    expect_camera_direction(world, [1., 0., 0.]);
}
#[when("I select the left-side navigation arrow")]
#[when("the user selects the left-side navigation arrow")]
fn step_left(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.step_left);
}
#[then("the control displays Front")]
fn displays_front(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Front");
}
#[then("the block returns to the face-on front view")]
fn front_face_on(world: &mut ApplicationWorld) {
    // 40 mm wide by 30 mm tall from the front.
    let [wide, tall] = solid_extent(world);
    expect_proportion(wide, tall, 4, 3);
    expect_camera_direction(world, [0., 0., 1.]);
}
// Rotate and restore a face-on view
#[given("the block's front face is wider than it is tall on screen")]
#[then("the block's front face is wider than it is tall again")]
fn wider(world: &mut ApplicationWorld) {
    let [wide, tall] = solid_extent(world);
    assert!(wide > tall, "extent {wide} by {tall}");
}
#[when("I select the upper curved rotation arrow on the orientation control")]
#[when("the user selects the upper curved rotation arrow on the orientation control")]
fn roll_upper(world: &mut ApplicationWorld) {
    world.observed_view_camera = Some(super::super::viewport_pan::camera(world));
    press_part(world, |parts| parts.roll_upper);
}
#[then("the control still displays Front with its label rotated by a quarter turn")]
pub(super) fn label_turned(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Front");
    assert!(!label_is_upright(world), "expected the label turned");
    expect_camera_direction(world, [0., 0., 1.]);
    let before = world.observed_view_camera.unwrap();
    let after = super::super::viewport_pan::camera(world);
    let dot: f64 = (0..3).map(|axis| before.up[axis] * after.up[axis]).sum();
    assert!(
        dot.abs() < 1e-6,
        "expected a quarter-turn, got up vectors {:?} and {:?}",
        before.up,
        after.up
    );
}
#[then("the block's front face is taller than it is wide on screen")]
pub(super) fn taller(world: &mut ApplicationWorld) {
    let [wide, tall] = solid_extent(world);
    assert!(tall > wide, "extent {wide} by {tall}");
}
#[when("I select the other curved rotation arrow")]
#[when("the user selects the other curved rotation arrow")]
fn roll_lower(world: &mut ApplicationWorld) {
    press_part(world, |parts| parts.roll_lower);
}
#[then("the Front label is upright again")]
fn upright_again(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Front");
    assert!(label_is_upright(world), "expected an upright label");
    let before = world.observed_view_camera.unwrap();
    let after = super::super::viewport_pan::camera(world);
    let error: f64 = (0..3)
        .map(|axis| (before.up[axis] - after.up[axis]).abs())
        .sum();
    assert!(
        error < 1e-6,
        "camera up {:?}: expected {:?}",
        after.up,
        before.up
    );
}
// Choose a corner from a face-on view
#[when("I select the upper-right corner of the orientation control")]
#[when("the user selects the upper-right corner of the orientation control")]
fn select_corner(world: &mut ApplicationWorld) {
    let (position, parts) = layout();
    press_at(
        world,
        position,
        [
            parts.center[0] + parts.half * IN_THE_CORNER,
            parts.center[1] - parts.half * IN_THE_CORNER,
        ],
    );
}
#[then("the block is displayed obliquely with its top, front and right faces visible")]
fn block_oblique(world: &mut ApplicationWorld) {
    expect_oblique_faces(world);
    expect_control_shows(world, "TopFrontRight");
    let camera = super::super::viewport_pan::camera(world);
    let direction: [f64; 3] = std::array::from_fn(|axis| camera.eye[axis] - camera.target[axis]);
    assert!(
        direction.iter().all(|value| *value > 0.),
        "expected an oblique top-front-right camera, got {direction:?}"
    );
}
#[then("the orientation control shows the Top, Front and Right faces")]
fn three_faces(world: &mut ApplicationWorld) {
    expect_control_shows(world, "TopFrontRight");
}

pub(super) fn expect_camera_direction(world: &mut ApplicationWorld, expected: [f64; 3]) {
    let camera = super::super::viewport_pan::camera(world);
    let delta: [f64; 3] = std::array::from_fn(|axis| camera.eye[axis] - camera.target[axis]);
    let length = delta.iter().map(|value| value * value).sum::<f64>().sqrt();
    let alignment: f64 = (0..3)
        .map(|axis| delta[axis] * expected[axis] / length)
        .sum();
    assert!(
        alignment > 0.999,
        "camera direction {delta:?}: expected {expected:?}"
    );
}

/// Three sample points lie inside the unequal-sided fixture's visible faces in this corner view.
fn expect_oblique_faces(world: &mut ApplicationWorld) {
    let (low, high) = solid_bounds(world);
    let span = [high[0] - low[0], high[1] - low[1]];
    let sample = |x: usize, y: usize| [low[0] + span[0] * x / 100, low[1] + span[1] * y / 100];
    let top = world.pixel_at(sample(40, 20));
    let front = world.pixel_at(sample(35, 50));
    let right = world.pixel_at(sample(85, 65));
    assert!(
        top[0] > right[0] + 10 && right[0] > front[0] + 20,
        "expected three shaded faces, got top {top:?}, front {front:?}, right {right:?}"
    );
}
