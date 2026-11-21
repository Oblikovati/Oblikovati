// SPDX-License-Identifier: GPL-2.0-only
//! A non-interactive projection of the model axes into the current view.
use super::{GuiViewAxes, Ui, WindowFlags, WorkspaceArea};

const SIDE: f32 = 90.;
const ARM: f32 = 28.;
const COLORS: [[f32; 4]; 3] = [
    [0.92, 0.18, 0.18, 1.],
    [0.20, 0.80, 0.28, 1.],
    [0.20, 0.48, 1.00, 1.],
];

/// Draw the world axes in the view; example: `draw(ui, area, &camera_axes)`.
pub(super) fn draw(ui: &Ui, area: WorkspaceArea, axes: &GuiViewAxes) {
    let side = SIDE
        .min((area.right - area.left).max(1.))
        .min((area.bottom - area.top).max(1.));
    let position = [area.left, area.bottom - side];
    ui.window("##viewport-axis-triad")
        .position(position, imgui::Condition::Always)
        .size([side, side], imgui::Condition::Always)
        .flags(flags())
        .build(|| draw_axes(ui, axes, position, side));
}

fn draw_axes(ui: &Ui, axes: &GuiViewAxes, position: [f32; 2], side: f32) {
    let center = [position[0] + side * 0.5, position[1] + side * 0.5];
    let list = ui.get_window_draw_list();
    for (axis, label) in ["X", "Y", "Z"].into_iter().enumerate() {
        let tip = axis_tip(center, axes, axis, side.min(SIDE) * ARM / SIDE);
        list.add_line(center, tip, COLORS[axis])
            .thickness(2.)
            .build();
        list.add_circle(tip, 3., COLORS[axis]).filled(true).build();
        list.add_text([tip[0] + 4., tip[1] - 8.], COLORS[axis], label);
    }
}

fn axis_tip(center: [f32; 2], axes: &GuiViewAxes, axis: usize, reach: f32) -> [f32; 2] {
    [
        center[0] + axes.right[axis] * reach,
        center[1] - axes.up[axis] * reach,
    ]
}

fn flags() -> WindowFlags {
    WindowFlags::NO_DECORATION
        | WindowFlags::NO_MOVE
        | WindowFlags::NO_DOCKING
        | WindowFlags::NO_BACKGROUND
        | WindowFlags::NO_SAVED_SETTINGS
        | WindowFlags::NO_FOCUS_ON_APPEARING
        | WindowFlags::NO_INPUTS
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn front_axes_project_right_up_and_at_the_origin() {
        let axes = GuiViewAxes {
            right: [1., 0., 0.],
            up: [0., 1., 0.],
            toward: [0., 0., 1.],
        };
        assert_eq!(axis_tip([45., 45.], &axes, 0, ARM), [73., 45.]);
        assert_eq!(axis_tip([45., 45.], &axes, 1, ARM), [45., 17.]);
        assert_eq!(axis_tip([45., 45.], &axes, 2, ARM), [45., 45.]);
    }
    #[test]
    fn projection_follows_the_view_basis() {
        let axes = GuiViewAxes {
            right: [0., 0., -1.],
            up: [0., 1., 0.],
            toward: [1., 0., 0.],
        };
        assert_eq!(axis_tip([45., 45.], &axes, 0, ARM), [45., 45.]);
        assert_eq!(axis_tip([45., 45.], &axes, 2, ARM), [17., 45.]);
    }
}
