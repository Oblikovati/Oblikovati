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
#[test]
fn a_dragged_browser_width_moves_the_document_tabs() {
    let mut layout = WorkspaceLayout::default();
    let start = layout.tab_span(1440.);
    layout.browser_width_override = Some(400.);
    let wider = layout.tab_span(1440.);
    assert!(wider.0 > start.0 && wider.1 < start.1);
    layout.browser_width_override = None;
    assert_eq!(layout.tab_span(1440.), start);
}
#[test]
fn the_document_area_leaves_room_for_the_tabs_only_when_they_show() {
    let mut layout = WorkspaceLayout::default();
    let with_tabs = layout.document_area([1440., 900.], 100.);
    layout.tabs_visible = false;
    let without = layout.document_area([1440., 900.], 100.);
    assert!(without.top < with_tabs.top);
    assert_eq!(without.bottom, with_tabs.bottom);
}
