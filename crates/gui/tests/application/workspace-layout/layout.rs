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
fn the_viewport_span_avoids_only_a_docked_visible_browser() {
    let mut layout = WorkspaceLayout::default();
    assert_eq!(layout.tab_span(1000.), (250., 750.));
    layout.dock = BrowserDock::Right;
    assert_eq!(layout.tab_span(1000.), (0., 750.));
    layout.dock = BrowserDock::Floating([50., 50.]);
    assert_eq!(layout.tab_span(1000.), (0., 1000.));
    layout.dock = BrowserDock::Left;
    layout.panels.model_open = false;
    assert_eq!(layout.tab_span(1000.), (42., 950.));
}
#[test]
fn a_dragged_browser_width_moves_the_viewport() {
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
    assert_eq!(without.top, with_tabs.top);
    assert!(without.bottom > with_tabs.bottom);
    assert_eq!(without.bottom - with_tabs.bottom, TAB_STRIP_HEIGHT);
}
#[test]
fn a_docked_ribbon_narrows_the_document_area_from_its_own_side() {
    let mut layout = WorkspaceLayout::default();
    let (start, span) = layout.tab_span(1000.);
    layout.reserve_for_ribbon(196., 0.);
    assert_eq!(layout.tab_span(1000.), (start + 196., span - 196.));
    layout.reserve_for_ribbon(0., 196.);
    assert_eq!(layout.tab_span(1000.), (start, span - 196.));
}

#[test]
fn selected_origin_plane_is_consumed_and_does_not_cross_documents() {
    let mut layout = WorkspaceLayout::default();
    layout.sync_origin_document(Some(1));
    layout.view.selected_origin_row = Some(0);
    assert_eq!(
        layout.take_selected_origin_plane(),
        Some(crate::GuiOriginPlaneKind::Yz)
    );
    assert_eq!(layout.take_selected_origin_plane(), None);
    layout.view.selected_origin_row = Some(1);
    layout.sync_origin_document(Some(1));
    assert_eq!(
        layout.take_selected_origin_plane(),
        Some(crate::GuiOriginPlaneKind::Xz)
    );
    layout.view.selected_origin_row = Some(2);
    assert_eq!(
        layout.take_selected_origin_plane(),
        Some(crate::GuiOriginPlaneKind::Xy)
    );
    layout.view.selected_origin_row = Some(2);
    layout.sync_origin_document(Some(2));
    assert_eq!(layout.take_selected_origin_plane(), None);
    layout.view.selected_origin_row = Some(3);
    assert_eq!(layout.take_selected_origin_plane(), None);
    assert_eq!(layout.view.selected_origin_row, Some(3));
}

#[test]
fn the_bottom_tab_strip_follows_the_viewport_and_browser_dock() {
    let mut layout = WorkspaceLayout::default();
    assert_eq!(layout.tab_strip_span(1000.), (250., 750.));
    layout.dock = BrowserDock::Right;
    assert_eq!(layout.tab_strip_span(1000.), (0., 750.));
    layout.reserve_for_ribbon(196., 0.);
    assert_eq!(layout.tab_strip_span(1000.), (196., 554.));
    assert_eq!(layout.tab_top(900.), 844.);
    layout.tabs_visible = false;
    assert_eq!(layout.tab_top(900.), 878.);
}
