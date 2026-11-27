// SPDX-License-Identifier: GPL-2.0-only
//! Verify the rendered orientation face and world axes against the viewport scenarios.
use super::*;

#[given("the viewport orientation control displays the Front view")]
fn front_view(world: &mut ApplicationWorld) {
    navigation::front_displayed(world);
}

#[when("the user inspects the visible orientation face")]
#[when("the user inspects the viewport corners")]
fn inspect_viewport(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
}

#[then("the square face is labeled FRONT in uppercase")]
fn uppercase_front(world: &mut ApplicationWorld) {
    assert_eq!(control_letters(world), sorted("FRONT"));
    assert!(
        control_runs(world)
            .iter()
            .all(|run| !run.text.chars().any(char::is_lowercase))
    );
}

#[then("no diagonal stroke crosses the face")]
fn no_diagonal(world: &mut ApplicationWorld) {
    // Sample the face paint without a hotspot outline crossing the probes.
    world.hover(AWAY);
    let (position, parts) = layout();
    let center = [position[0] + parts.center[0], position[1] + parts.center[1]];
    let image = super::super::capture_framebuffer(&mut world.app().window.native);
    for distance in [-0.70_f32, -0.55, 0.55, 0.70] {
        let sample = [
            center[0] + parts.half * distance,
            center[1] + parts.half * distance,
        ];
        let a = image_pixel(&image, sample);
        let b = image_pixel(&image, [sample[0] + 3., sample[1]]);
        assert!(
            (0..3).all(|channel| a[channel].abs_diff(b[channel]) < 5),
            "face diagonal {sample:?}: {a:?} versus {b:?}"
        );
    }
}

pub(super) fn image_pixel(
    image: &oblikovati_viewport::ViewportCapture,
    point: [f32; 2],
) -> [u8; 4] {
    let [x, y] = diagnostics::pointer_pixels(point).map(|value| value as usize);
    let index = (y * image.extent[0] as usize + x) * 4;
    image.rgba[index..index + 4].try_into().unwrap()
}

#[then("the block and model browser remain visible")]
fn block_and_browser(world: &mut ApplicationWorld) {
    solid_extent(world);
    expect_same_part(world);
}

#[then("an axis triad appears at the lower-left of the viewport")]
pub(super) fn triad_lower_left(world: &mut ApplicationWorld) {
    let (position, size) =
        diagnostics::window_geometry("##viewport-axis-triad").expect("axis triad");
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    assert!((position[0] - area[0] as f32).abs() <= 1.);
    assert!((position[1] + size[1] - area[3] as f32).abs() <= 1.);
}

fn check_axis(world: &mut ApplicationWorld, offset: [f32; 2], channel: usize) {
    let (position, size) =
        diagnostics::window_geometry("##viewport-axis-triad").expect("axis triad");
    let point = [
        position[0] + size[0] / 2. + offset[0],
        position[1] + size[1] / 2. + offset[1],
    ];
    let image = super::super::capture_framebuffer(&mut world.app().window.native);
    let pixel = image_pixel(&image, point);
    assert!(
        (0..3)
            .filter(|other| *other != channel)
            .all(|other| pixel[channel] > pixel[other] + 30),
        "axis pixel {point:?}: {pixel:?}"
    );
}

#[then("the X axis points right and is red")]
fn red_x(world: &mut ApplicationWorld) {
    check_axis(world, [15., 0.], 0);
}
#[then("the Y axis points up and is green")]
fn green_y(world: &mut ApplicationWorld) {
    check_axis(world, [0., -15.], 1);
}
#[then("the Z axis is indicated in blue at the triad origin")]
fn blue_z(world: &mut ApplicationWorld) {
    check_axis(world, [0., 0.], 2);
}

#[then("the orientation control remains at the upper-right of the viewport")]
fn control_upper_right(world: &mut ApplicationWorld) {
    let (position, size) = control_geometry();
    let extent = world.extent();
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap();
    assert!(position[0] + size[0] <= area[2] as f32);
    assert!(position[0] > (area[0] + area[2]) as f32 / 2.);
    assert!(position[1] < (area[1] + area[3]) as f32 / 2.);
}

#[cfg(target_os = "linux")]
#[test]
fn axis_triad_follows_native_view_rotation() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    navigation::front_displayed(&mut world);
    red_x(&mut world);
    press_part(&mut world, |parts| parts.step_right);
    navigation::expect_camera_direction(&mut world, [1., 0., 0.]);
    check_axis(&mut world, [-15., 0.], 2);
    green_y(&mut world);
    world.close();
}
