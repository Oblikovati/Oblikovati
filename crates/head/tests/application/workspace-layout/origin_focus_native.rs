// SPDX-License-Identifier: GPL-2.0-only
//! Verify Origin row focus remains visible after viewport selection clears (#4140).
use super::*;
use cucumber::then;

#[test]
fn origin_focus_outline_survives_clearing_and_moves_with_selection() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    select_and_clear(&mut world, "Y Axis");
    assert_focus_outline(&mut world, "Y Axis");
    select_and_clear(&mut world, "Center Point");
    assert_focus_outline(&mut world, "Center Point");
    assert_no_focus_outline(&mut world, "Y Axis");
}

fn select_and_clear(world: &mut ApplicationWorld, label: &str) {
    let row = super::super::diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::right_click(row);
    world.pump(Vec::new());
    let extent = world.extent();
    let point = [(extent[0] * 3 / 4) as i32, (extent[1] * 3 / 4) as i32];
    for _ in 0..2 {
        super::super::x11_input::click(point);
        world.pump(Vec::new());
    }
    assert!(
        !world.row_highlighted(label),
        "{label}: expected no filled selection"
    );
}

fn focus_outline_pixels(world: &mut ApplicationWorld, label: &str) -> usize {
    let row = world.run(label).clone();
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let min = super::super::diagnostics::pointer_pixels([row.minimum[0] - 5., row.minimum[1] - 5.]);
    let max =
        super::super::diagnostics::pointer_pixels([row.maximum[0] + 25., row.maximum[1] + 5.]);
    let mut count = 0;
    for y in min[1].max(0)..max[1].min(capture.extent[1] as i32) {
        for x in min[0].max(0)..max[0].min(capture.extent[0] as i32) {
            let index = (y as usize * capture.extent[0] as usize + x as usize) * 4;
            let pixel = &capture.rgba[index..index + 4];
            if pixel[2] > pixel[0].saturating_add(40) && pixel[1] > pixel[0].saturating_add(20) {
                count += 1;
            }
        }
    }
    count
}

fn assert_focus_outline(world: &mut ApplicationWorld, label: &str) {
    assert!(
        focus_outline_pixels(world, label) > 10,
        "{label}: expected a visible focus outline"
    );
}

fn assert_no_focus_outline(world: &mut ApplicationWorld, label: &str) {
    assert_eq!(
        focus_outline_pixels(world, label),
        0,
        "{label}: expected focus to move"
    );
}

#[then("the axis row retains a focus outline")]
fn axis_focus_outline(world: &mut ApplicationWorld) {
    assert_focus_outline(world, "Y Axis");
}

#[then("the Center Point row retains a focus outline")]
fn center_focus_outline(world: &mut ApplicationWorld) {
    assert_focus_outline(world, "Center Point");
}
