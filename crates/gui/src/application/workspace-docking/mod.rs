// SPDX-License-Identifier: GPL-2.0-only
//! The shared workspace dockspace and its pass-through document area.
use crate::application::document_creation::workspaces::WorkspaceArea;
use imgui::{Condition, StyleVar, Ui, WindowFlags};
pub(crate) mod persistence;

/// Draw the shared docking targets and return the remaining document area.
/// Example: `draw(ui, workspace)` after the ribbon has reserved its band.
pub(crate) fn draw(ui: &Ui, workspace: WorkspaceArea) -> WorkspaceArea {
    let size = [
        (workspace.right - workspace.left).max(1.),
        (workspace.bottom - workspace.top).max(1.),
    ];
    let _padding = ui.push_style_var(StyleVar::WindowPadding([0., 0.]));
    ui.window("##workspace-docking-host")
        .position([workspace.left, workspace.top], Condition::Always)
        .size(size, Condition::Always)
        .flags(host_flags())
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
