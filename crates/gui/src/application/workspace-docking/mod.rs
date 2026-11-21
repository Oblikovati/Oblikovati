// SPDX-License-Identifier: GPL-2.0-only
//! The shared workspace dockspace and its pass-through document area.
use crate::application::document_creation::workspaces::WorkspaceArea;
use imgui::{Condition, StyleVar, Ui, WindowFlags};
pub(crate) mod persistence;

/// Draw the nested panel docks and return the document viewport.
/// Example: `draw(ui, outer, layout, display, band, false)` once per frame.
pub(crate) fn draw(
    ui: &Ui,
    outer: WorkspaceArea,
    layout: &mut crate::application::workspace_layout::WorkspaceLayout,
    display: [f32; 2],
    band: f32,
    seed_asset_dock: bool,
) -> WorkspaceArea {
    let size = [
        (outer.right - outer.left).max(1.),
        (outer.bottom - outer.top).max(1.),
    ];
    let _padding = ui.push_style_var(StyleVar::WindowPadding([0., 0.]));
    let mut host_open = true;
    <crate::application::workspace_layout::WorkspaceLayout as crate::dockable_window::Window>::window(
        ui,
        "##workspace-outer-host",
        &mut host_open,
    )
        .position([outer.left, outer.top], Condition::Always)
        .size(size, Condition::Always)
        .flags(host_flags())
        .build(|| draw_inner_workspace(ui, outer, size, layout, display, band, seed_asset_dock))
        .unwrap_or(outer)
}

fn draw_inner_workspace(
    ui: &Ui,
    outer: WorkspaceArea,
    size: [f32; 2],
    layout: &mut crate::application::workspace_layout::WorkspaceLayout,
    display: [f32; 2],
    band: f32,
    seed_asset_dock: bool,
) -> WorkspaceArea {
    let (bounds, span) = crate::widget_abi::outer_dockspace(ui, size, seed_asset_dock)
        .and_then(|(bounds, span)| contained_center(outer, bounds).map(|area| (area, span)))
        .unwrap_or((outer, None));
    layout.set_outer_center(bounds, outer);
    layout.outer_bottom_span = span;
    draw_inner_child(ui, outer, layout.document_area(display, band))
}

fn draw_inner_child(ui: &Ui, outer: WorkspaceArea, workspace: WorkspaceArea) -> WorkspaceArea {
    let size = [
        (workspace.right - workspace.left).max(1.),
        (workspace.bottom - workspace.top).max(1.),
    ];
    crate::widget_abi::cursor_position(
        ui,
        [workspace.left - outer.left, workspace.top - outer.top],
    );
    ui.child_window("##workspace-docking-host")
        .size(size)
        .flags(
            WindowFlags::NO_BACKGROUND
                | WindowFlags::NO_SCROLLBAR
                | WindowFlags::NO_SCROLL_WITH_MOUSE,
        )
        .build(|| crate::widget_abi::workspace_dockspace(ui, size))
        .flatten()
        .and_then(|bounds| contained_center(workspace, bounds))
        .unwrap_or(workspace)
}

fn contained_center(workspace: WorkspaceArea, bounds: [f32; 4]) -> Option<WorkspaceArea> {
    let [left, top, right, bottom] = bounds;
    if !bounds.iter().all(|value| value.is_finite())
        || left < workspace.left
        || top < workspace.top
        || right > workspace.right
        || bottom > workspace.bottom
        || left >= right
        || top >= bottom
    {
        return None;
    }
    Some(WorkspaceArea {
        left,
        top,
        right,
        bottom,
    })
}

fn host_flags() -> WindowFlags {
    WindowFlags::NO_DECORATION
        | WindowFlags::NO_MOVE
        | WindowFlags::NO_DOCKING
        | WindowFlags::NO_BACKGROUND
        | WindowFlags::NO_SAVED_SETTINGS
        | WindowFlags::NO_FOCUS_ON_APPEARING
        | WindowFlags::NO_BRING_TO_FRONT_ON_FOCUS
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn center_must_be_finite_nonempty_and_inside_the_workspace() {
        let area = WorkspaceArea {
            left: 100.,
            top: 50.,
            right: 900.,
            bottom: 700.,
        };
        assert_eq!(
            contained_center(area, [300., 50., 900., 700.])
                .unwrap()
                .left,
            300.
        );
        assert!(contained_center(area, [99., 50., 900., 700.]).is_none());
        assert!(contained_center(area, [300., 50., 300., 700.]).is_none());
        assert!(contained_center(area, [f32::NAN, 50., 900., 700.]).is_none());
    }
}
