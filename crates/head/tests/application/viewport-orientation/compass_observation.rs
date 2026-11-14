// SPDX-License-Identifier: GPL-2.0-only
//! Keep navigation-bar labels out of compass observations.
use super::{ApplicationWorld, control_geometry, diagnostics::RenderedRun};

pub(super) fn compass_label<'a>(
    world: &'a ApplicationWorld,
    letter: &str,
) -> Option<&'a RenderedRun> {
    let (position, size) = control_geometry();
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == letter && within_control(run.center(), position, size))
}

fn within_control(center: [f32; 2], position: [f32; 2], size: [f32; 2]) -> bool {
    (0..2).all(|axis| center[axis] >= position[axis] && center[axis] <= position[axis] + size[axis])
}

#[test]
fn compass_bounds_exclude_navigation_bar_above_or_below() {
    let position = [1200., 100.];
    let size = [110., 110.];
    assert!(within_control([1255., 110.], position, size));
    assert!(within_control([1210., 155.], position, size));
    assert!(!within_control([1255., 70.], position, size));
    assert!(!within_control([1255., 240.], position, size));
    assert!(!within_control([1190., 155.], position, size));
}
