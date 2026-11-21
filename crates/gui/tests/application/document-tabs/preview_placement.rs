// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn prefers_above_and_uses_below_when_above_is_too_small() {
    let viewport = [800., 600.];
    let desired = [200., 120.];
    let above = preview_placement(viewport, [[50., 300.], [150., 325.]], desired, 40.).unwrap();
    assert_eq!(above.position, [50., 172.]);
    assert_eq!(above.size, desired);
    let below = preview_placement(viewport, [[50., 50.], [150., 75.]], desired, 40.).unwrap();
    assert_eq!(below.position, [50., 83.]);
    assert_eq!(below.size, desired);
}

#[test]
fn shifts_horizontally_for_left_and_right_edges() {
    let viewport = [300., 300.];
    let desired = [200., 100.];
    let left = preview_placement(viewport, [[-20., 150.], [50., 175.]], desired, 30.).unwrap();
    let right = preview_placement(viewport, [[270., 150.], [330., 175.]], desired, 30.).unwrap();
    assert_eq!(left.position[0], 0.);
    assert_eq!(right.position[0], 100.);
}

#[test]
fn partially_clipped_tabs_keep_the_preview_inside_the_viewport() {
    let viewport = [300., 200.];
    let left = preview_placement(viewport, [[-40., -5.], [50., 20.]], [180., 90.], 30.).unwrap();
    let right =
        preview_placement(viewport, [[270., 175.], [340., 210.]], [180., 90.], 30.).unwrap();
    assert_eq!(left.position, [0., 28.]);
    assert_eq!(right.position, [120., 77.]);
    assert!(right.position[1] + right.size[1] < 175.);
}

#[test]
fn reduces_height_without_crossing_the_tab_or_viewport() {
    let viewport = [320., 180.];
    let tab = [[40., 80.], [160., 100.]];
    let placed = preview_placement(viewport, tab, [200., 200.], 50.).unwrap();
    assert_eq!(placed.position, [40., 0.]);
    assert_eq!(placed.size, [200., 72.]);
    assert!(placed.position[1] + placed.size[1] <= tab[0][1]);
    assert!(preview_placement(viewport, tab, [200., 200.], 73.).is_none());
}

#[test]
fn refuses_a_viewport_too_narrow_for_the_requested_window() {
    assert!(preview_placement([100., 200.], [[0., 100.], [80., 120.]], [101., 50.], 30.).is_none());
    assert!(preview_placement([0., 200.], [[0., 100.], [0., 120.]], [1., 50.], 30.).is_none());
}
