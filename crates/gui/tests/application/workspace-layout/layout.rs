// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn the_document_span_follows_the_outer_dock_center() {
    let mut layout = WorkspaceLayout::default();
    layout.set_outer_center(
        crate::application::document_creation::workspaces::WorkspaceArea {
            left: 250.,
            top: 100.,
            right: 1000.,
            bottom: 878.,
        },
        crate::application::document_creation::workspaces::WorkspaceArea {
            left: 0.,
            top: 100.,
            right: 1000.,
            bottom: 878.,
        },
    );
    assert_eq!(layout.tab_span(1000.), (250., 750.));
    layout.set_outer_center(
        crate::application::document_creation::workspaces::WorkspaceArea {
            left: 0.,
            top: 100.,
            right: 750.,
            bottom: 878.,
        },
        crate::application::document_creation::workspaces::WorkspaceArea {
            left: 0.,
            top: 100.,
            right: 1000.,
            bottom: 878.,
        },
    );
    assert_eq!(layout.tab_span(1000.), (0., 750.));
}
#[test]
fn the_document_area_follows_all_outer_dock_bounds() {
    let mut layout = WorkspaceLayout::default();
    layout.set_outer_center(
        crate::application::document_creation::workspaces::WorkspaceArea {
            left: 50.,
            top: 200.,
            right: 900.,
            bottom: 700.,
        },
        crate::application::document_creation::workspaces::WorkspaceArea {
            left: 0.,
            top: 100.,
            right: 1000.,
            bottom: 700.,
        },
    );
    let area = layout.document_area([1000., 900.], 100.);
    assert_eq!((area.left, area.top, area.right), (50., 200., 900.));
    assert_eq!(area.bottom, 700. - TAB_STRIP_HEIGHT);
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
fn a_docked_ribbon_narrows_the_outer_dock() {
    let mut layout = WorkspaceLayout::default();
    layout.reserve_for_ribbon(196., 0.);
    assert_eq!(layout.outer_area([1000., 900.], 100.).left, 196.);
    layout.reserve_for_ribbon(0., 196.);
    assert_eq!(layout.outer_area([1000., 900.], 100.).right, 804.);
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
fn outer_bottom_dock_keeps_document_tabs_below_the_panel() {
    use crate::application::document_creation::workspaces::WorkspaceArea;
    let mut layout = WorkspaceLayout::default();
    let root = layout.outer_area([1000., 900.], 100.);
    layout.set_outer_center(
        WorkspaceArea {
            bottom: 600.,
            ..root
        },
        root,
    );
    assert!(layout.pending_layout);
    assert_eq!(layout.outer_area([1000., 900.], 100.).bottom, 850.);
    assert_eq!(layout.document_area([1000., 900.], 100.).bottom, 600.);
    assert_eq!(layout.tab_top(900.), 850.);
}

#[test]
fn tab_row_uses_bottom_dock_width_when_it_spans_the_workspace() {
    use crate::application::document_creation::workspaces::WorkspaceArea;
    let mut layout = WorkspaceLayout::default();
    layout.reserve_for_ribbon(100., 50.);
    let viewport = WorkspaceArea {
        left: 350.,
        top: 100.,
        right: 950.,
        bottom: 500.,
    };
    assert_eq!(layout.document_tab_span(1000., viewport), (350., 600.));
    let root = layout.outer_area([1000., 900.], 100.);
    layout.set_outer_center(
        WorkspaceArea {
            bottom: 500.,
            ..root
        },
        root,
    );
    layout.outer_bottom_span = Some([350., 950.]);
    assert_eq!(layout.document_tab_span(1000., viewport), (350., 600.));
    layout.outer_bottom_span = Some([100., 950.]);
    assert_eq!(layout.document_tab_span(1000., viewport), (100., 850.));
}

#[test]
fn hiding_all_model_panels_closes_the_dock_before_layout() {
    let mut layout = WorkspaceLayout::default();
    let mut registry = crate::dockable_lifecycle::DockRegistry::default();
    layout.register_dockables(&mut registry).unwrap();
    let state = GuiCommandState::default();
    layout.sync_before_docking(&state);
    registry.reconcile(|_| {}).unwrap();
    layout.panels.set_open(
        crate::application::workspace_layout::panel_tabs::DockPanel::Model,
        false,
    );
    layout.panels.set_open(
        crate::application::workspace_layout::panel_tabs::DockPanel::Log,
        false,
    );
    layout.panels.set_open(
        crate::application::workspace_layout::panel_tabs::DockPanel::Favorites,
        false,
    );
    layout.sync_before_docking(&state);
    let mut titles = Vec::new();
    assert!(
        registry
            .reconcile(|title| titles.push(title.to_string_lossy().into_owned()))
            .unwrap()
    );
    assert_eq!(titles, ["Model"]);
}

#[test]
fn origin_focus_is_scoped_to_its_document() {
    let mut layout = WorkspaceLayout::default();
    layout.sync_origin_document(Some(1));
    layout.view.focused_origin_row = Some(4);
    layout.sync_origin_document(Some(1));
    assert_eq!(layout.view.focused_origin_row, Some(4));
    layout.sync_origin_document(Some(2));
    assert_eq!(layout.view.focused_origin_row, None);
}

#[test]
fn status_visibility_changes_reserved_workspace_height() {
    let mut layout = WorkspaceLayout::default();
    let visible = layout.outer_area([1440., 900.], 120.);
    layout.status_bar_visible = false;
    let hidden = layout.outer_area([1440., 900.], 120.);
    assert_eq!(
        hidden.bottom - visible.bottom,
        status_bar::STATUS_BAR_HEIGHT
    );
    assert_eq!(layout.tab_top(900.), 900. - layout.tab_height());
}
