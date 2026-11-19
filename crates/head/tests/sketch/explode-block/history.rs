// SPDX-License-Identifier: GPL-2.0-only
//! Verify the sampled block colors through native history and selection.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

#[given("a committed block explosion")]
fn committed_explosion(world: &mut ApplicationWorld) {
    super::explode_block::block_definition_with_instance(world);
    super::explode_block::explode_instance(world);
    world.record_initial_sketch();
}
#[when("the user undo and clear selection")]
fn undo_and_clear(world: &mut ApplicationWorld) {
    world.undo();
    world.click([1150, 650]);
    world.pump(Vec::new());
}
#[then("Block1:1 and the grouped Cyan appearance return")]
fn instance_returns(world: &mut ApplicationWorld) {
    world.expect_text("Block1:1");
    let [line, circle] = sample_points(world);
    expect_color(world, line, ColorKind::Cyan);
    expect_color(world, circle, ColorKind::Cyan);
}
#[then("the instance disappears and the purple rectangle and Red inner circle return")]
fn members_return(world: &mut ApplicationWorld) {
    world.expect_no_text("Block1:1");
    let [line, circle] = sample_points(world);
    expect_color(world, line, ColorKind::Purple);
    expect_color(world, circle, ColorKind::Red);
}
fn sample_points(world: &mut ApplicationWorld) -> [[i32; 2]; 2] {
    let geometry = &world.observed_sketch.as_ref().unwrap().geometry;
    let line = geometry.lines()[0];
    let [a, b] = line
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    let circle = geometry.circles()[0];
    let center = geometry.point(circle.center).unwrap().position;
    let points = [
        [(a[0] + b[0]) / 2.0, (a[1] + b[1]) / 2.0],
        [center[0] + circle.radius, center[1]],
    ];
    points.map(|point| world.sketch_pixel_of(point))
}
#[derive(Clone, Copy, Debug)]
enum ColorKind {
    Cyan,
    Purple,
    Red,
}
fn expect_color(world: &mut ApplicationWorld, pixel: [i32; 2], color: ColorKind) {
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let found = (-4..=4).any(|dy| {
        (-4..=4).any(|dx| pixel_matches(&capture, [pixel[0] + dx, pixel[1] + dy], color))
    });
    assert!(found, "expected {color:?} near {pixel:?}");
}
fn pixel_matches(
    capture: &oblikovati_viewport::ViewportCapture,
    [x, y]: [i32; 2],
    color: ColorKind,
) -> bool {
    let Ok((x, y)) = usize::try_from(x).and_then(|x| usize::try_from(y).map(|y| (x, y))) else {
        return false;
    };
    if x >= capture.extent[0] as usize || y >= capture.extent[1] as usize {
        return false;
    }
    let at = (y * capture.extent[0] as usize + x) * 4;
    let [r, g, b] = [capture.rgba[at], capture.rgba[at + 1], capture.rgba[at + 2]];
    match color {
        ColorKind::Cyan => g > 150 && b > 150 && r < 80,
        ColorKind::Purple => r > 90 && b > 150 && g < 80,
        ColorKind::Red => r > 150 && g < 80 && b < 80,
    }
}
