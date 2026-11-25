// SPDX-License-Identifier: GPL-2.0-only
//! Position and draw the navigation controls beside the ViewCube.
use super::*;
use crate::GuiFrameRequest;
use navigation_bar::{NAVIGATION_SIZE, NAVIGATION_WINDOW};
use options::ScreenCorner;

pub(super) struct ViewControlPlacement {
    pub(super) cube: [f32; 2],
    pub(super) side: f32,
    pub(super) bar: [[f32; 2]; 2],
}

impl ViewControlPlacement {
    pub(super) fn new(
        area: WorkspaceArea,
        corner: ScreenCorner,
        preferred_side: f32,
        visibility: ViewControlVisibility,
    ) -> Self {
        let area = inset(area);
        let width = (area.right - area.left).max(1.);
        let height = (area.bottom - area.top).max(1.);
        let mut side = preferred_side.min(width).min(height);
        let both = visibility.cube && visibility.navigation;
        let [bar_width, bar_height] = NAVIGATION_SIZE;
        let adjacent =
            both && side + MARGIN + bar_height > height && width >= bar_width + MARGIN + 1.;
        if adjacent {
            side = side.min(width - bar_width - MARGIN);
        } else if both {
            side = side.min((height - MARGIN - 1.).max(1.));
        }
        let left = corner.is_left();
        let top = matches!(corner, ScreenCorner::TopLeft | ScreenCorner::TopRight);
        let cube = [
            if left { area.left } else { area.right - side },
            if top { area.top } else { area.bottom - side },
        ];
        let bar = place_bar(area, cube, side, both, adjacent, [left, top]);
        Self { cube, side, bar }
    }
}

fn inset(area: WorkspaceArea) -> WorkspaceArea {
    let horizontal = ((area.right - area.left - 1.) * 0.5).clamp(0., MARGIN);
    let vertical = ((area.bottom - area.top - 1.) * 0.5).clamp(0., MARGIN);
    WorkspaceArea {
        left: area.left + horizontal,
        top: area.top + vertical,
        right: area.right - horizontal,
        bottom: area.bottom - vertical,
    }
}

fn place_bar(
    area: WorkspaceArea,
    cube: [f32; 2],
    side: f32,
    both: bool,
    adjacent: bool,
    [left, top]: [bool; 2],
) -> [[f32; 2]; 2] {
    let width = NAVIGATION_SIZE[0].min((area.right - area.left).max(1.));
    let height = area.bottom - area.top - if both && !adjacent { side + MARGIN } else { 0. };
    let height = NAVIGATION_SIZE[1].min(height.max(1.));
    let centered =
        (cube[0] + (side - width) / 2.).clamp(area.left, (area.right - width).max(area.left));
    if adjacent {
        let x = if left {
            cube[0] + side + MARGIN
        } else {
            cube[0] - width - MARGIN
        };
        let y = if top { area.top } else { area.bottom - height };
        return [[x, y], [width, height]];
    }
    let y = if top {
        cube[1] + side + MARGIN
    } else {
        cube[1] - height - MARGIN
    };
    [
        [
            centered,
            y.clamp(area.top, (area.bottom - height).max(area.top)),
        ],
        [width, height],
    ]
}

impl OrientationControl {
    /// The bar keeps to the control's side of the view, under or over it.
    pub(super) fn navigation_bar(
        &mut self,
        ui: &Ui,
        icons: &crate::icons::GuiRibbonIcons,
        request: &GuiFrameRequest,
        [position, size]: [[f32; 2]; 2],
        actions: &mut Vec<GuiShellAction>,
    ) {
        let _surface =
            ui.push_style_color(StyleColor::WindowBg, ui.style_color(StyleColor::PopupBg));
        let _padding = ui.push_style_var(imgui::StyleVar::WindowPadding([4., 8.]));
        let _minimum = ui.push_style_var(imgui::StyleVar::WindowMinSize([1., 1.]));
        let _rounding = ui.push_style_var(imgui::StyleVar::WindowRounding(4.));
        ui.window(NAVIGATION_WINDOW)
            .position(position, imgui::Condition::Always)
            .size(size, imgui::Condition::Always)
            .flags(control_flags() - WindowFlags::NO_BACKGROUND)
            .build(|| {
                self.navigation
                    .buttons(ui, icons, &request.commands, request.extent, actions)
            });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    const BOTH: ViewControlVisibility = ViewControlVisibility {
        cube: true,
        navigation: true,
    };
    const SHORT: WorkspaceArea = WorkspaceArea {
        left: 142.,
        top: 149.,
        right: 498.,
        bottom: 431.,
    };

    #[test]
    fn short_viewports_put_the_full_bar_inside_the_cube_corner() {
        let right = ViewControlPlacement::new(SHORT, ScreenCorner::TopRight, 120., BOTH);
        assert_eq!(right.cube, [370., 157.]);
        assert_eq!(right.bar, [[332., 157.], [30., 176.]]);
        let left = ViewControlPlacement::new(SHORT, ScreenCorner::BottomLeft, 120., BOTH);
        assert_eq!(left.cube, [150., 303.]);
        assert_eq!(left.bar, [[278., 247.], [30., 176.]]);
    }

    #[test]
    fn tall_viewports_keep_the_bar_centered_below_the_cube() {
        let area = WorkspaceArea {
            bottom: 850.,
            ..SHORT
        };
        let layout = ViewControlPlacement::new(area, ScreenCorner::TopRight, 120., BOTH);
        assert_eq!(layout.cube, [370., 157.]);
        assert_eq!(layout.bar, [[415., 285.], [30., 176.]]);
    }
}
