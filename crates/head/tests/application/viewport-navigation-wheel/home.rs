// SPDX-License-Identifier: GPL-2.0-only
//! Establish and observe the home orientation through the native orientation control.
use super::super::viewport_orientation::{expect_control_shows, press_part};
use super::*;
use oblikovati_viewport::ViewportCamera;

fn direction(view: ViewportCamera) -> [f64; 3] {
    let delta: [f64; 3] = std::array::from_fn(|axis| view.eye[axis] - view.target[axis]);
    let length = delta.iter().map(|value| value * value).sum::<f64>().sqrt();
    delta.map(|value| value / length)
}
fn alignment(a: [f64; 3], b: [f64; 3]) -> f64 {
    a.into_iter().zip(b).map(|(x, y)| x * y).sum()
}
#[given("the home view has a known three-face orientation")]
fn known_home(world: &mut ApplicationWorld) {
    press_part(world, |layout| layout.home);
    expect_control_shows(world, "TopFrontRight");
    solid_extent(world);
    world.observed_view_camera = Some(camera(world));
}
#[given("the current view differs from the home view")]
fn view_differs(world: &mut ApplicationWorld) {
    press_part(world, |layout| {
        [layout.center[0], layout.center[1] - layout.half * 0.85]
    });
    expect_control_shows(world, "Top");
    let home = world
        .observed_view_camera
        .expect("expected established home camera");
    assert!(alignment(direction(home), direction(camera(world))) < 0.99);
}
#[when("I right-click the wheel and choose Go Home")]
#[when("the user right-clicks the wheel and chooses Go Home")]
fn choose_home(world: &mut ApplicationWorld) {
    right_click_wheel(world);
    choose_row(world, "Go Home");
}
#[then("the block returns to the known home orientation")]
fn at_home(world: &mut ApplicationWorld) {
    expect_control_shows(world, "TopFrontRight");
    let home = world
        .observed_view_camera
        .expect("expected established home camera");
    let now = camera(world);
    assert!(alignment(direction(home), direction(now)) > 0.9999);
    assert!(alignment(home.up, now.up) > 0.9999);
}

#[test]
fn home_fixture_recovers_three_face_view_from_a_different_view() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    expect_control_shows(&world, "Front");
    known_home(&mut world);
    view_differs(&mut world);
    wheel_is_open(&mut world);
    choose_home(&mut world);
    at_home(&mut world);
    world.close();
}
