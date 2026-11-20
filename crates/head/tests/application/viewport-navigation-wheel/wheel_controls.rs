// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the full wheel's visible controls with native pointer input.
use super::*;

fn control_center(world: &ApplicationWorld, label: &str) -> [i32; 2] {
    let (position, size) = wheel_geometry();
    let run = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let at = run.center();
            run.text == label
                && at[0] >= position[0]
                && at[0] <= position[0] + size[0]
                && at[1] >= position[1]
                && at[1] <= position[1] + 30.
        })
        .unwrap_or_else(|| panic!("wheel control {label:?}: expected a visible label"));
    diagnostics::pointer_pixels(run.center())
}

fn expect_distinct_wheel_labels(world: &ApplicationWorld) {
    let labels = [
        "Zoom", "Orbit", "Pan", "Rewind", "Center", "Walk", "Look", "Up Down",
    ];
    let runs: Vec<_> = labels.iter().map(|label| world.run(label)).collect();
    for (index, run) in runs.iter().enumerate() {
        assert_eq!(
            run.text.replace(' ', ""),
            labels[index].replace(' ', ""),
            "wheel label {}",
            labels[index]
        );
        for other in &runs[index + 1..] {
            let overlap = (0..2).all(|axis| {
                run.minimum[axis] < other.maximum[axis] && other.minimum[axis] < run.maximum[axis]
            });
            assert!(
                !overlap,
                "wheel labels {:?} and {:?}: expected separate bounds",
                run.text, other.text
            );
        }
    }
}

#[test]
fn visible_menu_and_close_buttons_operate_the_wheel() {
    let mut world = ApplicationWorld::new();
    open_wheel_on_block(&mut world);
    expect_distinct_wheel_labels(&world);
    world.save_capture("full-wheel-controls");
    let menu = control_center(&world, "v");
    super::super::x11_input::click(menu);
    settle(&mut world);
    world.expect_text("Close Wheel");
    super::super::x11_input::press_keysym(0xff1b);
    settle(&mut world);
    world.expect_no_text("Close Wheel");
    wheel_geometry();
    let close = control_center(&world, "x");
    super::super::x11_input::click(close);
    settle(&mut world);
    assert!(
        diagnostics::window_geometry(WHEEL).is_none(),
        "expected wheel closed"
    );
    world.expect_ready();
    world.close();
}
