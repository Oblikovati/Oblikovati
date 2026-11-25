// SPDX-License-Identifier: GPL-2.0-only
//! Position and draw the navigation controls beside the ViewCube.
use super::*;
use crate::GuiFrameRequest;
use navigation_bar::{NAVIGATION_SIZE, NAVIGATION_WINDOW};

impl OrientationControl {
    /// The bar keeps to the control's side of the view, under or over it.
    pub(super) fn navigation_bar(
        &mut self,
        ui: &Ui,
        icons: &crate::icons::GuiRibbonIcons,
        request: &GuiFrameRequest,
        area: WorkspaceArea,
        [control, control_size]: [[f32; 2]; 2],
        actions: &mut Vec<GuiShellAction>,
    ) {
        let corner = self.options.corner();
        let x = (control[0] + (control_size[0] - NAVIGATION_SIZE[0]) / 2.)
            .clamp(area.left, (area.right - NAVIGATION_SIZE[0]).max(area.left));
        let below = corner == ScreenCorner::TopLeft || corner == ScreenCorner::TopRight;
        let y = if below {
            control[1] + control_size[1] + MARGIN
        } else {
            control[1] - NAVIGATION_SIZE[1] - MARGIN
        };
        let _surface =
            ui.push_style_color(StyleColor::WindowBg, ui.style_color(StyleColor::PopupBg));
        let _padding = ui.push_style_var(imgui::StyleVar::WindowPadding([4., 8.]));
        let _minimum = ui.push_style_var(imgui::StyleVar::WindowMinSize([NAVIGATION_SIZE[0], 32.]));
        let _rounding = ui.push_style_var(imgui::StyleVar::WindowRounding(4.));
        ui.window(NAVIGATION_WINDOW)
            .position([x, y], imgui::Condition::Always)
            .size(NAVIGATION_SIZE, imgui::Condition::Always)
            .flags(control_flags() - WindowFlags::NO_BACKGROUND)
            .build(|| {
                self.navigation
                    .buttons(ui, icons, &request.commands, request.extent, actions)
            });
    }
}
