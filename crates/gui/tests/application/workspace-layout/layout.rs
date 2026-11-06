// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn a_header_released_near_a_side_edge_docks_there() {
    let floating = [500., 300.];
    assert_eq!(drop_target(10., 1000., floating), BrowserDock::Left);
    assert_eq!(drop_target(990., 1000., floating), BrowserDock::Right);
    assert_eq!(
        drop_target(500., 1000., floating),
        BrowserDock::Floating(floating)
    );
}
#[test]
fn the_tab_strip_avoids_only_a_docked_visible_browser() {
    let mut layout = WorkspaceLayout::default();
    assert_eq!(layout.tab_span(1000.), (316., 684.));
    layout.dock = BrowserDock::Right;
    assert_eq!(layout.tab_span(1000.), (8., 676.));
    layout.dock = BrowserDock::Floating([50., 50.]);
    assert_eq!(layout.tab_span(1000.), (8., 984.));
    layout.dock = BrowserDock::Left;
    layout.browser_visible = false;
    assert_eq!(layout.tab_span(1000.), (8., 984.));
}
